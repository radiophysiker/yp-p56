package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/radiophysiker/d56/internal/domain"
	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
)

type OrderService struct {
	uowFactory   domain.UnitOfWorkFactory
	stateMachine *order.StateMachine
}

func NewOrderService(uowFactory domain.UnitOfWorkFactory) *OrderService {
	return &OrderService{
		uowFactory:   uowFactory,
		stateMachine: order.NewStateMachine(),
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID user.UserID, number string) (isNew bool, err error) {
	var result bool

	err = ExecuteInTransaction(ctx, s.uowFactory, func(ctx context.Context, uow domain.UnitOfWork) error {
		existingOrder, err := uow.OrderRepository().FindByNumber(ctx, number)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return err
		}

		if existingOrder != nil {
			if existingOrder.UserID() == userID {
				result = false
				return nil
			}
			return order.ErrOrderAlreadyTaken
		}

		newOrder, err := order.New(userID, number)
		if err != nil {
			return err
		}

		if err := uow.OrderRepository().Save(ctx, newOrder); err != nil {
			return err
		}

		result = true
		return nil
	})

	return result, err
}

// GetUserOrders retrieves all orders for a specific user
func (s *OrderService) GetUserOrders(ctx context.Context, userID user.UserID) ([]*order.Order, error) {
	uow, err := s.uowFactory.Create(ctx)
	if err != nil {
		return nil, err
	}
	defer uow.Close()

	return uow.OrderRepository().FindByUserID(ctx, userID)
}

func (s *OrderService) GetPendingOrders(ctx context.Context) ([]*order.Order, error) {
	uow, err := s.uowFactory.Create(ctx)
	if err != nil {
		return nil, err
	}
	defer uow.Close()

	return uow.OrderRepository().FindPendingOrders(ctx)
}
