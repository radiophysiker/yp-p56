package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
)

type Executer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type OrderRepository struct {
	executer Executer
}

func NewOrderRepository(db *sqlx.DB) *OrderRepository {
	return &OrderRepository{executer: db}
}

func NewOrderRepositoryWithTx(tx *sqlx.Tx) *OrderRepository {
	return &OrderRepository{executer: tx}
}

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	query := `
		INSERT INTO orders (user_id, number, status, accrual, uploaded_at, processed_at, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`
	var id order.ID
	return r.executer.QueryRowContext(ctx, query,
		o.UserID(), o.Number(), string(o.Status()), o.Accrual(),
		o.UploadedAt(), o.ProcessedAt(), o.Version()).Scan(&id)
}

func (r *OrderRepository) FindByNumber(ctx context.Context, number string) (*order.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, processed_at, version 
		FROM orders 
		WHERE number = $1
	`

	var id order.ID
	var userID user.UserID
	var orderNumber string
	var status string
	var accrual *float64
	var uploadedAt, processedAt sql.NullTime
	var version int64

	err := r.executer.QueryRowContext(ctx, query, number).Scan(
		&id, &userID, &orderNumber, &status, &accrual, &uploadedAt, &processedAt, &version,
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
		accrual, uploadedAt.Time, processedAtPtr, version,
	), nil
}

func (r *OrderRepository) FindByUserID(ctx context.Context, userID user.UserID) ([]*order.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, processed_at, version 
		FROM orders 
		WHERE user_id = $1 
		ORDER BY uploaded_at DESC
	`

	rows, err := r.executer.QueryContext(ctx, query, userID)
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
		var version int64

		err := rows.Scan(&id, &uid, &number, &status, &accrual, &uploadedAt, &processedAt, &version)
		if err != nil {
			return nil, err
		}

		var processedAtPtr *time.Time
		if processedAt.Valid {
			processedAtPtr = &processedAt.Time
		}

		orders = append(orders, order.NewOrderFromRepository(
			id, uid, number, order.Status(status),
			accrual, uploadedAt.Time, processedAtPtr, version,
		))
	}

	return orders, rows.Err()
}

func (r *OrderRepository) FindPendingOrders(ctx context.Context) ([]*order.Order, error) {
	query := `
		SELECT id, user_id, number, status, accrual, uploaded_at, processed_at, version 
		FROM orders 
		WHERE status IN ('NEW', 'PROCESSING')
		ORDER BY uploaded_at ASC
	`

	rows, err := r.executer.QueryContext(ctx, query)
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
		var version int64

		err := rows.Scan(&id, &userID, &number, &status, &accrual, &uploadedAt, &processedAt, &version)
		if err != nil {
			return nil, err
		}

		var processedAtPtr *time.Time
		if processedAt.Valid {
			processedAtPtr = &processedAt.Time
		}

		orders = append(orders, order.NewOrderFromRepository(
			id, userID, number, order.Status(status),
			accrual, uploadedAt.Time, processedAtPtr, version,
		))
	}

	return orders, rows.Err()
}

func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	query := `
		UPDATE orders 
		SET status = $1, accrual = $2, processed_at = $3, version = $4 
		WHERE id = $5 AND version = $6
	`

	oldVersion := o.Version() - 1
	result, err := r.executer.ExecContext(ctx, query,
		string(o.Status()), o.Accrual(), o.ProcessedAt(), o.Version(), o.ID(), oldVersion)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return &OptimisticLockError{
			EntityID:       int64(o.ID()),
			EntityType:     "order",
			CurrentVersion: o.Version(),
		}
	}

	return nil
}

func (r *OrderRepository) UpdateWithOptimisticLock(ctx context.Context, o *order.Order, expectedVersion int64) error {
	query := `
		UPDATE orders 
		SET status = $1, accrual = $2, processed_at = $3, version = version + 1 
		WHERE id = $4 AND version = $5
	`

	result, err := r.executer.ExecContext(ctx, query,
		string(o.Status()), o.Accrual(), o.ProcessedAt(), o.ID(), expectedVersion)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return &OptimisticLockError{
			EntityID:       int64(o.ID()),
			EntityType:     "order",
			CurrentVersion: expectedVersion + 1,
		}
	}

	o.IncrementVersion()
	return nil
}

type OptimisticLockError struct {
	EntityID       int64
	EntityType     string
	CurrentVersion int64
}

func (e *OptimisticLockError) Error() string {
	return fmt.Sprintf("optimistic lock failed for %s with id %d, current version %d",
		e.EntityType, e.EntityID, e.CurrentVersion)
}

func IsOptimisticLockError(err error) bool {
	_, ok := err.(*OptimisticLockError)
	return ok
}
