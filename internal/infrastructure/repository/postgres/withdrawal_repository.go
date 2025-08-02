package postgres

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/domain/withdrawal"
)

type WithdrawalRepository struct {
	db *sqlx.DB
}

func NewWithdrawalRepository(db *sqlx.DB) *WithdrawalRepository {
	return &WithdrawalRepository{db: db}
}

func (r *WithdrawalRepository) Save(ctx context.Context, w *withdrawal.Withdrawal) error {
	query := `
		INSERT INTO withdrawals (user_id, order_number, amount, processed_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id withdrawal.WithdrawalID
	return r.db.QueryRowContext(ctx, query,
		w.UserID(), w.OrderNumber(), w.Amount(), w.ProcessedAt()).Scan(&id)
}

func (r *WithdrawalRepository) FindByUserID(ctx context.Context, userID user.UserID) ([]*withdrawal.Withdrawal, error) {
	query := `
		SELECT id, user_id, order_number, amount, processed_at 
		FROM withdrawals 
		WHERE user_id = $1 
		ORDER BY processed_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var withdrawals []*withdrawal.Withdrawal
	for rows.Next() {
		var id withdrawal.WithdrawalID
		var uid user.UserID
		var orderNumber string
		var amount float64
		var processedAt time.Time

		err := rows.Scan(&id, &uid, &orderNumber, &amount, &processedAt)
		if err != nil {
			return nil, err
		}

		withdrawals = append(withdrawals, withdrawal.NewWithdrawalFromRepository(
			id, uid, orderNumber, amount, processedAt,
		))
	}

	return withdrawals, rows.Err()
}
