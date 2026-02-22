package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	query := `
		INSERT INTO users (id, email, password, name, nickname, country, city, institution, role, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	_, err := r.pool.Exec(ctx, query,
		u.ID,
		u.Email.String(),
		u.Password.Hash(),
		u.Name,
		u.Nickname.String(),
		u.Country,
		u.City,
		u.Institution,
		u.Role.String(),
		u.Status.String(),
		u.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	return nil
}

func (r *UserRepository) ExistsByEmail(ctx context.Context, email user.Email) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, email.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email: %w", err)
	}

	return exists, nil
}

func (r *UserRepository) ExistsByNickname(ctx context.Context, nickname user.Nickname) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE nickname = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, nickname.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check nickname: %w", err)
	}

	return exists, nil
}
