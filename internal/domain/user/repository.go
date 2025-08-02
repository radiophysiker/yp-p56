package user

import (
	"context"
	"errors"
)

var (
	ErrLoginAlreadyExists = errors.New("login already exists")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrPasswordHashEmpty  = errors.New("password hash cannot be empty")
)

type Repository interface {
	Save(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	FindByLogin(ctx context.Context, login string) (*User, error)
	FindByID(ctx context.Context, id UserID) (*User, error)
}
