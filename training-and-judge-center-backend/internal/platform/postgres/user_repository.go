package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
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

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	query := `
		UPDATE users
		SET name = $1, nickname = $2, institution = $3, email = $4, password = $5,
		    role = $6, status = $7, updated_at = $8, deactivated_at = $9
		WHERE id = $10`

	_, err := r.pool.Exec(ctx, query,
		u.Name,
		u.Nickname.String(),
		u.Institution,
		u.Email.String(),
		u.Password.Hash(),
		u.Role.String(),
		u.Status.String(),
		u.UpdatedAt,
		u.DeactivatedAt,
		u.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
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

const userColumns = `id, email, password, name, nickname, country, city, institution, role, status, created_at, updated_at, deactivated_at`

func scanUser(row pgx.Row) (*user.User, error) {
	var u user.User
	var emailStr, passwordHash, nicknameStr, roleStr, statusStr string
	var updatedAt, deactivatedAt *time.Time

	err := row.Scan(
		&u.ID,
		&emailStr,
		&passwordHash,
		&u.Name,
		&nicknameStr,
		&u.Country,
		&u.City,
		&u.Institution,
		&roleStr,
		&statusStr,
		&u.CreatedAt,
		&updatedAt,
		&deactivatedAt,
	)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}

	parsedEmail, _ := user.NewEmail(emailStr)
	u.Email = parsedEmail
	u.Password = user.NewPasswordFromHash(passwordHash)

	parsedNickname, _ := user.NewNickname(nicknameStr)
	u.Nickname = parsedNickname

	parsedRole, _ := user.NewRole(roleStr)
	u.Role = parsedRole

	parsedStatus, _ := user.NewStatus(statusStr)
	u.Status = parsedStatus

	u.UpdatedAt = updatedAt
	u.DeactivatedAt = deactivatedAt

	return &u, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email user.Email) (*user.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`

	u, err := scanUser(r.pool.QueryRow(ctx, query, email.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	u, err := scanUser(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByNickname(ctx context.Context, nickname user.Nickname) (*user.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE LOWER(nickname) = LOWER($1)`

	u, err := scanUser(r.pool.QueryRow(ctx, query, nickname.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to find user by nickname: %w", err)
	}
	return u, nil
}

