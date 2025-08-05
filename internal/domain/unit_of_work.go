package domain

import (
	"context"

	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/domain/withdrawal"
)

// UnitOfWork represents a unit of work pattern for managing transactions
type UnitOfWork interface {
	// Repositories
	UserRepository() user.Repository
	OrderRepository() order.Repository
	WithdrawalRepository() withdrawal.Repository

	// Transaction management
	Begin(ctx context.Context) error
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
	Close() error
}

// UnitOfWorkFactory creates new UnitOfWork instances
type UnitOfWorkFactory interface {
	Create(ctx context.Context) (UnitOfWork, error)
}
