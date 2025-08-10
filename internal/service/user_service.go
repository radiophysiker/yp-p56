package service

import (
	"context"
	"errors"

	"github.com/radiophysiker/d56/internal/domain/user"
)

type UserService struct {
	userRepo        user.Repository
	passwordService user.PasswordService
}

// ErrInvalidCredentials is returned when authentication fails due to wrong login or password.
var ErrInvalidCredentials = errors.New("invalid credentials")

func NewUserService(userRepo user.Repository, passwordService user.PasswordService) *UserService {
	return &UserService{
		userRepo:        userRepo,
		passwordService: passwordService,
	}
}

func (s *UserService) Register(ctx context.Context, login, password string) (*user.User, error) {
	if err := user.ValidateUserCredentials(login, password); err != nil {
		return nil, err
	}

	passwordHash, err := s.passwordService.HashPassword(password)
	if err != nil {
		return nil, err
	}

	newUser, err := user.New(login, passwordHash)
	if err != nil {
		return nil, err
	}

	if err := s.userRepo.Save(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

func (s *UserService) Authenticate(ctx context.Context, login, password string) (*user.User, error) {
	u, err := s.userRepo.FindByLogin(ctx, login)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !s.passwordService.CheckPassword(u.PasswordHash(), password) {
		return nil, ErrInvalidCredentials
	}

	return u, nil
}

func (s *UserService) GetBalance(ctx context.Context, userID user.UserID) (user.Balance, error) {
	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return user.Balance{}, err
	}

	return u.Balance(), nil
}
