package order

import (
	"context"
	"errors"

	"github.com/radiophysiker/d56/internal/domain/user"
)

var (
	ErrUserIDEmpty       = errors.New("user ID cannot be empty")
	ErrOrderNumberEmpty  = errors.New("order number cannot be empty")
	ErrInvalidFormat     = errors.New("invalid order number format")
	ErrOrderAlreadyTaken = errors.New("order already uploaded by another user")
	ErrOrderNotFound     = errors.New("order not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
)

type Repository interface {
	Save(ctx context.Context, order *Order) error
	FindByNumber(ctx context.Context, number string) (*Order, error)
	FindByUserID(ctx context.Context, userID user.UserID) ([]*Order, error)
	FindPendingOrders(ctx context.Context) ([]*Order, error)
	Update(ctx context.Context, order *Order) error
}
