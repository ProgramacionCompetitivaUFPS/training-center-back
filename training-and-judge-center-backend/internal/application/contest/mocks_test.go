package contest

import (
	"context"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// testNow is a fixed anchor for createdAt/updatedAt in fixtures (not used for validation).
var testNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// testStart/testEnd are relative to the real clock so time-validation tests
// pass regardless of when tests run.
var (
	testStart = time.Now().Add(24 * time.Hour)
	testEnd   = time.Now().Add(29 * time.Hour)
)

const (
	testCallerID  = "aaaaaaaa-0000-0000-0000-000000000001"
	testOtherID   = "aaaaaaaa-0000-0000-0000-000000000002"
	testGroupID   = "bbbbbbbb-0000-0000-0000-000000000001"
	testContestID = "cccccccc-0000-0000-0000-000000000001"
	testProblemID = "dddddddd-0000-0000-0000-000000000001"
)

// ── CurrentUser helpers ──────────────────────────────────────────────────────

func asAdmin(id string) appshared.CurrentUser      { return appshared.CurrentUser{ID: id, Role: shared.RoleAdmin} }
func asCoach(id string) appshared.CurrentUser      { return appshared.CurrentUser{ID: id, Role: shared.RoleCoach} }
func asContestant(id string) appshared.CurrentUser { return appshared.CurrentUser{ID: id, Role: shared.RoleContestant} }

// ── Repository mock ──────────────────────────────────────────────────────────

type mockContestRepository struct {
	createFn  func(ctx context.Context, c *domainContest.Contest) error
	updateFn  func(ctx context.Context, c *domainContest.Contest) error
	findByIDFn func(ctx context.Context, id string) (*domainContest.Contest, error)
	deleteFn  func(ctx context.Context, id string) error
	listFn    func(ctx context.Context, f domainContest.ListFilters) ([]*domainContest.Contest, int, error)
}

func (m *mockContestRepository) Create(ctx context.Context, c *domainContest.Contest) error {
	if m.createFn != nil {
		return m.createFn(ctx, c)
	}
	return nil
}
func (m *mockContestRepository) Update(ctx context.Context, c *domainContest.Contest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, c)
	}
	return nil
}
func (m *mockContestRepository) FindByID(ctx context.Context, id string) (*domainContest.Contest, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
}
func (m *mockContestRepository) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}
func (m *mockContestRepository) List(ctx context.Context, f domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, f)
	}
	return []*domainContest.Contest{}, 0, nil
}

// ── GroupProvider mock ───────────────────────────────────────────────────────

type mockGroupProvider struct {
	findByIDFn func(ctx context.Context, groupID string) (*GroupInfo, error)
}

func (m *mockGroupProvider) FindByID(ctx context.Context, groupID string) (*GroupInfo, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, groupID)
	}
	return &GroupInfo{ID: groupID, Name: "Test Group"}, nil
}

func groupFound() *mockGroupProvider  { return &mockGroupProvider{} }
func groupNotFound() *mockGroupProvider {
	return &mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) {
		return nil, nil
	}}
}

// ── GroupMemberProvider mock ─────────────────────────────────────────────────

type mockGroupMemberProvider struct {
	isLeadFn func(ctx context.Context, userID, groupID string) (bool, error)
}

func (m *mockGroupMemberProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	if m.isLeadFn != nil {
		return m.isLeadFn(ctx, userID, groupID)
	}
	return false, nil
}

func isLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{isLeadFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil }}
}
func notLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{isLeadFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil }}
}

// ── ProblemProvider mock ─────────────────────────────────────────────────────

type mockProblemProvider struct {
	findBySlugsFn func(ctx context.Context, slugs []string, callerID string, isAdmin bool) (map[string]*ProblemInfo, error)
	findByIDsFn   func(ctx context.Context, ids []string) (map[string]*ProblemBasicInfo, error)
}

func (m *mockProblemProvider) FindBySlugs(ctx context.Context, slugs []string, callerID string, isAdmin bool) (map[string]*ProblemInfo, error) {
	if m.findBySlugsFn != nil {
		return m.findBySlugsFn(ctx, slugs, callerID, isAdmin)
	}
	result := make(map[string]*ProblemInfo, len(slugs))
	for _, s := range slugs {
		result[s] = &ProblemInfo{ID: testProblemID, Slug: s, Title: s + "-title", IsPublished: true, CanAdd: true}
	}
	return result, nil
}

func (m *mockProblemProvider) FindByIDs(ctx context.Context, ids []string) (map[string]*ProblemBasicInfo, error) {
	if m.findByIDsFn != nil {
		return m.findByIDsFn(ctx, ids)
	}
	result := make(map[string]*ProblemBasicInfo, len(ids))
	for _, id := range ids {
		result[id] = &ProblemBasicInfo{ID: id, Slug: "slug-" + id, Title: "title-" + id}
	}
	return result, nil
}

func noProblemProvider() *mockProblemProvider { return &mockProblemProvider{} }

// ── OwnerProvider mock ───────────────────────────────────────────────────────

type mockOwnerProvider struct {
	getDisplayFn func(ctx context.Context, userID string) (*UserDisplay, error)
}

func (m *mockOwnerProvider) GetDisplay(ctx context.Context, userID string) (*UserDisplay, error) {
	if m.getDisplayFn != nil {
		return m.getDisplayFn(ctx, userID)
	}
	return &UserDisplay{Nickname: "coach", Name: "Coach Name"}, nil
}

func stubOwner() *mockOwnerProvider { return &mockOwnerProvider{} }

// ── Contest fixture ──────────────────────────────────────────────────────────

func newTestContest(ownerID string) *domainContest.Contest {
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Test Contest"),
		nil,
		testStart,
		testEnd,
		domainContest.RestorePenalty(20),
		0,
		false,
		false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(ownerID),
		[]domainContest.ContestProblem{},
		testNow.Add(-time.Hour),
		nil,
	)
}

func repoWith(c *domainContest.Contest) *mockContestRepository {
	return &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return c, nil
		},
	}
}
