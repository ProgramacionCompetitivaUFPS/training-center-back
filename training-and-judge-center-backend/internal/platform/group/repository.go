package group

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	infraPostgres "github.com/training-judge-center/backend/internal/infrastructure/postgres"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GroupRepository struct {
	db *pgxpool.Pool
}

func NewGroupRepository(db *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{db: db}
}

func (r *GroupRepository) Save(ctx context.Context, g *domainGroup.Group) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	query := `
		INSERT INTO groups (
			id, name, description, visibility, join_policy,
			is_default, created_by, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9
		)
	`

	_, err := q.Exec(ctx, query,
		g.ID(),
		g.Name().Value(),
		g.Description(),
		g.Visibility().String(),
		g.JoinPolicy().String(),
		g.IsDefault(),
		g.CreatedBy().Value(),
		g.CreatedAt(),
		g.UpdatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return apperror.NewConflict(domainGroup.ErrCodeNameAlreadyExists, "A group with this name already exists")
		}
		slog.ErrorContext(ctx, "failed to save group", "error", err, "group_id", g.ID())
		return apperror.NewInternal()
	}

	return nil
}

func (r *GroupRepository) FindByID(ctx context.Context, id string) (*domainGroup.Group, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	row := q.QueryRow(ctx, `
		SELECT id, name, description, visibility, join_policy,
		       is_default, created_by, created_at, updated_at
		FROM groups
		WHERE id = $1
	`, id)

	return scanGroup(row)
}

func (r *GroupRepository) ExistsByName(ctx context.Context, name domainGroup.GroupName) (bool, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM groups WHERE LOWER(name) = LOWER($1))`,
		name.Value(),
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check group name existence", "error", err)
		return false, apperror.NewInternal()
	}

	return exists, nil
}

func (r *GroupRepository) FindDefault(ctx context.Context) (*domainGroup.Group, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	row := q.QueryRow(ctx, `
		SELECT id, name, description, visibility, join_policy,
		       is_default, created_by, created_at, updated_at
		FROM groups
		WHERE is_default = TRUE
		LIMIT 1
	`)

	return scanGroup(row)
}

func (r *GroupRepository) Delete(ctx context.Context, id string) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	tag, err := q.Exec(ctx, `DELETE FROM groups WHERE id = $1`, id)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete group", "error", err, "group_id", id)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
	}

	return nil
}

func (r *GroupRepository) List(ctx context.Context, filters domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	return []*domainGroup.Group{}, 0, nil
}

func scanGroup(row pgx.Row) (*domainGroup.Group, error) {
	var (
		id          string
		name        string
		description *string
		visibility  string
		joinPolicy  string
		isDefault   bool
		createdBy   string
		createdAt   time.Time
		updatedAt   time.Time
	)

	err := row.Scan(&id, &name, &description, &visibility, &joinPolicy,
		&isDefault, &createdBy, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		}
		slog.Error("failed to scan group row", "error", err)
		return nil, apperror.NewInternal()
	}

	return domainGroup.RestoreGroup(
		id,
		domainGroup.RestoreGroupName(name),
		description,
		domainGroup.RestoreVisibility(visibility),
		domainGroup.RestoreJoinPolicy(joinPolicy),
		isDefault,
		shared.RestoreUserID(createdBy),
		createdAt,
		updatedAt,
	), nil
}

// Verificación estática: GroupRepository implementa domainGroup.Repository.
var _ domainGroup.Repository = (*GroupRepository)(nil)
