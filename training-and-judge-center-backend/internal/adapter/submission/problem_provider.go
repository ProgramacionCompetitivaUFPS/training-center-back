package submission

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemProvider struct {
	db infraPostgres.Querier
}

func NewProblemProvider(db infraPostgres.Querier) *ProblemProvider {
	return &ProblemProvider{db: db}
}

func (p *ProblemProvider) GetProblemBySlug(ctx context.Context, slug string) (*appSubmission.ProblemInfo, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var id, authorID, pSlug, title, status, accessibility string
	var modifierIDs []string
	var testCasesKey *string

	err := q.QueryRow(ctx, `
		SELECT id, author_id, slug, title, status, accessibility, modifiers_ids, test_cases_key
		FROM problems
		WHERE slug = $1
	`, slug).Scan(&id, &authorID, &pSlug, &title, &status, &accessibility, &modifierIDs, &testCasesKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "problem not found")
		}
		slog.ErrorContext(ctx, "submission: failed to get problem by slug", "slug", slug, "error", err)
		return nil, apperror.NewInternal()
	}

	modIDs := make([]string, len(modifierIDs))
	copy(modIDs, modifierIDs)

	return &appSubmission.ProblemInfo{
		ID:           id,
		AuthorID:     authorID,
		Slug:         pSlug,
		Title:        title,
		IsPublished:  status == "PUBLISHED",
		IsPublic:     accessibility == "PUBLIC",
		ModifierIDs:  modIDs,
		HasTestCases: testCasesKey != nil,
	}, nil
}
