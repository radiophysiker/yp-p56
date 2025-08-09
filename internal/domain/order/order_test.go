package order

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsValidOrderNumber(t *testing.T) {
	longZeros100 := strings.Repeat("0", 100)
	longZeros99Plus1 := strings.Repeat("0", 99) + "1"

	tests := []struct {
		number  string
		isValid bool
		name    string
	}{
		{number: "", isValid: false, name: "empty"},
		{number: "abc", isValid: false, name: "non-digits"},
		{number: " 79927398713", isValid: false, name: "leading-space"},
		{number: "79927398713 ", isValid: false, name: "trailing-space"},
		{number: "79 9273 98713", isValid: false, name: "spaces-inside"},
		{number: "7992-7398-713", isValid: false, name: "hyphens-inside"},
		{number: "0000", isValid: true, name: "leading-zeros-valid-short"},
		{number: "79927398713", isValid: true, name: "luhn-classic-valid"},
		{number: "79927398714", isValid: false, name: "luhn-classic-invalid"},
		{number: "4539578763621486", isValid: true, name: "visa-test-valid"},
		{number: "1234567812345670", isValid: true, name: "sequence-valid"},
		{number: "1234567812345678", isValid: false, name: "sequence-invalid"},
		{number: longZeros100, isValid: true, name: "very-long-all-zeros-valid"},
		{number: longZeros99Plus1, isValid: false, name: "very-long-ending-with-1-invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidOrderNumber(tt.number)
			assert.Equalf(t, tt.isValid, got, "IsValidOrderNumber(%q)", tt.number)
		})
	}
}

func TestNewOrderValidation(t *testing.T) {
	userID := uuid.New()

	_, err := New(uuid.Nil, "49927398716")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUserIDEmpty)

	_, err = New(userID, "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrOrderNumberEmpty)

	_, err = New(userID, "12345")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidFormat)

	o, err := New(userID, "49927398716")
	require.NoError(t, err)
	assert.Equal(t, userID, o.UserID())
	assert.Equal(t, "49927398716", o.Number())
	assert.Equal(t, StatusNew, o.Status())
	assert.Equal(t, int64(1), o.Version())
	assert.False(t, o.UploadedAt().IsZero())
	assert.Nil(t, o.Accrual())
}

func TestOrderAccrualAndVersion(t *testing.T) {
	userID := uuid.New()
	o, err := New(userID, "79927398713")
	require.NoError(t, err)

	o.SetAccrual(12.34)
	require.NotNil(t, o.Accrual())
	assert.Equal(t, 12.34, *o.Accrual())

	v := o.Version()
	o.IncrementVersion()
	assert.Equal(t, v+1, o.Version())
}

func TestOrderStateTransitions(t *testing.T) {
	userID := uuid.New()
	o, err := New(userID, "4539578763621486")
	require.NoError(t, err)
	sm := NewStateMachine()

	err = o.TransitionTo(StatusNew, sm)
	require.Error(t, err)
	assert.True(t, IsInvalidTransitionError(err))

	err = o.TransitionTo(StatusProcessed, sm)
	require.Error(t, err)
	assert.True(t, IsInvalidTransitionError(err))

	v := o.Version()
	err = o.TransitionTo(StatusProcessing, sm)
	require.NoError(t, err)
	assert.Equal(t, StatusProcessing, o.Status())
	assert.Equal(t, v+1, o.Version())
	assert.Nil(t, o.ProcessedAt())

	v = o.Version()
	err = o.TransitionTo(StatusNew, sm)
	require.NoError(t, err)
	assert.Equal(t, StatusNew, o.Status())
	assert.Equal(t, v+1, o.Version())
	// processedAt must remain nil when transitioning to non-final states
	assert.Nil(t, o.ProcessedAt())

	err = o.TransitionTo(StatusInvalid, sm)
	require.NoError(t, err)
	assert.Equal(t, StatusInvalid, o.Status())
	require.NotNil(t, o.ProcessedAt())

	o2, err := New(userID, "49927398716")
	require.NoError(t, err)
	err = o2.TransitionTo(StatusProcessing, sm)
	require.NoError(t, err)
	err = o2.TransitionTo(StatusProcessed, sm)
	require.NoError(t, err)
	assert.Equal(t, StatusProcessed, o2.Status())
	require.NotNil(t, o2.ProcessedAt())
}

func TestStateMachineHelpers(t *testing.T) {
	sm := NewStateMachine()

	assert.True(t, sm.CanTransition(StatusNew, StatusProcessing))
	assert.False(t, sm.CanTransition(StatusNew, StatusProcessed))

	got := sm.GetValidTransitions(StatusNew)
	assert.Len(t, got, 2)
	m := map[Status]bool{}
	for _, s := range got {
		m[s] = true
	}
	assert.True(t, m[StatusProcessing] && m[StatusInvalid], "expected transitions to PROCESSING and INVALID, got %v", got)

	assert.False(t, sm.IsFinalState(StatusNew))
	assert.True(t, sm.IsFinalState(StatusProcessed))
	assert.True(t, sm.IsFinalState(StatusInvalid))
}

func TestNewOrderFromRepository(t *testing.T) {
	userID := uuid.New()
	acc := 42.0
	uploaded := time.Now().Add(-time.Hour)
	processed := time.Now()
	o := NewOrderFromRepository(10, userID, "79927398713", StatusProcessed, &acc, uploaded, &processed, 7)

	assert.Equal(t, ID(10), o.ID())
	assert.Equal(t, userID, o.UserID())
	assert.Equal(t, "79927398713", o.Number())
	assert.Equal(t, StatusProcessed, o.Status())
	require.NotNil(t, o.Accrual())
	assert.Equal(t, 42.0, *o.Accrual())
	assert.True(t, o.UploadedAt().Equal(uploaded))
	require.NotNil(t, o.ProcessedAt())
	assert.True(t, o.ProcessedAt().Equal(processed))
	assert.Equal(t, int64(7), o.Version())
}
