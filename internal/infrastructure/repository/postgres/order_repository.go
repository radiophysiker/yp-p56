package postgres

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
)

type OrderRepository struct {
	db *sqlx.DB
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	query := `
		INSERT INTO orders (user_id, number, status, accrual, uploaded_at, processed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var id order.ID
	return r.db.QueryRowContext(ctx, query,
		o.UserID(), o.Number(), string(o.Status()), o.Accrual(),
		o.UploadedAt(), o.ProcessedAt()).Scan(&id)
}

func (r *OrderRepository) FindByNumber(ctx context.Context, number string) (*order.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, processed_at 
		FROM orders 
		WHERE number = $1
	`

	var id order.ID
	var userID user.UserID
	var orderNumber string
	var status string
	var accrual *float64
	var uploadedAt, processedAt sql.NullTime

	err := r.db.QueryRowContext(ctx, query, number).Scan(
		&id, &userID, &orderNumber, &status, &accrual, &uploadedAt, &processedAt,
	)
	if err != nil {
		return nil, err
	}

	var processedAtPtr *time.Time
	if processedAt.Valid {
		processedAtPtr = &processedAt.Time
	}

	return order.NewOrderFromRepository(
		id, userID, orderNumber, order.Status(status),
		accrual, uploadedAt.Time, processedAtPtr,
	), nil
}

func (r *OrderRepository) FindByUserID(ctx context.Context, userID user.UserID) ([]*order.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, processed_at 
		FROM orders 
		WHERE user_id = $1 
		ORDER BY uploaded_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*order.Order
	for rows.Next() {
		var id order.ID
		var uid user.UserID
		var number string
		var status string
		var accrual *float64
		var uploadedAt, processedAt sql.NullTime

		err := rows.Scan(&id, &uid, &number, &status, &accrual, &uploadedAt, &processedAt)
		if err != nil {
			return nil, err
		}

		var processedAtPtr *time.Time
		if processedAt.Valid {
			processedAtPtr = &processedAt.Time
		}

		orders = append(orders, order.NewOrderFromRepository(
			id, uid, number, order.Status(status),
			accrual, uploadedAt.Time, processedAtPtr,
		))
	}

	return orders, rows.Err()
}

func (r *OrderRepository) FindPendingOrders(ctx context.Context) ([]*order.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, processed_at 
		FROM orders 
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []*order.Order
	for rows.Next() {
		var id order.ID
		var userID user.UserID
		var number string
		var status string
		var accrual *float64
		var uploadedAt, processedAt sql.NullTime

		err := rows.Scan(&id, &userID, &number, &status, &accrual, &uploadedAt, &processedAt)
		if err != nil {
			return nil, err
		}

		var processedAtPtr *time.Time
		if processedAt.Valid {
			processedAtPtr = &processedAt.Time
		}

		orders = append(orders, order.NewOrderFromRepository(
			id, userID, number, order.Status(status),
			accrual, uploadedAt.Time, processedAtPtr,
		))
	}

	return orders, rows.Err()
}

func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	query := `
		UPDATE orders 
		SET status = $1, accrual = $2, processed_at = $3 
		WHERE id = $4
	`
	_, err := r.db.ExecContext(ctx, query,
		string(o.Status()), o.Accrual(), o.ProcessedAt(), o.ID())
	return err
}
