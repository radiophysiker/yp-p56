package service

import (
	"context"
	"database/sql"
	"testing"

	"github.com/radiophysiker/d56/internal/domain/order"
	domainMocks "github.com/radiophysiker/d56/internal/mocks/domain"
	orderMocks "github.com/radiophysiker/d56/internal/mocks/domain/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func setupMocksWithoutExpectations(t *testing.T) (*domainMocks.MockUnitOfWorkFactory, *domainMocks.MockUnitOfWork, *orderMocks.MockRepository) {
	t.Helper()
	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)
	return factory, uow, orderRepo
}

func setupMocks(t *testing.T) (*domainMocks.MockUnitOfWorkFactory, *domainMocks.MockUnitOfWork, *orderMocks.MockRepository) {
	t.Helper()
	factory, uow, orderRepo := setupMocksWithoutExpectations(t)
	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	return factory, uow, orderRepo
}

func expectSuccessfulTransaction(uow *domainMocks.MockUnitOfWork) {
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()
}

func TestOrderService_CreateOrder_New_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	expectSuccessfulTransaction(uow)
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), testOrderNumber).
		Return((*order.Order)(nil), sql.ErrNoRows).Once()
	expectedUserID := randomUserID()
	orderRepo.EXPECT().Save(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(matchOrder(expectedUserID, testOrderNumber, order.StatusNew))).Return(nil).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, expectedUserID, testOrderNumber)
	require.NoError(t, err)
	assert.True(t, isNew)
}

func TestOrderService_CreateOrder_ExistingSameUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	uid := randomUserID()
	existing, _ := order.New(uid, testOrderNumber)

	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), testOrderNumber).Return(existing, nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, uid, testOrderNumber)
	require.NoError(t, err)
	assert.False(t, isNew)
	orderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestOrderService_CreateOrder_ExistingOtherUser(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	uid := randomUserID()
	other := randomUserID()
	existing, _ := order.New(other, testOrderNumber)

	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), testOrderNumber).Return(existing, nil).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, uid, testOrderNumber)
	require.Error(t, err)
	assert.False(t, isNew)
	assert.ErrorIs(t, err, order.ErrOrderAlreadyTaken)
	orderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestOrderService_CreateOrder_FindByNumber_DBError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), testOrderNumber).Return((*order.Order)(nil), assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, randomUserID(), testOrderNumber)
	require.Error(t, err)
	assert.False(t, isNew)
	orderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestOrderService_CreateOrder_SaveError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	expectedUserID := randomUserID()
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), testOrderNumber).Return((*order.Order)(nil), sql.ErrNoRows).Once()
	orderRepo.EXPECT().Save(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(matchOrder(expectedUserID, testOrderNumber, order.StatusNew))).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, expectedUserID, testOrderNumber)
	require.Error(t, err)
	assert.False(t, isNew)
}

func TestOrderService_CreateOrder_InvalidNumbers(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name   string
		number string
	}{
		{name: "empty", number: ""},
		{name: "too short", number: "123"},
		{name: "invalid format", number: "invalid"},
		{name: "letters", number: "abc123def"},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := context.Background()

			factory, uow, orderRepo := setupMocks(t)
			uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
			uow.EXPECT().OrderRepository().Return(orderRepo).Once()
			orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), tc.number).Return((*order.Order)(nil), sql.ErrNoRows).Once()
			uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
			uow.EXPECT().Close().Return(nil).Once()

			svc := NewOrderService(factory)

			isNew, err := svc.CreateOrder(ctx, randomUserID(), tc.number)
			require.Error(t, err)
			assert.False(t, isNew)
			if tc.number == "" {
				assert.ErrorIs(t, err, order.ErrOrderNumberEmpty)
			} else {
				assert.ErrorIs(t, err, order.ErrInvalidFormat)
			}
			orderRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
		})
	}
}

func TestOrderService_CreateOrder_BeginError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, randomUserID(), testOrderNumber)
	require.Error(t, err)
	assert.False(t, isNew)
}

func TestOrderService_CreateOrder_FactoryError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	factory.EXPECT().Create(mock.MatchedBy(func(c context.Context) bool { return c == ctx })).Return(nil, assert.AnError).Once()

	svc := NewOrderService(factory)

	isNew, err := svc.CreateOrder(ctx, randomUserID(), testOrderNumber)
	require.Error(t, err)
	assert.False(t, isNew)
}

func TestOrderService_GetUserOrders_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	uid := randomUserID()
	ord, _ := order.New(uid, testOrderNumber)

	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByUserID(mock.MatchedBy(func(c context.Context) bool { return c == ctx }), uid).Return([]*order.Order{ord}, nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	orders, err := svc.GetUserOrders(ctx, uid)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	uow.AssertNotCalled(t, "Begin")
	uow.AssertNotCalled(t, "Commit")
	uow.AssertNotCalled(t, "Rollback")
}

func TestOrderService_GetUserOrders_RepoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocksWithoutExpectations(t)
	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()

	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindByUserID(mock.MatchedBy(matchContext(ctx)), mock.Anything).Return(([]*order.Order)(nil), assert.AnError).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	orders, err := svc.GetUserOrders(ctx, randomUserID())
	require.Error(t, err)
	assert.Nil(t, orders)

	uow.AssertNotCalled(t, "Begin")
	uow.AssertNotCalled(t, "Commit")
	uow.AssertNotCalled(t, "Rollback")
}

func TestOrderService_GetUserOrders_FactoryError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	factory.EXPECT().Create(mock.Anything).Return(nil, assert.AnError).Once()

	svc := NewOrderService(factory)

	orders, err := svc.GetUserOrders(ctx, randomUserID())
	require.Error(t, err)
	assert.Nil(t, orders)
}

func TestOrderService_GetPendingOrders_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocks(t)

	uid := randomUserID()
	ord, _ := order.New(uid, testOrderNumber)

	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindPendingOrders(mock.MatchedBy(func(c context.Context) bool { return c == ctx })).Return([]*order.Order{ord}, nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	orders, err := svc.GetPendingOrders(ctx)
	require.NoError(t, err)
	require.Len(t, orders, 1)
	uow.AssertNotCalled(t, "Begin")
	uow.AssertNotCalled(t, "Commit")
	uow.AssertNotCalled(t, "Rollback")
}

func TestOrderService_GetPendingOrders_RepoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory, uow, orderRepo := setupMocksWithoutExpectations(t)
	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()

	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindPendingOrders(mock.MatchedBy(matchContext(ctx))).Return(([]*order.Order)(nil), assert.AnError).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewOrderService(factory)

	orders, err := svc.GetPendingOrders(ctx)
	require.Error(t, err)
	assert.Nil(t, orders)

	uow.AssertNotCalled(t, "Begin")
	uow.AssertNotCalled(t, "Commit")
	uow.AssertNotCalled(t, "Rollback")
}

func TestOrderService_GetPendingOrders_FactoryError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	factory.EXPECT().Create(mock.Anything).Return(nil, assert.AnError).Once()

	svc := NewOrderService(factory)

	orders, err := svc.GetPendingOrders(ctx)
	require.Error(t, err)
	assert.Nil(t, orders)
}
