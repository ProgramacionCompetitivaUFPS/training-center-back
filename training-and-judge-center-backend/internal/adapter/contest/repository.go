package contest

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Repository struct {
	db infraPostgres.Querier
}

func NewRepository(db infraPostgres.Querier) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, c *domainContest.Contest) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	_, err := q.Exec(ctx, `
		INSERT INTO contests (id, name, description, start_time, end_time,
			penalty, freeze_minutes, enable_post_contest, locked, show_team_members,
			group_id, owner_id, participation_mode, team_size_min, team_size_max,
			created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`,
		c.ID(), c.Name().Value(), c.Description(),
		c.StartTime(), c.EndTime(),
		c.Penalty().Value(), c.FreezeMinutes(), c.EnablePostContest(), c.Locked(), c.ShowTeamMembers(),
		c.GroupID().Value(), c.OwnerID().Value(),
		c.ParticipationMode().String(), c.TeamSize().Min(), c.TeamSize().Max(),
		c.CreatedAt(), c.UpdatedAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to insert contest", "error", err, "contest_id", c.ID())
		return apperror.NewInternal()
	}

	if err := r.upsertProblems(ctx, q, c.ID(), c.Problems()); err != nil {
		return err
	}
	return nil
}

func (r *Repository) Update(ctx context.Context, c *domainContest.Contest) error {
	q := infraPostgres.GetQuerier(ctx, r.db)

	tag, err := q.Exec(ctx, `
		UPDATE contests SET
			name=$2, description=$3, start_time=$4, end_time=$5,
			penalty=$6, freeze_minutes=$7, enable_post_contest=$8, locked=$9,
			show_team_members=$10, participation_mode=$11, team_size_min=$12, team_size_max=$13,
			updated_at=$14
		WHERE id=$1`,
		c.ID(), c.Name().Value(), c.Description(),
		c.StartTime(), c.EndTime(),
		c.Penalty().Value(), c.FreezeMinutes(), c.EnablePostContest(), c.Locked(),
		c.ShowTeamMembers(), c.ParticipationMode().String(), c.TeamSize().Min(), c.TeamSize().Max(),
		c.UpdatedAt(),
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to update contest", "error", err, "contest_id", c.ID())
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	// Replace contest_problems: delete all then reinsert.
	if _, err := q.Exec(ctx, `DELETE FROM contest_problems WHERE contest_id=$1`, c.ID()); err != nil {
		slog.ErrorContext(ctx, "failed to delete contest problems", "error", err, "contest_id", c.ID())
		return apperror.NewInternal()
	}
	if err := r.upsertProblems(ctx, q, c.ID(), c.Problems()); err != nil {
		return err
	}
	return nil
}

func (r *Repository) FindByID(ctx context.Context, id string) (*domainContest.Contest, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	var (
		cID, name, groupID, ownerID                string
		participationMode                          string
		description                                *string
		startTime, endTime, createdAt              time.Time
		updatedAt                                  *time.Time
		penalty, freezeMinutes                     int
		teamSizeMin, teamSizeMax                   int
		enablePostContest, locked, showTeamMembers bool
	)
	err := q.QueryRow(ctx, `
		SELECT id, name, description, start_time, end_time,
		       penalty, freeze_minutes, enable_post_contest, locked, show_team_members,
		       group_id, owner_id, participation_mode, team_size_min, team_size_max,
		       created_at, updated_at
		FROM contests WHERE id=$1`, id).Scan(
		&cID, &name, &description, &startTime, &endTime,
		&penalty, &freezeMinutes, &enablePostContest, &locked, &showTeamMembers,
		&groupID, &ownerID, &participationMode, &teamSizeMin, &teamSizeMax,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}
		slog.ErrorContext(ctx, "failed to find contest", "error", err, "contest_id", id)
		return nil, apperror.NewInternal()
	}

	problems, err := r.loadProblems(ctx, q, cID)
	if err != nil {
		return nil, err
	}

	return domainContest.RestoreContest(
		cID,
		domainContest.RestoreContestName(name),
		description,
		startTime, endTime,
		domainContest.RestorePenalty(penalty),
		freezeMinutes,
		enablePostContest,
		locked,
		showTeamMembers,
		shared.RestoreGroupID(groupID),
		shared.RestoreUserID(ownerID),
		domainContest.RestoreParticipationMode(participationMode),
		domainContest.RestoreTeamSize(teamSizeMin, teamSizeMax),
		problems,
		createdAt,
		updatedAt,
	), nil
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	q := infraPostgres.GetQuerier(ctx, r.db)
	tag, err := q.Exec(ctx, `DELETE FROM contests WHERE id=$1`, id)
	if err != nil {
		slog.ErrorContext(ctx, "failed to delete contest", "error", err, "contest_id", id)
		return apperror.NewInternal()
	}
	if tag.RowsAffected() == 0 {
		return apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}
	return nil
}

