package service

import (
	"context"

	"github.com/radiophysiker/d56/internal/domain"
	"github.com/radiophysiker/d56/internal/domain/user"
	"github.com/radiophysiker/d56/internal/domain/withdrawal"
)

type WithdrawalService struct {
	uowFactory domain.UnitOfWorkFactory
}

func NewWithdrawalService(uowFactory domain.UnitOfWorkFactory) *WithdrawalService {
	return &WithdrawalService{
		uowFactory: uowFactory,
	}
}

func (s *WithdrawalService) WithdrawFunds(ctx context.Context, userID user.UserID, orderNumber string, amount float64) (*withdrawal.Withdrawal, error) {
	var result *withdrawal.Withdrawal

	err := ExecuteInTransaction(ctx, s.uowFactory, func(ctx context.Context, uow domain.UnitOfWork) error {
		w, err := withdrawal.New(userID, orderNumber, amount)
		if err != nil {
			return err
		}

		user, err := uow.UserRepository().FindByID(ctx, userID)
		if err != nil {
			return err
		}

		if err := user.WithdrawBalance(amount); err != nil {
			return err
		}

		if err := uow.UserRepository().Update(ctx, user); err != nil {
			return err
		}

		if err := uow.WithdrawalRepository().Save(ctx, w); err != nil {
			return err
		}

		result = w
		return nil
	})

	return result, err
}

func (s *WithdrawalService) GetUserWithdrawals(ctx context.Context, userID user.UserID) ([]*withdrawal.Withdrawal, error) {
	uow, err := s.uowFactory.Create(ctx)
	if err != nil {
		return nil, err
	}
	defer uow.Close()

	return uow.WithdrawalRepository().FindByUserID(ctx, userID)
}
