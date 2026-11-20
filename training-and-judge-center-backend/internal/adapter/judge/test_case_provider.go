package judge

import (
	"context"
	"errors"
	"log/slog"

	"cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type TestCaseProvider struct {
	cache *testDataCache
	db    infraPostgres.Querier
}

var _ appJudge.TestCaseProvider = (*TestCaseProvider)(nil)

func NewTestCaseProvider(client *storage.Client, bucket string, db infraPostgres.Querier, cfg TestDataConfig) (*TestCaseProvider, error) {
	cache, err := newTestDataCache(cfg, newSizedGCSReader(client, bucket))
	if err != nil {
		return nil, err
	}
	return &TestCaseProvider{cache: cache, db: db}, nil
}

func (p *TestCaseProvider) Open(ctx context.Context, problemID string) (appJudge.TestCaseSet, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var testCasesKey *string
	err := q.QueryRow(ctx, `SELECT test_cases_key FROM problems WHERE id = $1`, problemID).Scan(&testCasesKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "test_case_provider: query failed", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}

	if testCasesKey == nil || *testCasesKey == "" {
		return appJudge.NewMemoryTestCaseSet(nil), nil
	}

	return p.OpenAt(ctx, *testCasesKey)
}

func (p *TestCaseProvider) OpenAt(ctx context.Context, testCasesKey string) (appJudge.TestCaseSet, error) {
	zipKey := appProblem.TestCasesZipKey(testCasesKey)

	entry, err := p.cache.acquire(ctx, zipKey)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.ErrorContext(ctx, "test_case_provider: ZIP not found in storage", "key", zipKey)
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "test cases not found")
		}
		slog.ErrorContext(ctx, "test_case_provider: could not get the test case ZIP", "key", zipKey, "error", err)
		return nil, apperror.NewInternal()
	}

	return newFileTestCaseSet(entry.cases, func() { p.cache.release(entry) }), nil
}