// sortColumn maps domain SortField values to safe SQL column names.
var sortColumn = map[domainContest.SortField]string{
	domainContest.SortByName:      "name",
	domainContest.SortByStartTime: "start_time",
	domainContest.SortByCreatedAt: "created_at",
}

func (r *Repository) List(ctx context.Context, filters domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return r.listContests(ctx, "group_id=$1", []interface{}{filters.GroupID.Value()}, filters)
}

// ListByGroupIDs lists contests scoped to groupIDs. groupIDs == nil omits the
// group filter entirely (Admins, "all contests"); groupIDs == []string{}
// matches nothing (no accessible groups).
func (r *Repository) ListByGroupIDs(ctx context.Context, groupIDs []string, filters domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	if groupIDs == nil {
		return r.listContests(ctx, "", nil, filters)
	}
	return r.listContests(ctx, "group_id = ANY($1)", []interface{}{groupIDs}, filters)
}

// listContests runs the shared list+count query. baseCond is an optional
// group-scoping condition (e.g. "group_id=$1" or "group_id = ANY($1)") whose
// placeholders are bound by baseArgs; pass "" / nil to omit group scoping.
func (r *Repository) listContests(ctx context.Context, baseCond string, baseArgs []interface{}, filters domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	q := infraPostgres.GetQuerier(ctx, r.db)

	var conds []string
	args := append([]interface{}{}, baseArgs...)
	if baseCond != "" {
		conds = append(conds, baseCond)
	}

	if filters.Status != nil {
		switch *filters.Status {
		case domainContest.StatusScheduled:
			conds = append(conds, "start_time > NOW()")
		case domainContest.StatusActive:
			conds = append(conds, "start_time <= NOW() AND end_time >= NOW()")
		case domainContest.StatusFinished:
			conds = append(conds, "end_time < NOW()")
		}
	}

	if filters.Search != "" {
		conds = append(conds, "name ILIKE $"+strconv.Itoa(len(args)+1))
		args = append(args, "%"+infraPostgres.EscapeILIKE(filters.Search)+"%")
	}

	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	// ORDER BY — whitelist prevents injection.
	col, ok := sortColumn[filters.SortBy]
	if !ok {
		col = "start_time"
	}
	order := "DESC"
	if filters.Order == domainContest.OrderAsc {
		order = "ASC"
	}

	limitPos := len(args) + 1
	offsetPos := len(args) + 2
	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)

	query := `SELECT id, name, description, start_time, end_time,
		       penalty, freeze_minutes, enable_post_contest, locked, show_team_members,
		       group_id, owner_id, participation_mode, team_size_min, team_size_max,
		       created_at, updated_at
		FROM contests ` + where +
		` ORDER BY ` + col + ` ` + order +
		` LIMIT $` + strconv.Itoa(limitPos) + ` OFFSET $` + strconv.Itoa(offsetPos)

	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "failed to list contests", "error", err)
		return nil, 0, apperror.NewInternal()
	}
	defer rows.Close()

	var result []*domainContest.Contest
	for rows.Next() {
		var (
			cID, name, groupID, ownerID                string
			participationMode                          string
			description                                *string
			startTime, endTime, createdAt              time.Time
			updatedAt                                  *time.Time
			penalty, freezeMinutes                     int
			teamSizeMin, teamSizeMax                   int
			enablePostContest, locked, showTeamMembers bool
		)
		if err := rows.Scan(
			&cID, &name, &description, &startTime, &endTime,
			&penalty, &freezeMinutes, &enablePostContest, &locked, &showTeamMembers,
			&groupID, &ownerID, &participationMode, &teamSizeMin, &teamSizeMax,
			&createdAt, &updatedAt,
		); err != nil {
			slog.ErrorContext(ctx, "failed to scan contest row", "error", err)
			return nil, 0, apperror.NewInternal()
		}
		result = append(result, domainContest.RestoreContest(
			cID,
			domainContest.RestoreContestName(name),
			description,
			startTime, endTime,
			domainContest.RestorePenalty(penalty),
			freezeMinutes,
			enablePostContest,
			locked,
			showTeamMembers,
			shared.RestoreGroupID(groupID),
			shared.RestoreUserID(ownerID),
			domainContest.RestoreParticipationMode(participationMode),
			domainContest.RestoreTeamSize(teamSizeMin, teamSizeMax),
			nil,
			createdAt,
			updatedAt,
		))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "list contests rows error", "error", err)
		return nil, 0, apperror.NewInternal()
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM contests " + where
	if err := q.QueryRow(ctx, countQuery, args[:len(args)-2]...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "failed to count contests", "error", err)
		return nil, 0, apperror.NewInternal()
	}

	if result == nil {
		result = []*domainContest.Contest{}
	}
	return result, total, nil
}

