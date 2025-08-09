package withdrawal

import (
	"time"

	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/radiophysiker/d56/internal/domain/user"
)

type WithdrawalID int64

type Withdrawal struct {
	id          WithdrawalID
	userID      user.UserID
	orderNumber string
	amount      float64
	processedAt time.Time
}

func New(userID user.UserID, orderNumber string, amount float64) (*Withdrawal, error) {
	if userID == (user.UserID{}) {
		return nil, ErrUserIDEmpty
	}

	if orderNumber == "" {
		return nil, order.ErrOrderNumberEmpty
	}

	if !order.IsValidOrderNumber(orderNumber) {
		return nil, order.ErrInvalidFormat
	}

	if amount <= 0 {
		return nil, ErrAmountMustBePositive
	}

	return &Withdrawal{
		userID:      userID,
		orderNumber: orderNumber,
		amount:      amount,
		processedAt: time.Now(),
	}, nil
}

func (w *Withdrawal) ID() WithdrawalID {
	return w.id
}

func (w *Withdrawal) UserID() user.UserID {
	return w.userID
}

func (w *Withdrawal) OrderNumber() string {
	return w.orderNumber
}

func (w *Withdrawal) Amount() float64 {
	return w.amount
}

func (w *Withdrawal) ProcessedAt() time.Time {
	return w.processedAt
}

// NewWithdrawalFromRepository creates a new Withdrawal instance from repository data
func NewWithdrawalFromRepository(id WithdrawalID, userID user.UserID, orderNumber string, amount float64, processedAt time.Time) *Withdrawal {
	return &Withdrawal{
		id:          id,
		userID:      userID,
		orderNumber: orderNumber,
		amount:      amount,
		processedAt: processedAt,
	}
}
