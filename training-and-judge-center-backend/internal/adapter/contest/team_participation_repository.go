package contest

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TeamParticipationRepository struct {
	db infraPostgres.Querier
}

func NewTeamParticipationRepository(db infraPostgres.Querier) *TeamParticipationRepository {
	return &TeamParticipationRepository{db: db}
}

func (r *TeamParticipationRepository) Save(ctx context.Context, p *domainContest.ContestTeamParticipation) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	_, err := q.Exec(ctx,
		`INSERT INTO contest_team_participants (id, contest_id, team_id, selected_members, registered_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		p.ID(), p.ContestID(), p.TeamID(), p.SelectedMembers(), p.RegisteredAt(),
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == infraPostgres.UniqueViolation {
			return apperror.NewConflict(domainContest.ErrCodeTeamAlreadyRegistered, "team is already registered to this contest")
		}
		slog.ErrorContext(ctx, "failed to save contest team participation",
			"contest_id", p.ContestID(), "team_id", p.TeamID(), "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (r *TeamParticipationRepository) FindByContestAndTeam(ctx context.Context, contestID, teamID string) (*domainContest.ContestTeamParticipation, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	var id, cID, tID string
	var selectedMembers []string
	var registeredAt time.Time
	err := q.QueryRow(ctx,
		`SELECT id, contest_id, team_id, selected_members, registered_at
		 FROM contest_team_participants
		 WHERE contest_id=$1 AND team_id=$2`,
		contestID, teamID,
	).Scan(&id, &cID, &tID, &selectedMembers, &registeredAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeTeamRegistrationNotFound, "team registration not found")
		}
		slog.ErrorContext(ctx, "failed to find team participation",
			"contest_id", contestID, "team_id", teamID, "error", err)
		return nil, apperror.NewInternal()
	}
	return domainContest.RestoreContestTeamParticipation(id, cID, tID, selectedMembers, registeredAt), nil
}

func (r *TeamParticipationRepository) Update(ctx context.Context, p *domainContest.ContestTeamParticipation) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx,
		`UPDATE contest_team_participants SET selected_members=$3
		 WHERE contest_id=$1 AND team_id=$2`,
		p.ContestID(), p.TeamID(), p.SelectedMembers(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update team participation",
			"contest_id", p.ContestID(), "team_id", p.TeamID(), "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainContest.ErrCodeTeamRegistrationNotFound, "team registration not found")
	}
	return nil
}

func (r *TeamParticipationRepository) Delete(ctx context.Context, contestID, teamID string) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx,
		`DELETE FROM contest_team_participants WHERE contest_id=$1 AND team_id=$2`,
		contestID, teamID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete team participation",
			"contest_id", contestID, "team_id", teamID, "error", err)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainContest.ErrCodeTeamRegistrationNotFound, "team registration not found")
	}
	return nil
}

func (r *TeamParticipationRepository) List(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestTeamParticipation, int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	var total int
	if err := q.QueryRow(ctx,
		`SELECT COUNT(*) FROM contest_team_participants WHERE contest_id=$1`,
		contestID,
	).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "failed to count team participations", "contest_id", contestID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	if total == 0 {
		return []*domainContest.ContestTeamParticipation{}, 0, nil
	}

	offset := (page - 1) * limit
	rows, err := q.Query(ctx,
		`SELECT id, contest_id, team_id, selected_members, registered_at
		 FROM contest_team_participants
		 WHERE contest_id=$1
		 ORDER BY registered_at ASC
		 LIMIT $2 OFFSET $3`,
		contestID, limit, offset,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list team participations", "contest_id", contestID, "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	var result []*domainContest.ContestTeamParticipation
	for rows.Next() {
		var id, cID, tID string
		var selectedMembers []string
		var registeredAt time.Time
		if err := rows.Scan(&id, &cID, &tID, &selectedMembers, &registeredAt); err != nil {
			slog.ErrorContext(ctx, "failed to scan team participation row", "error", err)
			return nil, 0, apperror.NewInternal()
		}
		result = append(result, domainContest.RestoreContestTeamParticipation(id, cID, tID, selectedMembers, registeredAt))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "team participations rows error", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	return result, total, nil
}

func (r *TeamParticipationRepository) ListSelectedMembersByContest(ctx context.Context, contestID string) (map[string][]string, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)
	rows, err := q.Query(ctx,
		`SELECT team_id, selected_members FROM contest_team_participants WHERE contest_id = $1`,
		contestID,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list team participants for standings", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var teamID string
		var members []string
		if err := rows.Scan(&teamID, &members); err != nil {
			slog.ErrorContext(ctx, "failed to scan team participant row", "error", err)
			return nil, apperror.NewInternal()
		}
		result[teamID] = members
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "team participants rows error", "contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}

func (r *TeamParticipationRepository) AreUsersSelectedInAnyTeam(ctx context.Context, contestID string, userIDs []string, excludeTeamID string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	if len(userIDs) == 0 {
		return result, nil
	}

	q := infraPostgres.GetQuerier(ctx, r.db)

	var rows pgx.Rows
	var err error
	if excludeTeamID == "" {
		rows, err = q.Query(ctx,
			`SELECT DISTINCT unnest(selected_members)
			 FROM contest_team_participants
			 WHERE contest_id=$1 AND selected_members && $2::UUID[]`,
			contestID, userIDs,
		)
	} else {
		rows, err = q.Query(ctx,
			`SELECT DISTINCT unnest(selected_members)
			 FROM contest_team_participants
			 WHERE contest_id=$1 AND team_id<>$2 AND selected_members && $3::UUID[]`,
			contestID, excludeTeamID, userIDs,
		)
	}
	if err != nil {
		slog.ErrorContext(ctx, "failed to check users selected in teams",
			"contest_id", contestID, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	// Build a set of requested IDs for O(1) lookup
	requested := make(map[string]struct{}, len(userIDs))
	for _, id := range userIDs {
		requested[id] = struct{}{}
	}

	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			slog.ErrorContext(ctx, "failed to scan selected member row", "error", err)
			return nil, apperror.NewInternal()
		}
		if _, ok := requested[userID]; ok {
			result[userID] = true
		}
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "selected members rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return result, nil
}