func (r *Repository) upsertProblems(ctx context.Context, q infraPostgres.Querier, contestID string, problems []domainContest.ContestProblem) error {
	if len(problems) == 0 {
		return nil
	}
	ids := make([]string, len(problems))
	problemIDs := make([]string, len(problems))
	orders := make([]int32, len(problems))
	for i, cp := range problems {
		ids[i] = cp.ID()
		problemIDs[i] = cp.ProblemID()
		orders[i] = int32(cp.Order())
	}
	_, err := q.Exec(ctx, `
		INSERT INTO contest_problems (id, contest_id, problem_id, "order")
		SELECT unnest($1::uuid[]), $2, unnest($3::uuid[]), unnest($4::int[])
		ON CONFLICT (contest_id, problem_id) DO UPDATE SET "order"=EXCLUDED."order"`,
		ids, contestID, problemIDs, orders,
	)
	if err != nil {
		slog.ErrorContext(ctx, "failed to upsert contest problems", "error", err, "contest_id", contestID, "count", len(problems))
		return apperror.NewInternal()
	}
	return nil
}

func (r *Repository) loadProblems(ctx context.Context, q infraPostgres.Querier, contestID string) ([]domainContest.ContestProblem, error) {
	rows, err := q.Query(ctx, `
		SELECT id, problem_id, "order"
		FROM contest_problems
		WHERE contest_id=$1
		ORDER BY "order"`, contestID)
	if err != nil {
		slog.ErrorContext(ctx, "failed to load contest problems", "error", err, "contest_id", contestID)
		return nil, apperror.NewInternal()
	}
	defer rows.Close()

	var problems []domainContest.ContestProblem
	for rows.Next() {
		var id, problemID string
		var order int
		if err := rows.Scan(&id, &problemID, &order); err != nil {
			slog.ErrorContext(ctx, "failed to scan contest problem", "error", err)
			return nil, apperror.NewInternal()
		}
		problems = append(problems, domainContest.RestoreContestProblem(id, problemID, order))
	}
	if err := rows.Err(); err != nil {
		slog.ErrorContext(ctx, "contest problems rows error", "error", err)
		return nil, apperror.NewInternal()
	}
	return problems, nil
}
