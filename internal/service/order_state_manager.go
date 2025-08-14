package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/radiophysiker/d56/internal/domain"
	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/infrastructure/repository/postgres"
)

// OrderStateManager manages safe order state transitions using a State Machine
type OrderStateManager struct {
	uowFactory   domain.UnitOfWorkFactory
	stateMachine *order.StateMachine
}

// NewOrderStateManager creates a new order state manager
func NewOrderStateManager(uowFactory domain.UnitOfWorkFactory) *OrderStateManager {
	return &OrderStateManager{
		uowFactory:   uowFactory,
		stateMachine: order.NewStateMachine(),
	}
}

// TransitionOrderStatus performs a safe state transition with validation and optimistic locking
func (m *OrderStateManager) TransitionOrderStatus(ctx context.Context, orderNumber string, newStatus order.Status, accrual *float64) error {
	const maxRetries = 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		err := m.attemptTransition(ctx, orderNumber, newStatus, accrual)
		if err == nil {
			return nil
		}

		if postgres.IsOptimisticLockError(err) {
			continue
		}

		if order.IsInvalidTransitionError(err) {
			return err
		}

		return err
	}

	return errors.New("failed to update order status after maximum retries due to concurrent modifications")
}

// attemptTransition performs a single transition attempt
func (m *OrderStateManager) attemptTransition(ctx context.Context, orderNumber string, newStatus order.Status, accrual *float64) error {
	return ExecuteInTransaction(ctx, m.uowFactory, func(ctx context.Context, uow domain.UnitOfWork) error {
		orderEntity, err := uow.OrderRepository().FindByNumber(ctx, orderNumber)
		if err != nil {
			return fmt.Errorf("failed to find order: %w", err)
		}

		currentVersion := orderEntity.Version()

		if !m.stateMachine.CanTransition(orderEntity.Status(), newStatus) {
			return &order.InvalidTransitionError{
				From: orderEntity.Status(),
				To:   newStatus,
			}
		}

		if err := orderEntity.TransitionTo(newStatus, m.stateMachine); err != nil {
			return fmt.Errorf("failed to transition order state: %w", err)
		}

		if newStatus == order.StatusProcessed {
			if accrual != nil && *accrual > 0 {
				orderEntity.SetAccrual(*accrual)

				user, err := uow.UserRepository().FindByID(ctx, orderEntity.UserID())
				if err != nil {
					return fmt.Errorf("failed to find user: %w", err)
				}

				user.AddBalance(*accrual)
				if err := uow.UserRepository().Update(ctx, user); err != nil {
					return fmt.Errorf("failed to update user balance: %w", err)
				}
			} else {
				orderEntity.SetAccrual(0.0)
			}
		} else if newStatus == order.StatusInvalid {
			orderEntity.SetAccrual(0.0)
		} else if accrual != nil {
			orderEntity.SetAccrual(*accrual)
		}

		if postgresRepo, ok := uow.OrderRepository().(*postgres.OrderRepository); ok {
			return postgresRepo.UpdateWithOptimisticLock(ctx, orderEntity, currentVersion)
		}

		return uow.OrderRepository().Update(ctx, orderEntity)
	})
}

// CanTransitionTo checks if transition to new state is possible without performing the transition
func (m *OrderStateManager) CanTransitionTo(currentStatus order.Status, newStatus order.Status) bool {
	return m.stateMachine.CanTransition(currentStatus, newStatus)
}

// GetValidTransitions returns list of possible states for transition from current state
func (m *OrderStateManager) GetValidTransitions(currentStatus order.Status) []order.Status {
	return m.stateMachine.GetValidTransitions(currentStatus)
}

// IsFinalState checks if state is final
func (m *OrderStateManager) IsFinalState(status order.Status) bool {
	return m.stateMachine.IsFinalState(status)
}

// StartProcessing safely transitions order to PROCESSING state
func (m *OrderStateManager) StartProcessing(ctx context.Context, orderNumber string) error {
	return m.TransitionOrderStatus(ctx, orderNumber, order.StatusProcessing, nil)
}

// CompleteProcessing completes order processing with accrual
func (m *OrderStateManager) CompleteProcessing(ctx context.Context, orderNumber string, accrual float64) error {
	return m.TransitionOrderStatus(ctx, orderNumber, order.StatusProcessed, &accrual)
}

// MarkAsInvalid marks order as invalid
func (m *OrderStateManager) MarkAsInvalid(ctx context.Context, orderNumber string) error {
	return m.TransitionOrderStatus(ctx, orderNumber, order.StatusInvalid, nil)
}

// RetryProcessing returns order from PROCESSING to NEW for retry
func (m *OrderStateManager) RetryProcessing(ctx context.Context, orderNumber string) error {
	return m.TransitionOrderStatus(ctx, orderNumber, order.StatusNew, nil)
}
