package user

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateLogin(t *testing.T) {
	tests := []struct {
		name   string
		login  string
		hasErr bool
	}{
		{"empty", "", true},
		{"too short", "abc", true},
		{"below min length", strings.Repeat("a", MinLoginLength-1), true},
		{"at min valid length", strings.Repeat("a", MinLoginLength), false},
		{"at max valid length", strings.Repeat("a", MaxLoginLength), false},
		{"above max valid length", strings.Repeat("a", MaxLoginLength+1), true},
		{"invalid chars", "user!name", true},
		{"valid simple", "user_123", false},
		{"valid hyphen", "user-name", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidateLogin(tt.login)
			assert.Equalf(t, tt.hasErr, errs.HasErrors(), "errs=%v", errs)
		})
	}
}

func TestValidatePassword(t *testing.T) {
	min := strings.Repeat("a1", MinPasswordLength/2)
	max := strings.Repeat("a1", MaxPasswordLength/2)
	aboveMax := strings.Repeat("a1", (MaxPasswordLength/2)+1)[:MaxPasswordLength+1]

	tests := []struct {
		name     string
		password string
		hasErr   bool
	}{
		{"empty", "", true},
		{"below min length", "a1b2", true},
		{"at min valid length", min, false},
		{"at max valid length", max, false},
		{"above max valid length", aboveMax, true},
		{"only letters", "abcdef", true},
		{"only digits", "123456", true},
		{"letters and digits", "abc123", false},
		{"mixed case letters and digits", "AbCdEf123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errs := ValidatePassword(tt.password)
			assert.Equalf(t, tt.hasErr, errs.HasErrors(), "errs=%v", errs)
		})
	}
}

func TestValidateUserCredentials(t *testing.T) {
	err := ValidateUserCredentials("ok_user", "ok12345")
	require.NoError(t, err)

	err = ValidateUserCredentials("", "ok12345")
	require.Error(t, err)

	err = ValidateUserCredentials("ok_user", "letters")
	require.Error(t, err)
}

func TestValidateUserCredentials_Aggregates(t *testing.T) {
	err := ValidateUserCredentials("a!", "123")
	var verrs ValidationErrors
	require.Error(t, err)
	assert.True(t, errors.As(err, &verrs))
	assert.GreaterOrEqual(t, len(verrs), 2)
}

func TestNewUser(t *testing.T) {
	_, err := New("bad login!", "hash")
	require.Error(t, err)
	_, err = New("good_login", "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrPasswordHashEmpty)
	u, err := New("good_login", "hash")
	require.NoError(t, err)
	assert.Equal(t, "good_login", u.Login())
	assert.Equal(t, "hash", u.PasswordHash())
	assert.Equal(t, 0.0, u.Balance().Current)
	assert.Equal(t, 0.0, u.Balance().Withdrawn)
	assert.NotEqual(t, UserID{}, u.ID())
	assert.False(t, u.CreatedAt().IsZero())
}

func TestUserBalanceOperations(t *testing.T) {
	u, err := New("login123", "hash")
	require.NoError(t, err)

	u.AddBalance(100)
	assert.Equal(t, 100.0, u.Balance().Current)

	err = u.WithdrawBalance(30)
	require.NoError(t, err)
	assert.Equal(t, 70.0, u.Balance().Current)
	assert.Equal(t, 30.0, u.Balance().Withdrawn)

	err = u.WithdrawBalance(1000)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInsufficientFunds)

	u.UpdateBalance(10, 5)
	assert.Equal(t, 10.0, u.Balance().Current)
	assert.Equal(t, 5.0, u.Balance().Withdrawn)
}

func TestNewUserFromRepository(t *testing.T) {
	before := time.Now()
	u := NewUserFromRepository(UserID{}, "repo_login", "hash", Balance{Current: 10, Withdrawn: 2}, before)
	assert.Equal(t, "repo_login", u.Login())
	assert.Equal(t, "hash", u.PasswordHash())
	assert.Equal(t, 10.0, u.Balance().Current)
	assert.Equal(t, 2.0, u.Balance().Withdrawn)
	assert.True(t, u.CreatedAt().Equal(before))
}
