package service

import (
	"context"
	"testing"

	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/domain/withdrawal"
	domainMocks "github.com/radiophysiker/d56/internal/mocks/domain"
	userMocks "github.com/radiophysiker/d56/internal/mocks/domain/user"
	withdrawalMocks "github.com/radiophysiker/d56/internal/mocks/domain/withdrawal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWithdrawalService_WithdrawFunds_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	userRepo := userMocks.NewMockRepository(t)
	wdRepo := withdrawalMocks.NewMockRepository(t)

	uid := randomUserID()

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	uow.EXPECT().WithdrawalRepository().Return(wdRepo).Once()

	u, _ := user.New("login", "hash")
	u.UpdateBalance(100, 0)
	userRepo.EXPECT().FindByID(mock.MatchedBy(matchContext(ctx)), uid).Return(u, nil).Once()
	userRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(uu *user.User) bool {
		b := uu.Balance()
		return b.Current == 60 && b.Withdrawn == 40
	})).Return(nil).Once()
	wdRepo.EXPECT().Save(mock.MatchedBy(matchContext(ctx)), mock.MatchedBy(func(w *withdrawal.Withdrawal) bool {
		return w.UserID() == uid && w.OrderNumber() == testOrderNumber && w.Amount() == 40
	})).Return(nil).Once()

	uow.EXPECT().Commit(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)

	w, err := svc.WithdrawFunds(ctx, uid, testOrderNumber, 40)
	require.NoError(t, err)
	require.NotNil(t, w)
}

func TestWithdrawalService_GetUserWithdrawals_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	wdRepo := withdrawalMocks.NewMockRepository(t)

	uid := randomUserID()
	w, _ := withdrawal.New(uid, testOrderNumber, 10)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().WithdrawalRepository().Return(wdRepo).Once()
	wdRepo.EXPECT().FindByUserID(mock.MatchedBy(matchContext(ctx)), uid).Return([]*withdrawal.Withdrawal{w}, nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)

	ws, err := svc.GetUserWithdrawals(ctx, uid)
	require.NoError(t, err)
	require.Len(t, ws, 1)
}

func TestWithdrawalService_GetUserWithdrawals_FactoryError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	factory.EXPECT().Create(mock.Anything).Return(nil, assert.AnError).Once()

	svc := NewWithdrawalService(factory)

	ws, err := svc.GetUserWithdrawals(ctx, randomUserID())
	require.Error(t, err)
	assert.Nil(t, ws)
}

func TestWithdrawalService_GetUserWithdrawals_RepoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	wdRepo := withdrawalMocks.NewMockRepository(t)

	uid := randomUserID()

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().WithdrawalRepository().Return(wdRepo).Once()
	wdRepo.EXPECT().FindByUserID(mock.MatchedBy(matchContext(ctx)), uid).Return(([]*withdrawal.Withdrawal)(nil), assert.AnError).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)

	ws, err := svc.GetUserWithdrawals(ctx, uid)
	require.Error(t, err)
	assert.Nil(t, ws)

	uow.AssertNotCalled(t, "Begin")
	uow.AssertNotCalled(t, "Commit")
	uow.AssertNotCalled(t, "Rollback")
}

func TestWithdrawalService_WithdrawFunds_InsufficientFunds(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	userRepo := userMocks.NewMockRepository(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()

	u, _ := user.New("login", "hash")
	u.UpdateBalance(10, 0)
	userRepo.EXPECT().FindByID(mock.MatchedBy(matchContext(ctx)), mock.Anything).Return(u, nil).Once()

	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)

	w, err := svc.WithdrawFunds(ctx, randomUserID(), testOrderNumber, 40)
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrInsufficientFunds)
	assert.Nil(t, w)

	userRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestWithdrawalService_WithdrawFunds_FindByID_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	userRepo := userMocks.NewMockRepository(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	userRepo.EXPECT().FindByID(mock.MatchedBy(matchContext(ctx)), mock.Anything).Return((*user.User)(nil), assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)
	w, err := svc.WithdrawFunds(ctx, randomUserID(), testOrderNumber, 10)
	require.Error(t, err)
	assert.Nil(t, w)
}

func TestWithdrawalService_WithdrawFunds_Update_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	userRepo := userMocks.NewMockRepository(t)
	wdRepo := withdrawalMocks.NewMockRepository(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	u, _ := user.New("login", "hash")
	u.UpdateBalance(100, 0)
	userRepo.EXPECT().FindByID(mock.MatchedBy(matchContext(ctx)), mock.Anything).Return(u, nil).Once()
	userRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.AnythingOfType("*user.User")).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)
	w, err := svc.WithdrawFunds(ctx, randomUserID(), testOrderNumber, 10)
	require.Error(t, err)
	assert.Nil(t, w)
	wdRepo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestWithdrawalService_WithdrawFunds_Save_Error(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	factory := domainMocks.NewMockUnitOfWorkFactory(t)
	uow := domainMocks.NewMockUnitOfWork(t)
	userRepo := userMocks.NewMockRepository(t)
	wdRepo := withdrawalMocks.NewMockRepository(t)

	factory.EXPECT().Create(mock.Anything).Return(uow, nil).Once()
	uow.EXPECT().Begin(mock.Anything).Return(nil).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	uow.EXPECT().UserRepository().Return(userRepo).Once()
	uow.EXPECT().WithdrawalRepository().Return(wdRepo).Once()

	u, _ := user.New("login", "hash")
	u.UpdateBalance(100, 0)
	userRepo.EXPECT().FindByID(mock.MatchedBy(matchContext(ctx)), mock.Anything).Return(u, nil).Once()
	userRepo.EXPECT().Update(mock.MatchedBy(matchContext(ctx)), mock.AnythingOfType("*user.User")).Return(nil).Once()
	wdRepo.EXPECT().Save(mock.MatchedBy(matchContext(ctx)), mock.AnythingOfType("*withdrawal.Withdrawal")).Return(assert.AnError).Once()
	uow.EXPECT().Rollback(mock.Anything).Return(nil).Once()
	uow.EXPECT().Close().Return(nil).Once()

	svc := NewWithdrawalService(factory)
	w, err := svc.WithdrawFunds(ctx, randomUserID(), testOrderNumber, 10)
	require.Error(t, err)
	assert.Nil(t, w)
}
