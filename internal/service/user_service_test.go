package service

import (
	"context"
	"errors"
	"testing"

	user "github.com/radiophysiker/d56/internal/domain/user"
	userMocks "github.com/radiophysiker/d56/internal/mocks/domain/user"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestNewUserService(t *testing.T) {
	t.Parallel()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	svc := NewUserService(repo, pwd)

	require.NotNil(t, svc)

	rImpl, ok := svc.userRepo.(*userMocks.MockRepository)
	require.True(t, ok)
	require.Same(t, repo, rImpl)

	pImpl, ok := svc.passwordService.(*userMocks.MockPasswordService)
	require.True(t, ok)
	require.Same(t, pwd, pImpl)
}

func TestUserService_Register_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	pwd.EXPECT().HashPassword("abc12345").Return("hashed", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.Login() == "valid_user" && u.PasswordHash() == "hashed"
	})).Return(nil).Once()

	svc := NewUserService(repo, pwd)

	u, err := svc.Register(ctx, "valid_user", "abc12345")
	require.NoError(t, err)
	require.NotNil(t, u)
	assert.Equal(t, "valid_user", u.Login())
	assert.Equal(t, "hashed", u.PasswordHash())
}

func TestUserService_Register_InvalidCredentials(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	svc := NewUserService(repo, pwd)

	u, err := svc.Register(ctx, "bad", "123")
	require.Error(t, err)
	require.Nil(t, u)
	pwd.AssertNotCalled(t, "HashPassword", mock.Anything)
	repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestUserService_Register_HashError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	pwd.EXPECT().HashPassword("abc12345").Return("", errors.New("boom")).Once()

	svc := NewUserService(repo, pwd)

	u, err := svc.Register(ctx, "valid_user", "abc12345")
	require.Error(t, err)
	require.Nil(t, u)
	repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestUserService_Register_EmptyHashFromService(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	pwd.EXPECT().HashPassword("abc12345").Return("", nil).Once()

	svc := NewUserService(repo, pwd)

	u, err := svc.Register(ctx, "valid_user", "abc12345")
	require.Error(t, err)
	require.Nil(t, u)
	assert.ErrorIs(t, err, user.ErrPasswordHashEmpty)
	repo.AssertNotCalled(t, "Save", mock.Anything, mock.Anything)
}

func TestUserService_Register_SaveError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	pwd.EXPECT().HashPassword("abc12345").Return("hashed", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.AnythingOfType("*user.User")).Return(errors.New("db error")).Once()

	svc := NewUserService(repo, pwd)

	u, err := svc.Register(ctx, "valid_user", "abc12345")
	require.Error(t, err)
	require.Nil(t, u)
}

func TestUserService_Register_SaveConflict_LoginAlreadyExists(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	repo := userMocks.NewMockRepository(t)
	pwd := userMocks.NewMockPasswordService(t)

	pwd.EXPECT().HashPassword("abc12345").Return("hashed", nil).Once()
	repo.EXPECT().Save(mock.Anything, mock.AnythingOfType("*user.User")).Return(user.ErrLoginAlreadyExists).Once()

	svc := NewUserService(repo, pwd)

	u, err := svc.Register(ctx, "valid_user", "abc12345")
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrLoginAlreadyExists)
	require.Nil(t, u)
}

func TestUserService_Authenticate_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	existing, _ := user.New("valid_user", "hashed")

	repo := userMocks.NewMockRepository(t)
	repo.EXPECT().FindByLogin(mock.Anything, "valid_user").Return(existing, nil).Once()

	pwd := userMocks.NewMockPasswordService(t)
	pwd.EXPECT().CheckPassword("hashed", "abc12345").Return(true).Once()

	svc := NewUserService(repo, pwd)

	got, err := svc.Authenticate(ctx, "valid_user", "abc12345")
	require.NoError(t, err)
	require.NotNil(t, got)
}

func TestUserService_Authenticate_RepoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	repo := userMocks.NewMockRepository(t)
	repo.EXPECT().FindByLogin(mock.Anything, "valid_user").Return((*user.User)(nil), errors.New("not found")).Once()

	pwd := userMocks.NewMockPasswordService(t)

	svc := NewUserService(repo, pwd)

	got, err := svc.Authenticate(ctx, "valid_user", "abc12345")
	require.Error(t, err)
	require.Nil(t, got)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
	pwd.AssertNotCalled(t, "CheckPassword", mock.Anything, mock.Anything)
}

func TestUserService_Authenticate_WrongPassword(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	existing, _ := user.New("valid_user", "hashed")

	repo := userMocks.NewMockRepository(t)
	repo.EXPECT().FindByLogin(mock.Anything, "valid_user").Return(existing, nil).Once()

	pwd := userMocks.NewMockPasswordService(t)
	pwd.EXPECT().CheckPassword("hashed", "bad").Return(false).Once()

	svc := NewUserService(repo, pwd)

	got, err := svc.Authenticate(ctx, "valid_user", "bad")
	require.Error(t, err)
	require.Nil(t, got)
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_GetBalance_Success(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	u, _ := user.New("valid_user", "hashed")
	u.AddBalance(42)

	repo := userMocks.NewMockRepository(t)
	repo.EXPECT().FindByID(mock.Anything, u.ID()).Return(u, nil).Once()

	pwd := userMocks.NewMockPasswordService(t)
	svc := NewUserService(repo, pwd)

	b, err := svc.GetBalance(ctx, u.ID())
	require.NoError(t, err)
	assert.Equal(t, user.Balance{Current: 42, Withdrawn: 0}, b)
}

func TestUserService_GetBalance_RepoError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	repo := userMocks.NewMockRepository(t)
	repo.EXPECT().FindByID(mock.Anything, mock.Anything).Return((*user.User)(nil), errors.New("db err")).Once()

	pwd := userMocks.NewMockPasswordService(t)
	svc := NewUserService(repo, pwd)

	b, err := svc.GetBalance(ctx, user.UserID{})
	require.Error(t, err)
	assert.Equal(t, user.Balance{}, b)
}
