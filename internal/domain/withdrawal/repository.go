package withdrawal

import (
	"context"
	"errors"

	"github.com/radiophysiker/d56/internal/domain/user"
)

var (
	ErrUserIDEmpty          = errors.New("user ID cannot be empty")
	ErrAmountMustBePositive = errors.New("amount must be positive")
	ErrInsufficientFunds    = errors.New("insufficient funds")
)

type Repository interface {
	Save(ctx context.Context, withdrawal *Withdrawal) error
	FindByUserID(ctx context.Context, userID user.UserID) ([]*Withdrawal, error)
}
