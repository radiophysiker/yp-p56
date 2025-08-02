package order

import (
	"time"

	"github.com/radiophysiker/d56/internal/domain/user"
)

type ID int64

type Status string

const (
	StatusNew        Status = "NEW"
	StatusProcessing Status = "PROCESSING"
	StatusInvalid    Status = "INVALID"
	StatusProcessed  Status = "PROCESSED"
)

type Order struct {
	id          ID
	userID      user.UserID
	number      string
	status      Status
	accrual     *float64
	uploadedAt  time.Time
	processedAt *time.Time
}

func New(userID user.UserID, number string) (*Order, error) {
	if userID.String() == "" {
		return nil, ErrUserIDEmpty
	}

	if number == "" {
		return nil, ErrOrderNumberEmpty
	}

	if !IsValidOrderNumber(number) {
		return nil, ErrInvalidFormat
	}

	return &Order{
		userID:     userID,
		number:     number,
		status:     StatusNew,
		uploadedAt: time.Now(),
	}, nil
}

func (o *Order) ID() ID {
	return o.id
}

func (o *Order) UserID() user.UserID {
	return o.userID
}

func (o *Order) Number() string {
	return o.number
}

func (o *Order) Status() Status {
	return o.status
}

func (o *Order) Accrual() *float64 {
	return o.accrual
}

func (o *Order) UploadedAt() time.Time {
	return o.uploadedAt
}

func (o *Order) ProcessedAt() *time.Time {
	return o.processedAt
}

func (o *Order) SetStatus(status Status) {
	o.status = status
	if status == StatusProcessed || status == StatusInvalid {
		now := time.Now()
		o.processedAt = &now
	}
}

func (o *Order) SetAccrual(accrual float64) {
	o.accrual = &accrual
}

// NewOrderFromRepository creates a new Order instance from repository data
func NewOrderFromRepository(id ID, userID user.UserID, number string, status Status, accrual *float64, uploadedAt time.Time, processedAt *time.Time) *Order {
	return &Order{
		id:          id,
		userID:      userID,
		number:      number,
		status:      status,
		accrual:     accrual,
		uploadedAt:  uploadedAt,
		processedAt: processedAt,
	}
}
