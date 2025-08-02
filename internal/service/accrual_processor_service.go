package service

import (
	"context"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/infrastructure/accrual"
)

type AccrualProcessorService struct {
	accrualClient *accrual.Client
	orderService  *OrderService
	logger        *zap.Logger
}

func NewAccrualProcessorService(accrualClient *accrual.Client, orderService *OrderService, logger *zap.Logger) *AccrualProcessorService {
	return &AccrualProcessorService{
		accrualClient: accrualClient,
		orderService:  orderService,
		logger:        logger,
	}
}

func (s *AccrualProcessorService) Start(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Accrual processor service stopped")
			return
		case <-ticker.C:
			s.processOrders(ctx)
		}
	}
}

func (s *AccrualProcessorService) processOrders(ctx context.Context) {
	orders, err := s.orderService.GetPendingOrders(ctx)
	if err != nil {
		s.logger.Error("Failed to get pending orders", zap.Error(err))
		return
	}

	s.logger.Info("Processing orders", zap.Int("count", len(orders)))
	for _, order := range orders {
		s.logger.Info("Processing order", zap.String("order", order.Number()), zap.String("status", string(order.Status())))
		s.processOrder(ctx, order.Number())
	}
}

func (s *AccrualProcessorService) processOrder(ctx context.Context, orderNumber string) {
	s.logger.Info("Getting order info from accrual system", zap.String("order", orderNumber))
	accrualResp, err := s.accrualClient.GetOrderInfo(ctx, orderNumber)
	if err != nil {
		if strings.Contains(err.Error(), "rate limit exceeded") {
			s.logger.Warn("Rate limit exceeded, will retry later", zap.String("order", orderNumber))
			return
		}
		s.logger.Error("Failed to get order info from accrual system",
			zap.String("order", orderNumber), zap.Error(err))
		return
	}

	if accrualResp == nil {
		s.logger.Info("No response from accrual system, order not found", zap.String("order", orderNumber))
		return
	}

	s.logger.Info("Received response from accrual system",
		zap.String("order", orderNumber),
		zap.String("status", accrualResp.Status),
		zap.Any("accrual", accrualResp.Accrual))

	status := s.accrualClient.ConvertToOrderStatus(accrualResp.Status)

	err = s.orderService.UpdateOrderStatus(ctx, orderNumber, status, accrualResp.Accrual)
	if err != nil {
		s.logger.Error("Failed to update order status",
			zap.String("order", orderNumber), zap.Error(err))
		return
	}

	s.logger.Info("Updated order status",
		zap.String("order", orderNumber),
		zap.String("status", string(status)))
}
