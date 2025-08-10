package service

import (
	"context"
	"testing"

	"github.com/radiophysiker/d56/internal/domain"
	domainMocks "github.com/radiophysiker/d56/internal/mocks/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestExecuteInTransaction_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	err := ExecuteInTransaction(ctx, factory, func(ctx context.Context, _ domain.UnitOfWork) error {
		return nil
	})
	require.NoError(t, err)
}

func TestExecuteInTransaction_FactoryError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	factory.EXPECT().Create(mock.Anything).Return(nil, assert.AnError).Once()

	err := ExecuteInTransaction(ctx, factory, func(ctx context.Context, _ domain.UnitOfWork) error { return nil })
	require.Error(t, err)
}

func TestExecuteInTransaction_BeginError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	err := ExecuteInTransaction(ctx, factory, func(ctx context.Context, _ domain.UnitOfWork) error { return nil })
	require.Error(t, err)
}

func TestExecuteInTransaction_FuncError_Rollback(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	err := ExecuteInTransaction(ctx, factory, func(ctx context.Context, _ domain.UnitOfWork) error { return assert.AnError })
	require.Error(t, err)
}

func TestExecuteInTransaction_CommitError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	err := ExecuteInTransaction(ctx, factory, func(ctx context.Context, _ domain.UnitOfWork) error { return nil })
	require.Error(t, err)
}
