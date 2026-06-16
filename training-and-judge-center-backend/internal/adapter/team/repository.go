package team

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository struct {
	db infraPostgres.Querier
}

func NewRepository(db infraPostgres.Querier) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Save(ctx context.Context, t *domainTeam.Team) error {
	const q = `INSERT INTO teams (id, name, created_by, created_at) VALUES ($1, $2, $3, $4)`
	_, err := infraPostgres.GetQuerier(ctx, r.db).Exec(ctx, q,
		t.ID(), t.Name().Value(), t.CreatedBy().Value(), t.CreatedAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation && pgErr.ConstraintName == "teams_name_lower_idx" {
			return apperror.NewConflict(domainTeam.ErrCodeTeamNameExists, "A team with this name already exists")
		}
		slog.ErrorContext(ctx, "failed to save team", "error", err, "team_id", t.ID())
		return apperror.NewInternal()
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*domainTeam.Team, error) {
	const q = `SELECT id, name, created_by, created_at FROM teams WHERE id = $1`
	var tid, name, createdBy string
	var createdAt time.Time
	err := infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx, q, id).Scan(&tid, &name, &createdBy, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainTeam.ErrCodeTeamNotFound, "team not found")
		}
		slog.ErrorContext(ctx, "failed to find team by ID", "error", err, "team_id", id)
		return nil, apperror.NewInternal()
	}
	return domainTeam.RestoreTeam(tid, domainTeam.RestoreTeamName(name), shared.RestoreUserID(createdBy), createdAt), nil
}

func (r *Repository) ExistsByName(ctx context.Context, name domainTeam.TeamName) (bool, error) {
	var exists bool
	err := infraPostgres.GetQuerier(ctx, r.db).QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM teams WHERE LOWER(name) = LOWER($1))`,
		name.Value(),
	).Scan(&exists)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check team name existence", "error", err)
		return false, apperror.NewInternal()
	}
	return exists, nil
}
