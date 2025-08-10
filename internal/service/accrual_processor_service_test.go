package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/domain/order"
	userdomain "github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/infrastructure/accrual"
	domainMocks "github.com/radiophysiker/d56/internal/mocks/domain"
	orderMocks "github.com/radiophysiker/d56/internal/mocks/domain/order"
	userMocks "github.com/radiophysiker/d56/internal/mocks/domain/user"
	"github.com/stretchr/testify/mock"
)

func newAccrualProcessorWith(t *testing.T, baseURL string, factory *domainMocks.MockUnitOfWorkFactory) *AccrualProcessorService {
	t.Helper()
	client := accrual.NewClient(baseURL)
	orderSvc := NewOrderService(factory)
	stateMgr := NewOrderStateManager(factory)
	logger := zap.NewNop()
	return NewAccrualProcessorService(client, orderSvc, stateMgr, logger)
}

func TestAccrualProcessor_processOrders_Empty(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindPendingOrders(mock.MatchedBy(matchContext(ctx))).Return([]*order.Order{}, nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrders(ctx)
}

func TestAccrualProcessor_processOrders_NewOrder_StartsProcessingThenProcesses(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	uid := randomUserID()
	ord := order.NewOrderFromRepository(1, uid, testOrderNumber, order.StatusNew, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Once()
	orderRepo.EXPECT().FindPendingOrders(mock.MatchedBy(matchContext(ctx))).Return([]*order.Order{ord}, nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	// StartProcessing -> transaction transition to PROCESSING, then processOrder -> mark invalid (204)
	uow2 := domainMocks.NewMockUnitOfWork(t)
	orderRepo2 := orderMocks.NewMockRepository(t)
	factory.EXPECT().Create(mock.Anything).Return(uow2, nil).Once()
	uow2.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow2.EXPECT().OrderRepository().Return(orderRepo2).Times(3)
	orderRepo2.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ord.Number()).Return(ord, nil).Once()
	orderRepo2.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool { return o.Status() == order.StatusProcessing })).Return(nil).Once()
	uow2.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow2.EXPECT().Close().Return(nil).Once()

	uow3 := domainMocks.NewMockUnitOfWork(t)
	orderRepo3 := orderMocks.NewMockRepository(t)
	factory.EXPECT().Create(mock.Anything).Return(uow3, nil).Once()
	uow3.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow3.EXPECT().OrderRepository().Return(orderRepo3).Times(3)
	orderRepo3.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ord.Number()).Return(ord, nil).Once()
	orderRepo3.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool { return o.Status() == order.StatusInvalid })).Return(nil).Once()
	uow3.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow3.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrders(ctx)
}

func TestAccrualProcessor_processOrder_RateLimit_Retry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	ordNum := testOrderNumber
	uid := randomUserID()
	ord := order.NewOrderFromRepository(1, uid, ordNum, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ordNum).Return(ord, nil).Once()
	orderRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool {
		return o.Status() == order.StatusNew
	})).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrder(ctx, ordNum)
}

func TestAccrualProcessor_processOrder_Error_NotRateLimit_Retry(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	ordNum := testOrderNumber
	uid := randomUserID()
	ord := order.NewOrderFromRepository(1, uid, ordNum, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ordNum).Return(ord, nil).Once()
	orderRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool { return o.Status() == order.StatusNew })).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrder(ctx, ordNum)
}

func TestAccrualProcessor_processOrder_StillProcessing_NoTransition(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resp := map[string]any{
		"order":  testOrderNumber,
		"status": "PROCESSING",
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	// No expectations: processOrder should not touch repositories for PROCESSING

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrder(ctx, testOrderNumber)
}

func TestAccrualProcessor_processOrder_DefaultStatus_TransitionWithAccrualPassthrough(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	resp := map[string]any{
		"order":   testOrderNumber,
		"status":  "REGISTERED",
		"accrual": 3.3,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	ordNum := testOrderNumber
	uid := randomUserID()
	ord := order.NewOrderFromRepository(1, uid, ordNum, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ordNum).Return(ord, nil).Once()
	orderRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool {
		return o.Status() == order.StatusNew && o.Accrual() != nil && *o.Accrual() == 3.3
	})).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrder(ctx, ordNum)
}

func TestAccrualProcessor_processOrder_NoContent_Invalid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)

	ordNum := testOrderNumber
	uid := randomUserID()
	ord := order.NewOrderFromRepository(1, uid, ordNum, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ordNum).Return(ord, nil).Once()
	orderRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool {
		return o.Status() == order.StatusInvalid
	})).Return(nil).Once()
	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrder(ctx, ordNum)
}

func TestAccrualProcessor_processOrder_Processed_WithAccrual(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	accrualValue := 12.5
	resp := map[string]any{
		"order":   testOrderNumber,
		"status":  "PROCESSED",
		"accrual": accrualValue,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	orderRepo := orderMocks.NewMockRepository(t)
	userRepo := userMocks.NewMockRepository(t)

	ordNum := testOrderNumber
	uid := randomUserID()
	ord := order.NewOrderFromRepository(1, uid, ordNum, order.StatusProcessing, nil, time.Now(), nil, 1)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().OrderRepository().Return(orderRepo).Times(3)
	orderRepo.EXPECT().FindByNumber(mock.MatchedBy(matchContext(ctx)), ordNum).Return(ord, nil).Once()

	uow.EXPECT().UserRepository().Return(userRepo).Twice()
	foundUser, _ := userdomain.New("login", "hash")
	userRepo.EXPECT().FindByID(mock.MatchedBy(matchContext(ctx)), uid).Return(foundUser, nil).Once()
	userRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(uu *userdomain.User) bool {
		b := uu.Balance()
		return b.Current == accrualValue && b.Withdrawn == 0
	})).Return(nil).Once()

	orderRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(o *order.Order) bool {
		if o.Status() != order.StatusProcessed {
			return false
		}
		if o.Accrual() == nil {
			return false
		}
		return *o.Accrual() == accrualValue
	})).Return(nil).Once()

	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := newAccrualProcessorWith(t, srv.URL, factory)
	svc.processOrder(ctx, ordNum)
}
