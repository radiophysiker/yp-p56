package database

import (
	"context"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/domain"
	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/domain/withdrawal"
	"github.com/radiophysiker/d56/internal/infrastructure/repository/postgres"
)

// UnitOfWork implements domain.UnitOfWork for PostgreSQL
type UnitOfWork struct {
	tx             *sqlx.Tx
	userRepo       user.Repository
	orderRepo      order.Repository
	withdrawalRepo withdrawal.Repository
}

// NewUnitOfWork creates a new UnitOfWork instance
func NewUnitOfWork(tx *sqlx.Tx) *UnitOfWork {
	return &UnitOfWork{
		tx:             tx,
		userRepo:       postgres.NewUserRepositoryWithTx(tx),
		orderRepo:      postgres.NewOrderRepositoryWithTx(tx),
		withdrawalRepo: postgres.NewWithdrawalRepositoryWithTx(tx),
	}
}

// UserRepository returns the user repository
func (uow *UnitOfWork) UserRepository() user.Repository {
	return uow.userRepo
}

// OrderRepository returns the order repository
func (uow *UnitOfWork) OrderRepository() order.Repository {
	return uow.orderRepo
}

// WithdrawalRepository returns the withdrawal repository
func (uow *UnitOfWork) WithdrawalRepository() withdrawal.Repository {
	return uow.withdrawalRepo
}

// Begin starts a new transaction
func (uow *UnitOfWork) Begin(ctx context.Context) error {
	return nil
}

// Commit commits the transaction
func (uow *UnitOfWork) Commit(ctx context.Context) error {
	return uow.tx.Commit()
}

// Rollback rolls back the transaction
func (uow *UnitOfWork) Rollback(ctx context.Context) error {
	return uow.tx.Rollback()
}

// Close closes the transaction
func (uow *UnitOfWork) Close() error {
	return uow.tx.Rollback()
}

// UnitOfWorkFactory creates new UnitOfWork instances
type UnitOfWorkFactory struct {
	db *sqlx.DB
}

// NewUnitOfWorkFactory creates a new UnitOfWorkFactory
func NewUnitOfWorkFactory(db *sqlx.DB) *UnitOfWorkFactory {
	return &UnitOfWorkFactory{db: db}
}

// Create creates a new UnitOfWork with a transaction
func (f *UnitOfWorkFactory) Create(ctx context.Context) (domain.UnitOfWork, error) {
	tx, err := f.db.BeginTxx(ctx, nil)
	if err != nil {
		zap.L().Error("Failed to begin transaction", zap.Error(err))
		return nil, err
	}

	return NewUnitOfWork(tx), nil
}
