package service

import (
	"context"
	"database/sql"
	"errors"

	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
)

type OrderService struct {
	orderRepo order.Repository
	userRepo  user.Repository
}

func NewOrderService(
	orderRepo order.Repository,
	userRepo user.Repository,
) *OrderService {
	return &OrderService{
		orderRepo: orderRepo,
		userRepo:  userRepo,
	}
}

func (s *OrderService) CreateOrder(ctx context.Context, userID user.UserID, number string) (isNew bool, err error) {
	existingOrder, err := s.orderRepo.FindByNumber(ctx, number)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}

	if existingOrder != nil {
		if existingOrder.UserID() == userID {
			return false, nil
		}
		return false, order.ErrOrderAlreadyTaken
	}

	newOrder, err := order.New(userID, number)
	if err != nil {
		return false, err
	}

	if err := s.orderRepo.Save(ctx, newOrder); err != nil {
		return false, err
	}

	return true, nil
}

// GetUserOrders retrieves all orders for a specific user
func (s *OrderService) GetUserOrders(ctx context.Context, userID user.UserID) ([]*order.Order, error) {
	return s.orderRepo.FindByUserID(ctx, userID)
}

// UpdateOrderStatus updates the status of an order and handles accrual if applicable
func (s *OrderService) UpdateOrderStatus(ctx context.Context, orderNumber string, status order.Status, accrual *float64) error {
	orderEntity, err := s.orderRepo.FindByNumber(ctx, orderNumber)
	if err != nil {
		return err
	}

	orderEntity.SetStatus(status)
	if accrual != nil && status == order.StatusProcessed {
		orderEntity.SetAccrual(*accrual)

		if *accrual > 0 {
			user, err := s.userRepo.FindByID(ctx, orderEntity.UserID())
			if err != nil {
				return err
			}

			user.AddBalance(*accrual)
			if err := s.userRepo.Update(ctx, user); err != nil {
				return err
			}
		}
	} else if status == order.StatusInvalid {
		// Clear accrual for invalid orders
		orderEntity.SetAccrual(0.0)
	}

	return s.orderRepo.Update(ctx, orderEntity)
}

func (s *OrderService) GetPendingOrders(ctx context.Context) ([]*order.Order, error) {
	return s.orderRepo.FindPendingOrders(ctx)
}
