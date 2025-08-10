package service

import (
	"context"
	"testing"
	"time"

	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/infrastructure/repository/postgres"
	domainMocks "github.com/radiophysiker/d56/internal/mocks/domain"
	orderMocks "github.com/radiophysiker/d56/internal/mocks/domain/order"
	userMocks "github.com/radiophysiker/d56/internal/mocks/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestOrderStateManager_Transition_Processed_WithAccrual_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)
	userRepo := userMocks.NewMockRepository(t)

	uID := randomUserID()
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()

	accrual := 15.0
	foundUser, _ := user.New("login", "hash")
	uow.EXPECT().UserRepository().Return(userRepo).Twice()
	userRepo.EXPECT().FindByID(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), uID).Return(foundUser, nil).Once()
	userRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.MatchedBy(func(u *user.User) bool {
		b := u.Balance()
		return b.Current >= accrual && b.Withdrawn == 0
	})).Return(nil).Once()

	orderRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.MatchedBy(func(o *order.Order) bool {
		return o.Status() == order.StatusProcessed && o.Accrual() != nil && *o.Accrual() == accrual
	})).Return(nil).Once()

	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.CompleteProcessing(ctx, ord.Number(), accrual)
	require.NoError(t, err)
}

func TestOrderStateManager_Transition_Processed_WithZeroAccrual_SetsZero_NoUserUpdate(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	uID := randomUserID()
	prevAccrual := 7.5
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusProcessing, &prevAccrual, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()

	zero := 0.0
	orderRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.MatchedBy(func(o *order.Order) bool {
		return o.Status() == order.StatusProcessed && o.Accrual() != nil && *o.Accrual() == zero
	})).Return(nil).Once()

	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.CompleteProcessing(ctx, ord.Number(), 0.0)
	require.NoError(t, err)
}

// Note: nil accrual for Processed is not used by processor; it passes 0.0 instead. Covered by the zero test above.

func TestOrderStateManager_Transition_Invalid_ClearsAccrual(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	uID := randomUserID()
	acc := 5.0
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusProcessing, &acc, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Twice()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()
	orderRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.MatchedBy(func(o *order.Order) bool {
		return o.Status() == order.StatusInvalid && (o.Accrual() == nil || *o.Accrual() == 0)
	})).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.MarkAsInvalid(ctx, ord.Number())
	require.NoError(t, err)
}

func TestOrderStateManager_Transition_InvalidTransition_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	uID := randomUserID()
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusNew, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.TransitionOrderStatus(ctx, ord.Number(), order.StatusProcessed, nil)
	require.Error(t, err)
	assert.True(t, order.IsInvalidTransitionError(err))
}

func TestOrderStateManager_Transition_RetryOnOptimisticLock(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)

	uow1 := domainMocks.NewMockUnitOfWork(t)
	orderRepo1 := orderMocks.NewMockRepository(t)
	uID := randomUserID()
	ord1 := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusNew, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow1, nil).Once()
	uow1.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow1.EXPECT().OrderRepository().Return(orderRepo1).Times(3)
	orderRepo1.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord1.Number()).Return(ord1, nil).Once()
	orderRepo1.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.AnythingOfType("*order.Order")).Return(&postgres.OptimisticLockError{}).Once()
	uow1.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow1.EXPECT().Close().Return(nil).Once()

	uow2 := domainMocks.NewMockUnitOfWork(t)
	orderRepo2 := orderMocks.NewMockRepository(t)
	ord2 := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusNew, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow2, nil).Once()
	uow2.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow2.EXPECT().OrderRepository().Return(orderRepo2).Times(3)
	orderRepo2.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord2.Number()).Return(ord2, nil).Once()
	orderRepo2.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.AnythingOfType("*order.Order")).Return(nil).Once()
	uow2.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow2.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.StartProcessing(ctx, ord1.Number())
	require.NoError(t, err)
}

func TestOrderStateManager_HelperMethods(t *testing.T) {
	m := NewOrderStateManager(nil)
	assert.True(t, m.CanTransitionTo(order.StatusNew, order.StatusProcessing))
	assert.False(t, m.CanTransitionTo(order.StatusNew, order.StatusNew))

	transitions := m.GetValidTransitions(order.StatusProcessing)
	assert.Subset(t, transitions, []order.Status{order.StatusProcessed, order.StatusInvalid, order.StatusNew})

	assert.False(t, m.IsFinalState(order.StatusProcessing))
	assert.True(t, m.IsFinalState(order.StatusProcessed))
}

func TestOrderStateManager_FindByNumber_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), testOrderNumber).Return((*order.Order)(nil), assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.StartProcessing(ctx, testOrderNumber)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find order")
}

func TestOrderStateManager_Processed_FindUser_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)
	userRepo := userMocks.NewMockRepository(t)

	uID := randomUserID()
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()

	accrual := 10.0
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().FindByID(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), uID).Return((*user.User)(nil), assert.AnError).Once()

	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.CompleteProcessing(ctx, ord.Number(), accrual)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to find user")
}

func TestOrderStateManager_Processed_UpdateUser_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)
	userRepo := userMocks.NewMockRepository(t)

	uID := randomUserID()
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()

	accrual := 12.0
	foundUser, _ := user.New("login", "hash")
	uow.EXPECT().UserRepository().Return(userRepo).Times(2)
	userRepo.EXPECT().FindByID(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), uID).Return(foundUser, nil).Once()
	userRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.AnythingOfType("*user.User")).Return(assert.AnError).Once()

	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.CompleteProcessing(ctx, ord.Number(), accrual)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update user balance")
}

func TestOrderStateManager_UpdateOrder_Error_NoOptimisticRetry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	uID := randomUserID()
	ord := order.NewOrderFromRepository(1, uID, testOrderNumber, order.StatusNew, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()
	orderRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.AnythingOfType("*order.Order")).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	m := NewOrderStateManager(factory)
	err := m.StartProcessing(ctx, ord.Number())
	require.Error(t, err)
	assert.ErrorIs(t, err, assert.AnError)
}

func TestOrderStateManager_Retries_Exhausted(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)

	uID := randomUserID()
	ordNum := testOrderNumber

	for i := 0; i < 3; i++ {
		uow := domainMocks.NewMockUnitOfWork(t)
		orderRepo := orderMocks.NewMockRepository(t)
		ord := order.NewOrderFromRepository(1, uID, ordNum, order.StatusNew, nil, time.Now(), nil, 1)

		factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
		uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
		uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
		orderRepo.EXPECT().FindByNumber(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), ord.Number()).Return(ord, nil).Once()
		orderRepo.EXPECT().Update(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), mock.AnythingOfType("*order.Order")).Return(&postgres.OptimisticLockError{}).Once()
		uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
		uow.EXPECT().Close().Return(nil).Once()
	}

	m := NewOrderStateManager(factory)
	err := m.StartProcessing(ctx, ordNum)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update order status after maximum retries")
}
