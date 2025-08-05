package service

import (
	"context"

	"github.com/radiophysiker/d56/internal/domain"
)

// ExecuteInTransaction executes function inside transaction with automatic
// Commit/Rollback based on returned error
func ExecuteInTransaction(ctx context.Context, factory domain.UnitOfWorkFactory, fn func(context.Context, domain.UnitOfWork) error) error {
	uow, err := factory.Create(ctx)
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback(ctx)
		}
		uow.Close()
	}()

	if err := uow.Begin(ctx); err != nil {
		return err
	}

	if err := fn(ctx, uow); err != nil {
		return err
	}

	if err := uow.Commit(ctx); err != nil {
		return err
	}

	committed = true
	return nil
}
