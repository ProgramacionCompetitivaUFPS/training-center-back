package contest

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
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
func (s *mockContestRepo) ListByGroupIDs(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

type mockGroupProvider struct{}

func (s *mockGroupProvider) FindByID(_ context.Context, groupID string) (*appcontest.GroupInfo, error) {
	return &appcontest.GroupInfo{ID: groupID, Name: "Test Group", IsVisible: true}, nil
}
func (s *mockGroupProvider) FindByIDs(_ context.Context, groupIDs []string) (map[string]*appcontest.GroupInfo, error) {
	out := make(map[string]*appcontest.GroupInfo, len(groupIDs))
	for _, id := range groupIDs {
		out[id] = &appcontest.GroupInfo{ID: id, Name: "Test Group", IsVisible: true}
	}
	return out, nil
}
func (s *mockGroupProvider) ListAccessibleGroupIDs(_ context.Context, _ string, isAdmin bool) ([]string, error) {
	if isAdmin {
		return nil, nil
	}
	return []string{}, nil
}

type mockMemberProvider struct {
	isLead   bool
	isMember bool
}

func (s *mockMemberProvider) GetMemberRole(_ context.Context, _, _ string) (*string, error) {
	if s.isLead {
		role := "LEAD"
		return &role, nil
	}
	if s.isMember {
		role := "MEMBER"
		return &role, nil
	}
	return nil, nil
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

type mockTeamParticipantProvider struct{}

func (m *mockTeamParticipantProvider) ListSelectedMembersByContest(_ context.Context, _ string) (map[string][]string, error) {
	return map[string][]string{}, nil
}

type mockParticipantProfileProvider struct{}

func (m *mockParticipantProfileProvider) GetProfiles(_ context.Context, _ []string) (map[string]*appcontest.ParticipantProfile, error) {
	return map[string]*appcontest.ParticipantProfile{}, nil
}

type mockTeamParticipRepo struct{}

func (s *mockTeamParticipRepo) Save(_ context.Context, _ *domainContest.ContestTeamParticipation) error {
	return nil
}
func (s *mockTeamParticipRepo) FindByContestAndTeam(_ context.Context, _, _ string) (*domainContest.ContestTeamParticipation, error) {
	return nil, nil
}
func (s *mockTeamParticipRepo) Update(_ context.Context, _ *domainContest.ContestTeamParticipation) error {
	return nil
}
func (s *mockTeamParticipRepo) Delete(_ context.Context, _, _ string) error { return nil }
func (s *mockTeamParticipRepo) List(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestTeamParticipation, int, error) {
	return []*domainContest.ContestTeamParticipation{}, 0, nil
}
func (s *mockTeamParticipRepo) AreUsersSelectedInAnyTeam(_ context.Context, _ string, userIDs []string, _ string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	return result, nil
}

// ── RegistrationRepository mock ──────────────────────────────────────────────

type mockRegistrationRepository struct {
	saveFn                   func(ctx context.Context, r *domainContest.ContestRegistration) error
	findByContestAndUserFn   func(ctx context.Context, contestID, userID string) (*domainContest.ContestRegistration, error)
	deleteFn                 func(ctx context.Context, contestID, userID string) error
	listByContestFn          func(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestRegistration, int, error)
	existsByContestAndUserFn func(ctx context.Context, contestID, userID string) (bool, error)
}

func (m *mockRegistrationRepository) Save(ctx context.Context, r *domainContest.ContestRegistration) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, r)
	}
	return nil
}

func (m *mockRegistrationRepository) FindByContestAndUser(ctx context.Context, contestID, userID string) (*domainContest.ContestRegistration, error) {
	if m.findByContestAndUserFn != nil {
		return m.findByContestAndUserFn(ctx, contestID, userID)
	}
	return nil, nil
}

func (m *mockRegistrationRepository) Delete(ctx context.Context, contestID, userID string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, contestID, userID)
	}
	return nil
}

func (m *mockRegistrationRepository) ListByContest(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestRegistration, int, error) {
	if m.listByContestFn != nil {
		return m.listByContestFn(ctx, contestID, page, limit)
	}
	return []*domainContest.ContestRegistration{}, 0, nil
}

func (m *mockRegistrationRepository) ExistsByContestAndUser(ctx context.Context, contestID, userID string) (bool, error) {
	if m.existsByContestAndUserFn != nil {
		return m.existsByContestAndUserFn(ctx, contestID, userID)
	}
	return false, nil
}

func (m *mockRegistrationRepository) CountByContest(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (m *mockRegistrationRepository) CountByContestBulk(_ context.Context, contestIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	return result, nil
}

func (m *mockRegistrationRepository) ExistsByUserBulk(_ context.Context, contestIDs []string, _ string) (map[string]bool, error) {
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = false
	}
	return result, nil
}

// ── ParticipantNicknameProvider mock ─────────────────────────────────────────

type mockNicknameProvider struct{}

func (s *mockNicknameProvider) GetNicknamesByIDs(_ context.Context, userIDs []string) (map[string]string, error) {
	result := make(map[string]string, len(userIDs))
	for _, id := range userIDs {
		result[id] = "nick_" + id
	}
	return result, nil
}

// ── ContestSubmissionsProvider mock ──────────────────────────────────────────

type mockContestSubmissionsProvider struct {
	subs []appcontest.RichSubmissionData
}

func (m *mockContestSubmissionsProvider) ListByContest(_ context.Context, _ string, _ appcontest.ContestSubmissionFilters) ([]appcontest.RichSubmissionData, error) {
	return m.subs, nil
}

// ── TeamSelectionChecker no-op ───────────────────────────────────────────────

type noopTeamSelectionChecker struct{}

func (n *noopTeamSelectionChecker) IsUserSelectedInAnyTeam(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// ── CallerStandingProvider mock ───────────────────────────────────────────────

type mockCallerStandingProvider struct{}

func (m *mockCallerStandingProvider) GetCallerStandingID(_ context.Context, _, userID string) (string, bool, error) {
	return userID, true, nil
}

// ── Contest fixture helpers ───────────────────────────────────────────────────

func scheduledContest() *domainContest.Contest {
	start := time.Now().Add(24 * time.Hour)
	end := time.Now().Add(29 * time.Hour)
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Test Contest"),
		nil,
		start, end,
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(),
		nil,
	)
}

func finishedContest() *domainContest.Contest {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Finished Contest"),
		nil,
		time.Now().Add(-48*time.Hour),
		time.Now().Add(-24*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now(),
		nil,
	)
}
