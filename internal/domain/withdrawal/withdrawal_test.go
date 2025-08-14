package withdrawal

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/radiophysiker/d56/internal/domain/order"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewWithdrawalValidation(t *testing.T) {
	userID := uuid.New()

	_, err := New(uuid.Nil, "49927398716", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserIDEmpty)

	_, err = New(userID, "", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, order.ErrOrderNumberEmpty)

	_, err = New(userID, "12345", 10)
	require.Error(t, err)
	assert.ErrorIs(t, err, order.ErrInvalidFormat)

	_, err = New(userID, "49927398716", 0)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAmountMustBePositive)

	_, err = New(userID, "49927398716", -5)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrAmountMustBePositive)

	w, err := New(userID, "49927398716", 12.5)
	require.NoError(t, err)
	assert.Equal(t, userID, w.UserID())
	assert.Equal(t, "49927398716", w.OrderNumber())
	assert.Equal(t, 12.5, w.Amount())
	assert.False(t, w.ProcessedAt().IsZero())
}

func TestNewWithdrawalFromRepository(t *testing.T) {
	userID := uuid.New()
	processed := time.Now()
	w := NewWithdrawalFromRepository(5, userID, "79927398713", 33.3, processed)
	assert.Equal(t, WithdrawalID(5), w.ID())
	assert.Equal(t, userID, w.UserID())
	assert.Equal(t, "79927398713", w.OrderNumber())
	assert.Equal(t, 33.3, w.Amount())
	assert.True(t, w.ProcessedAt().Equal(processed))
}
