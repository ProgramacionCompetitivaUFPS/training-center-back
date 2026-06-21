package problem

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Auth mock ────────────────────────────────────────────────────────────────

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: shared.RestoreUserID("u1").Value(), Role: shared.RoleCoach}, nil
}

func authedRequest(method, target string, body []byte) *http.Request {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, target, bytes.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, target, nil)
	}
	r.Header.Set("Authorization", "Bearer tok")
	return r
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, nil)(h)
}

// ── Problem repository mock ──────────────────────────────────────────────────

type mockProblemRepo struct {
	findBySlugFn func(ctx context.Context, slug domainProblem.Slug) (*domainProblem.Problem, error)
}

func (m *mockProblemRepo) Save(_ context.Context, _ *domainProblem.Problem) error { return nil }
func (m *mockProblemRepo) FindBySlug(ctx context.Context, slug domainProblem.Slug) (*domainProblem.Problem, error) {
	if m.findBySlugFn != nil {
		return m.findBySlugFn(ctx, slug)
	}
	return nil, apperror.NewNotFound(domainProblem.ErrCodeProblemNotFound, "not found")
}
func (m *mockProblemRepo) ExistsBySlug(_ context.Context, _ domainProblem.Slug) (bool, error) {
	return false, nil
}
func (m *mockProblemRepo) List(_ context.Context, _ domainProblem.ListFilters) ([]*domainProblem.Problem, int, error) {
	return nil, 0, nil
}
func (m *mockProblemRepo) Delete(_ context.Context, _ string) error { return nil }

// ── StatisticsProvider mock ──────────────────────────────────────────────────

type mockStatsProvider struct {
	stats *appProblem.ProblemStats
	err   error
}

func (m *mockStatsProvider) GetByProblemID(_ context.Context, _ string) (*appProblem.ProblemStats, error) {
	return m.stats, m.err
}

// ── Handler constructor helpers ──────────────────────────────────────────────

func newHandlerWithStatistics(uc *appProblem.GetProblemStatisticsUseCase) *Handler {
	return &Handler{getStatisticsUC: uc}
}

// ── Problem fixtures ─────────────────────────────────────────────────────────

var testNow = time.Now()

func publishedProblem() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID("u1"),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func draftProblem() *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"DRAFT", "PRIVATE",
		shared.RestoreUserID("u1"),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func repoReturning(p *domainProblem.Problem) *mockProblemRepo {
	return &mockProblemRepo{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return p, nil
		},
	}
}
