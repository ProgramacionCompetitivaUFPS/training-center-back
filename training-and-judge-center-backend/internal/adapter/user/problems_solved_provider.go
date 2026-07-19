package user

import (
	"context"
	"log/slog"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemsSolvedProvider struct {
	db infraPostgres.Querier
}

var _ appuser.ProblemsSolvedProvider = (*ProblemsSolvedProvider)(nil)

func NewProblemsSolvedProvider(db infraPostgres.Querier) *ProblemsSolvedProvider {
	return &ProblemsSolvedProvider{db: db}
}

// GetProblemsSolved computes the user's count of unique problems with at
// least one ACCEPTED submission.
//
// This duplicates the "unique problems solved" computation embedded in
// adapter/user/ranking_provider.go (GetRanking) — kept separate on purpose so
// this cheap query doesn't pay the cost of the ranking CTE. If the definition
// of "solved" ever changes, update both.
func (p *ProblemsSolvedProvider) GetProblemsSolved(ctx context.Context, userID string) (int, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	row := q.QueryRow(ctx, `
		SELECT COUNT(DISTINCT problem_id)::INTEGER
		FROM submissions
		WHERE user_id = $1 AND status = 'ACCEPTED'`,
		userID,
	)

	var problemsSolved int
	if err := row.Scan(&problemsSolved); err != nil {
		slog.ErrorContext(ctx, "dashboard: failed to query problems solved", "user_id", userID, "error", err)
		return 0, apperror.NewInternal()
	}
	return problemsSolved, nil
}
