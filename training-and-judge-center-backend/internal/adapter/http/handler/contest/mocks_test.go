package contest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Auth mock ────────────────────────────────────────────────────────────────

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ *domainUser.User) (string, error) { return "tok", nil }
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

// ── Stub dependencies ────────────────────────────────────────────────────────

type stubContestRepo struct {
	createFn func(ctx context.Context, c *domainContest.Contest) error
}

func (s *stubContestRepo) Create(ctx context.Context, c *domainContest.Contest) error {
	if s.createFn != nil {
		return s.createFn(ctx, c)
	}
	return nil
}
func (s *stubContestRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *stubContestRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "not found")
}
func (s *stubContestRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubContestRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

type stubGroupProvider struct{}

func (s *stubGroupProvider) FindByID(_ context.Context, groupID string) (*appcontest.GroupInfo, error) {
	return &appcontest.GroupInfo{ID: groupID, Name: "Test Group"}, nil
}

type stubMemberProvider struct{ isLead bool }

func (s *stubMemberProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return s.isLead, nil
}

type stubProblemProvider struct{}

func (s *stubProblemProvider) FindBySlugs(_ context.Context, slugs []string, _ string, _ bool) (map[string]*appcontest.ProblemInfo, error) {
	result := make(map[string]*appcontest.ProblemInfo, len(slugs))
	for _, sl := range slugs {
		result[sl] = &appcontest.ProblemInfo{ID: "p1", Slug: sl, Title: sl, IsPublished: true, CanAdd: true}
	}
	return result, nil
}
func (s *stubProblemProvider) FindByIDs(_ context.Context, ids []string) (map[string]*appcontest.ProblemBasicInfo, error) {
	result := make(map[string]*appcontest.ProblemBasicInfo, len(ids))
	for _, id := range ids {
		result[id] = &appcontest.ProblemBasicInfo{ID: id, Slug: "slug", Title: "title"}
	}
	return result, nil
}

type stubOwnerProvider struct{}

func (s *stubOwnerProvider) GetDisplay(_ context.Context, _ string) (*appcontest.UserDisplay, error) {
	return &appcontest.UserDisplay{Nickname: "coach", Name: "Coach"}, nil
}
