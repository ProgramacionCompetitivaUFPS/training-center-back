package user

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UserRepository struct {
	querier infraPostgres.Querier
}

func NewUserRepository(querier infraPostgres.Querier) *UserRepository {
	return &UserRepository{querier: querier}
}

func (r *UserRepository) Save(ctx context.Context, u *domainUser.User) error {
	query := `
		INSERT INTO users (id, email, password, name, nickname, country, city, institution, role, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		u.ID(),
		u.Email().String(),
		u.Password().Hash(),
		u.Name(),
		u.Nickname().String(),
		u.Country(),
		u.City(),
		u.Institution(),
		u.Role().String(),
		u.Status().String(),
		u.CreatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_email_key":
				return apperror.NewConflict(domainUser.ErrCodeEmailConflict, "email already in use")
			case "users_nickname_key":
				return apperror.NewConflict(domainUser.ErrCodeNicknameConflict, "nickname already in use")
			}
		}
		slog.ErrorContext(ctx, "failed to save user", "error", err)
		return apperror.NewInternal()
	}

	return nil
}

func (r *UserRepository) Update(ctx context.Context, u *domainUser.User) error {
	query := `
		UPDATE users
		SET name = $1, nickname = $2, institution = $3, email = $4, password = $5,
		    role = $6, status = $7, updated_at = $8, deactivated_at = $9,
		    country = $10, city = $11
		WHERE id = $12`

	var emailVal interface{}
	if emailStr := u.Email().String(); emailStr != "" {
		emailVal = emailStr
	}

	querier := infraPostgres.GetQuerier(ctx, r.querier)
	_, err := querier.Exec(ctx, query,
		u.Name(),
		u.Nickname().String(),
		u.Institution(),
		emailVal,
		u.Password().Hash(),
		u.Role().String(),
		u.Status().String(),
		u.UpdatedAt(),
		u.DeactivatedAt(),
		u.Country(),
		u.City(),
		u.ID(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_nickname_key":
				return apperror.NewConflict(domainUser.ErrCodeNicknameConflict, "nickname already in use")
			case "users_email_key":
				return apperror.NewConflict(domainUser.ErrCodeEmailConflict, "email already in use")
			}
		}
		slog.ErrorContext(ctx, "failed to update user", "error", err)
		return apperror.NewInternal()
	}

	return nil
}

const userColumns = `id, email, password, name, nickname, country, city, institution, role, status, created_at, updated_at, deactivated_at`

func scanUser(row pgx.Row) (*domainUser.User, error) {
	var id, name, country, city, institution string
	var emailStr *string
	var passwordHash, nicknameStr, roleStr, statusStr string
	var createdAt time.Time
	var updatedAt, deactivatedAt *time.Time

	err := row.Scan(
		&id,
		&emailStr,
		&passwordHash,
		&name,
		&nicknameStr,
		&country,
		&city,
		&institution,
		&roleStr,
		&statusStr,
		&createdAt,
		&updatedAt,
		&deactivatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	return domainUser.RestoreUser(
		id,
		emailStr,
		passwordHash,
		name,
		nicknameStr,
		country,
		city,
		institution,
		roleStr,
		statusStr,
		createdAt,
		updatedAt,
		deactivatedAt,
	), nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email domainUser.Email) (*domainUser.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE email = $1`

	u, err := scanUser(r.querier.QueryRow(ctx, query, email.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*domainUser.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE id = $1`

	u, err := scanUser(r.querier.QueryRow(ctx, query, id))
	if err != nil {
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}
	return u, nil
}

func (r *UserRepository) FindByNickname(ctx context.Context, nickname domainUser.Nickname) (*domainUser.User, error) {
	query := `SELECT ` + userColumns + ` FROM users WHERE LOWER(nickname) = LOWER($1)`

	u, err := scanUser(r.querier.QueryRow(ctx, query, nickname.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to find user by nickname: %w", err)
	}
	return u, nil
}

func escapeILIKE(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `%`, `\%`)
	s = strings.ReplaceAll(s, `_`, `\_`)
	return s
}

var sortColumnMap = map[domainUser.SortField]string{
	domainUser.SortByCreatedAt:     "created_at",
	domainUser.SortByName:          "name",
	domainUser.SortByNickname:      "nickname",
	domainUser.SortByEmail:         "email",
	domainUser.SortByDeactivatedAt: "deactivated_at",
}

func (r *UserRepository) FindAll(ctx context.Context, filter domainUser.UserFilter) ([]*domainUser.User, int, error) {
	args := []interface{}{}
	conditions := []string{}
	n := 1

	// Filter by roles (OR logic)
	if len(filter.Roles) > 0 {
		placeholders := make([]string, len(filter.Roles))
		for i, role := range filter.Roles {
			placeholders[i] = fmt.Sprintf("$%d", n)
			args = append(args, role.String())
			n++
		}
		conditions = append(conditions, fmt.Sprintf("role IN (%s)", strings.Join(placeholders, ", ")))
	}

	// Filter by status
	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("status = $%d", n))
		args = append(args, filter.Status.String())
		n++
	}

	// Filter by country (case-insensitive exact match)
	if filter.Country != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(country) = LOWER($%d)", n))
		args = append(args, filter.Country)
		n++
	}

	// Filter by city
	if filter.City != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(city) = LOWER($%d)", n))
		args = append(args, filter.City)
		n++
	}

	// Filter by institution
	if filter.Institution != "" {
		conditions = append(conditions, fmt.Sprintf("LOWER(institution) = LOWER($%d)", n))
		args = append(args, filter.Institution)
		n++
	}

	// Text search
	if filter.SearchTerm != "" {
		searchPattern := "%" + escapeILIKE(filter.SearchTerm) + "%"
		switch filter.SearchField {
		case domainUser.SearchByName:
			conditions = append(conditions, fmt.Sprintf("name ILIKE $%d", n))
			args = append(args, searchPattern)
			n++
		case domainUser.SearchByNickname:
			conditions = append(conditions, fmt.Sprintf("nickname ILIKE $%d", n))
			args = append(args, searchPattern)
			n++
		case domainUser.SearchByEmail:
			conditions = append(conditions, fmt.Sprintf("email ILIKE $%d", n))
			args = append(args, searchPattern)
			n++
		case domainUser.SearchByInstitution:
			conditions = append(conditions, fmt.Sprintf("institution ILIKE $%d", n))
			args = append(args, searchPattern)
			n++
		default: // SearchByAll
			conditions = append(conditions, fmt.Sprintf(
				"(name ILIKE $%d OR nickname ILIKE $%d OR email ILIKE $%d OR institution ILIKE $%d)",
				n, n, n, n,
			))
			args = append(args, searchPattern)
			n++
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Safe ORDER BY using whitelist
	colName := sortColumnMap[filter.Sort]
	if colName == "" {
		colName = "created_at"
	}
	order := "DESC"
	if filter.Order == domainUser.SortOrderAsc {
		order = "ASC"
	}
	orderClause := fmt.Sprintf("ORDER BY %s %s NULLS LAST", colName, order)

	// COUNT query for pagination metadata
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM users %s`, whereClause)
	var totalCount int
	if err := r.querier.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Main query with pagination
	offset := (filter.Page - 1) * filter.Limit
	dataQuery := fmt.Sprintf(
		`SELECT %s FROM users %s %s LIMIT $%d OFFSET $%d`,
		userColumns, whereClause, orderClause, n, n+1,
	)
	args = append(args, filter.Limit, offset)

	rows, err := r.querier.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*domainUser.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}
		if u != nil {
			users = append(users, u)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error iterating user rows: %w", err)
	}

	return users, totalCount, nil
}
