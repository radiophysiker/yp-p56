package postgres

import (
	"context"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/radiophysiker/d56/internal/domain/user"
)

// UserRepository implements the user.Repository interface for PostgreSQL
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository creates a new UserRepository with the given database connection
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save creates a new user in the database (INSERT only)
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	query := `
			INSERT INTO users (id, login, password_hash, created_at)
			VALUES ($1, $2, $3, $4)
		`
	_, err := r.db.ExecContext(ctx, query, u.ID(), u.Login(), u.PasswordHash(), u.CreatedAt())
	if err != nil {
		zap.L().Error("Failed to save user", zap.Error(err), zap.String("login", u.Login()))

		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") &&
			strings.Contains(err.Error(), "users_login_key") {
			return user.ErrLoginAlreadyExists
		}
		return err
	}
	return nil
}

// Update updates an existing user in the database (UPDATE only)
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users 
		SET current_balance = $2, withdrawn_balance = $3
		WHERE id = $1
	`
	balance := u.Balance()
	_, err := r.db.ExecContext(ctx, query, u.ID(), balance.Current, balance.Withdrawn)
	return err
}

// FindByLogin retrieves a user by their login
func (r *UserRepository) FindByLogin(ctx context.Context, login string) (*user.User, error) {
	query := `SELECT id, login, password_hash, current_balance, withdrawn_balance, created_at FROM users WHERE login = $1`

	var id user.UserID
	var userLogin, passwordHash string
	var balance user.Balance
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, query, login).Scan(
		&id, &userLogin, &passwordHash, &balance.Current, &balance.Withdrawn, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return user.NewUserFromRepository(id, userLogin, passwordHash, balance, createdAt), nil
}

// FindByID retrieves a user by their ID
func (r *UserRepository) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	query := `SELECT id, login, password_hash, current_balance, withdrawn_balance, created_at FROM users WHERE id = $1`

	var userID user.UserID
	var login, passwordHash string
	var balance user.Balance
	var createdAt time.Time

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&userID, &login, &passwordHash, &balance.Current, &balance.Withdrawn, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return user.NewUserFromRepository(userID, login, passwordHash, balance, createdAt), nil
}

// UserRepositoryWithTx implements the user.Repository interface for PostgreSQL with transaction support
type UserRepositoryWithTx struct {
	tx *sqlx.Tx
}

// NewUserRepositoryWithTx creates a new UserRepositoryWithTx with the given transaction
func NewUserRepositoryWithTx(tx *sqlx.Tx) *UserRepositoryWithTx {
	return &UserRepositoryWithTx{tx: tx}
}

// Save creates a new user in the database (INSERT only)
func (r *UserRepositoryWithTx) Save(ctx context.Context, u *user.User) error {
	query := `
			INSERT INTO users (id, login, password_hash, created_at)
			VALUES ($1, $2, $3, $4)
		`
	_, err := r.tx.ExecContext(ctx, query, u.ID(), u.Login(), u.PasswordHash(), u.CreatedAt())
	if err != nil {
		zap.L().Error("Failed to save user", zap.Error(err), zap.String("login", u.Login()))

		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") &&
			strings.Contains(err.Error(), "users_login_key") {
			return user.ErrLoginAlreadyExists
		}
		return err
	}
	return nil
}

// Update updates an existing user in the database (UPDATE only)
func (r *UserRepositoryWithTx) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users 
		SET current_balance = $2, withdrawn_balance = $3
		WHERE id = $1
	`
	balance := u.Balance()
	_, err := r.tx.ExecContext(ctx, query, u.ID(), balance.Current, balance.Withdrawn)
	return err
}

// FindByLogin retrieves a user by their login
func (r *UserRepositoryWithTx) FindByLogin(ctx context.Context, login string) (*user.User, error) {
	query := `SELECT id, login, password_hash, current_balance, withdrawn_balance, created_at FROM users WHERE login = $1`

	var id user.UserID
	var userLogin, passwordHash string
	var balance user.Balance
	var createdAt time.Time

	err := r.tx.QueryRowContext(ctx, query, login).Scan(
		&id, &userLogin, &passwordHash, &balance.Current, &balance.Withdrawn, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return user.NewUserFromRepository(id, userLogin, passwordHash, balance, createdAt), nil
}

// FindByID retrieves a user by their ID
func (r *UserRepositoryWithTx) FindByID(ctx context.Context, id user.UserID) (*user.User, error) {
	query := `SELECT id, login, password_hash, current_balance, withdrawn_balance, created_at FROM users WHERE id = $1`

	var userID user.UserID
	var login, passwordHash string
	var balance user.Balance
	var createdAt time.Time

	err := r.tx.QueryRowContext(ctx, query, id).Scan(
		&userID, &login, &passwordHash, &balance.Current, &balance.Withdrawn, &createdAt,
	)
	if err != nil {
		return nil, err
	}

	return user.NewUserFromRepository(userID, login, passwordHash, balance, createdAt), nil
}
