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

func mustJSON(s string) []byte { return []byte(s) }

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

// ── FileStorage mock ─────────────────────────────────────────────────────────

type mockFileStorageH struct{}

func (m *mockFileStorageH) UploadFile(_ context.Context, _ string, _ []byte) error { return nil }
func (m *mockFileStorageH) DeleteFile(_ context.Context, _ string) error            { return nil }
func (m *mockFileStorageH) DeleteFilesWithPrefix(_ context.Context, _ string) error { return nil }

// ── ActiveContestChecker mock ────────────────────────────────────────────────

type mockActiveContestCheckerH struct {
	inActive bool
	err      error
}

func (m *mockActiveContestCheckerH) IsProblemInActiveContest(_ context.Context, _ string) (bool, error) {
	return m.inActive, m.err
}

// ── SubmissionRejudger mock ──────────────────────────────────────────────────

type mockSubmissionRejudgerH struct {
	subs        []appProblem.SubmissionRejudgeInfo
	listErr     error
	contestSubs []appProblem.SubmissionRejudgeInfo
	contestErr  error
}

func (m *mockSubmissionRejudgerH) ListByProblemBefore(_ context.Context, _ string, _ time.Time) ([]appProblem.SubmissionRejudgeInfo, error) {
	return m.subs, m.listErr
}

func (m *mockSubmissionRejudgerH) ListByProblemAndContestBefore(_ context.Context, _, _ string, _ time.Time) ([]appProblem.SubmissionRejudgeInfo, error) {
	return m.contestSubs, m.contestErr
}

func (m *mockSubmissionRejudgerH) RejudgeBatch(_ context.Context, subs []appProblem.SubmissionRejudgeInfo, _ string, _ time.Time) (int, error) {
	return len(subs), nil
}

// ── UserProvider mock ────────────────────────────────────────────────────────

type mockUserProviderH struct {
	getIDByNicknameFn func(ctx context.Context, nickname string) (string, bool, error)
}

func (m *mockUserProviderH) ExistsByID(_ context.Context, _ string) (bool, error) { return true, nil }
func (m *mockUserProviderH) GetDisplay(_ context.Context, userID string) (*appProblem.UserDisplay, error) {
	return &appProblem.UserDisplay{Nickname: "user_" + userID, Name: "Test User"}, nil
}
func (m *mockUserProviderH) GetDisplays(_ context.Context, userIDs []string) (map[string]*appProblem.UserDisplay, error) {
	out := make(map[string]*appProblem.UserDisplay, len(userIDs))
	for _, id := range userIDs {
		out[id] = &appProblem.UserDisplay{Nickname: "user_" + id, Name: "Test User"}
	}
	return out, nil
}
func (m *mockUserProviderH) GetIDByNickname(ctx context.Context, nickname string) (string, bool, error) {
	if m.getIDByNicknameFn != nil {
		return m.getIDByNicknameFn(ctx, nickname)
	}
	return "", false, nil
}

func userProviderResolvingH(userID string) *mockUserProviderH {
	return &mockUserProviderH{
		getIDByNicknameFn: func(_ context.Context, _ string) (string, bool, error) {
			return userID, true, nil
		},
	}
}

// ── ContestRejudgeProvider mock ──────────────────────────────────────────────

type mockContestRejudgeProviderH struct {
	contest            *appProblem.ContestRejudgeInfo
	isLeadOfGroup      bool
	isProblemInContest bool
}

func (m *mockContestRejudgeProviderH) GetContestForRejudge(_ context.Context, _ string) (*appProblem.ContestRejudgeInfo, error) {
	return m.contest, nil
}
func (m *mockContestRejudgeProviderH) IsProblemInContest(_ context.Context, _, _ string) (bool, error) {
	return m.isProblemInContest, nil
}
func (m *mockContestRejudgeProviderH) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return m.isLeadOfGroup, nil
}

// ── Handler constructor helpers ──────────────────────────────────────────────

func newHandlerWithStatistics(uc *appProblem.GetProblemStatisticsUseCase) *Handler {
	return &Handler{getProblemStatistics: uc}
}

func newHandlerWithUnpublish(repo domainProblem.Repository, checker appProblem.ActiveContestChecker) *Handler {
	return &Handler{unpublishProblem: appProblem.NewUnpublishProblemUseCase(repo, checker)}
}

func newHandlerWithDeleteProblem(repo domainProblem.Repository, checker appProblem.ActiveContestChecker) *Handler {
	return &Handler{deleteProblem: appProblem.NewDeleteProblemUseCase(repo, &mockFileStorageH{}, checker)}
}

func newHandlerWithChangeAccessibility(repo domainProblem.Repository) *Handler {
	return &Handler{changeAccessibility: appProblem.NewChangeAccessibilityUseCase(repo)}
}

func newHandlerWithRejudge(repo domainProblem.Repository, rejudger appProblem.SubmissionRejudger) *Handler {
	return &Handler{rejudgeSubmissions: appProblem.NewRejudgeSubmissionsUseCase(repo, rejudger)}
}

func newHandlerWithAddModifier(repo domainProblem.Repository, userProvider appProblem.UserProvider) *Handler {
	return &Handler{addModifier: appProblem.NewAddModifierUseCase(repo, userProvider)}
}

func newHandlerWithRemoveModifier(repo domainProblem.Repository, userProvider appProblem.UserProvider) *Handler {
	return &Handler{removeModifier: appProblem.NewRemoveModifierUseCase(repo, userProvider)}
}

func newHandlerWithRejudgeContest(repo domainProblem.Repository, rejudger appProblem.SubmissionRejudger, contestProvider appProblem.ContestRejudgeProvider) *Handler {
	return &Handler{rejudgeContestSubmissions: appProblem.NewRejudgeContestSubmissionsUseCase(repo, rejudger, contestProvider)}
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

func draftProblemWithAuthor(authorID string) *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"DRAFT", "PRIVATE",
		shared.RestoreUserID(authorID),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func draftProblemWithModifier(modifierID string) *domainProblem.Problem {
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"DRAFT", "PRIVATE",
		shared.RestoreUserID("u1"),
		[]shared.UserID{shared.RestoreUserID(modifierID)},
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
