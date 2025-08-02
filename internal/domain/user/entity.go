package user

import (
	"time"

	"github.com/google/uuid"
)

type UserID = uuid.UUID

// User represents a user in the system
type User struct {
	id           UserID
	login        string
	passwordHash string
	balance      Balance
	createdAt    time.Time
}

// Balance represents the user's balance
type Balance struct {
	Current   float64
	Withdrawn float64
}

// New creates a new user with the given login and password hash
func New(login, passwordHash string) (*User, error) {
	if validationErrors := ValidateLogin(login); validationErrors.HasErrors() {
		return nil, validationErrors
	}
	if passwordHash == "" {
		return nil, ErrPasswordHashEmpty
	}

	return &User{
		id:           uuid.New(),
		login:        login,
		passwordHash: passwordHash,
		balance:      Balance{Current: 0, Withdrawn: 0},
		createdAt:    time.Now(),
	}, nil
}

// ID returns the user's ID
func (u *User) ID() UserID {
	return u.id
}

// Login returns the user's login
func (u *User) Login() string {
	return u.login
}

// PasswordHash returns the user's password hash
func (u *User) PasswordHash() string {
	return u.passwordHash
}

// Balance returns the user's balance
func (u *User) Balance() Balance {
	return u.balance
}

// CreatedAt returns the user's creation time
func (u *User) CreatedAt() time.Time {
	return u.createdAt
}

// UpdateBalance updates the user's balance
func (u *User) UpdateBalance(current, withdrawn float64) {
	u.balance.Current = current
	u.balance.Withdrawn = withdrawn
}

// AddBalance adds the specified amount to the user's current balance
func (u *User) AddBalance(amount float64) {
	u.balance.Current += amount
}

// WithdrawBalance withdraws the specified amount from the user's balance
func (u *User) WithdrawBalance(amount float64) error {
	if u.balance.Current < amount {
		return ErrInsufficientFunds
	}
	u.balance.Current -= amount
	u.balance.Withdrawn += amount
	return nil
}

// NewUserFromRepository creates a user from the repository data
func NewUserFromRepository(id UserID, login, passwordHash string, balance Balance, createdAt time.Time) *User {
	return &User{
		id:           id,
		login:        login,
		passwordHash: passwordHash,
		balance:      balance,
		createdAt:    createdAt,
	}
}
