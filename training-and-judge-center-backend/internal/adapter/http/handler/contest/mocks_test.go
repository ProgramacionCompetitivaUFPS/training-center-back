package contest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
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

// ── Mock dependencies ────────────────────────────────────────────────────────

type mockContestRepo struct {
	createFn func(ctx context.Context, c *domainContest.Contest) error
}

func (s *mockContestRepo) Create(ctx context.Context, c *domainContest.Contest) error {
	if s.createFn != nil {
		return s.createFn(ctx, c)
	}
	return nil
}
func (s *mockContestRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *mockContestRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "not found")
}
func (s *mockContestRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *mockContestRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

type mockGroupProvider struct{}

func (s *mockGroupProvider) FindByID(_ context.Context, groupID string) (*appcontest.GroupInfo, error) {
	return &appcontest.GroupInfo{ID: groupID, Name: "Test Group", IsVisible: true}, nil
}

type mockMemberProvider struct {
	isLead   bool
	isMember bool
}

func (s *mockMemberProvider) IsLeadOfGroup(_ context.Context, _, _ string) (bool, error) {
	return s.isLead, nil
}

func (s *mockMemberProvider) IsMemberOfGroup(_ context.Context, _, _ string) (bool, error) {
	return s.isMember, nil
}

type mockProblemProvider struct{}

func (s *mockProblemProvider) FindBySlugs(_ context.Context, slugs []string, _ string, _ bool) (map[string]*appcontest.ProblemInfo, error) {
	result := make(map[string]*appcontest.ProblemInfo, len(slugs))
	for _, sl := range slugs {
		result[sl] = &appcontest.ProblemInfo{ID: "p1", Slug: sl, Title: sl, IsPublished: true, CanAdd: true}
	}
	return result, nil
}
func (s *mockProblemProvider) FindByIDs(_ context.Context, ids []string) (map[string]*appcontest.ProblemBasicInfo, error) {
	result := make(map[string]*appcontest.ProblemBasicInfo, len(ids))
	for _, id := range ids {
		result[id] = &appcontest.ProblemBasicInfo{ID: id, Slug: "slug", Title: "title"}
	}
	return result, nil
}

func (s *mockProblemProvider) FindByIDsWithLimits(_ context.Context, ids []string) (map[string]*appcontest.ProblemWithLimits, error) {
	result := make(map[string]*appcontest.ProblemWithLimits, len(ids))
	for _, id := range ids {
		result[id] = &appcontest.ProblemWithLimits{ID: id, Slug: "slug", Title: "title", TimeLimit: 1000, MemoryLimit: 256}
	}
	return result, nil
}

type mockContestParticipantProvider struct{}

func (s *mockContestParticipantProvider) IsRegistered(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}
func (s *mockContestParticipantProvider) CountParticipants(_ context.Context, _ string) (int, error) {
	return 0, nil
}
func (s *mockContestParticipantProvider) CountParticipantsBulk(_ context.Context, contestIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	return result, nil
}
func (s *mockContestParticipantProvider) IsRegisteredBulk(_ context.Context, contestIDs []string, _ string) (map[string]bool, error) {
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = false
	}
	return result, nil
}

type mockOwnerProvider struct{}

func (s *mockOwnerProvider) GetDisplay(_ context.Context, userID string) (*appcontest.UserDisplay, error) {
	return &appcontest.UserDisplay{ID: userID, Nickname: "coach", Name: "Coach"}, nil
}

type mockTransactionManager struct{}

func (s *mockTransactionManager) WithTx(_ context.Context, fn func(context.Context) error) error {
	return fn(context.Background())
}
