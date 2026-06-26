package contest

import (
	"context"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil"
)

// ── Time fixture ─────────────────────────────────────────────────────────────

// testNow is a fixed anchor for createdAt/updatedAt in fixtures (not used for validation).
var testNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// testStart/testEnd are relative to the real clock so time-validation tests
// pass regardless of when tests run.
var (
	testStart = time.Now().Add(24 * time.Hour)
	testEnd   = time.Now().Add(29 * time.Hour)
)

// ── Repository mock ──────────────────────────────────────────────────────────

type mockContestRepository struct {
	createFn   func(ctx context.Context, c *domainContest.Contest) error
	updateFn   func(ctx context.Context, c *domainContest.Contest) error
	findByIDFn func(ctx context.Context, id string) (*domainContest.Contest, error)
	deleteFn   func(ctx context.Context, id string) error
	listFn     func(ctx context.Context, f domainContest.ListFilters) ([]*domainContest.Contest, int, error)
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
	return nil, nil
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
	return &GroupInfo{ID: groupID, Name: "Test Group", IsVisible: true}, nil
}

func groupFound() *mockGroupProvider { return &mockGroupProvider{} }
func groupFoundNotVisible() *mockGroupProvider {
	return &mockGroupProvider{findByIDFn: func(_ context.Context, id string) (*GroupInfo, error) {
		return &GroupInfo{ID: id, Name: "Hidden Group", IsVisible: false}, nil
	}}
}
func groupNotFound() *mockGroupProvider {
	return &mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) {
		return nil, nil
	}}
}

// ── GroupMemberProvider mock ─────────────────────────────────────────────────

type mockGroupMemberProvider struct {
	isLeadFn   func(ctx context.Context, userID, groupID string) (bool, error)
	isMemberFn func(ctx context.Context, userID, groupID string) (bool, error)
}

func (m *mockGroupMemberProvider) IsLeadOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	if m.isLeadFn != nil {
		return m.isLeadFn(ctx, userID, groupID)
	}
	return false, nil
}

func (m *mockGroupMemberProvider) IsMemberOfGroup(ctx context.Context, userID, groupID string) (bool, error) {
	if m.isMemberFn != nil {
		return m.isMemberFn(ctx, userID, groupID)
	}
	return false, nil
}

func isLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{
		isLeadFn:   func(_ context.Context, _, _ string) (bool, error) { return true, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
}
func notLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{
		isLeadFn:   func(_ context.Context, _, _ string) (bool, error) { return false, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
}
func isMemberNotLead() *mockGroupMemberProvider {
	return &mockGroupMemberProvider{
		isLeadFn:   func(_ context.Context, _, _ string) (bool, error) { return false, nil },
		isMemberFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
}

// ── ProblemProvider mock ─────────────────────────────────────────────────────

type mockProblemProvider struct {
	findBySlugsFn         func(ctx context.Context, slugs []string, callerID string, isAdmin bool) (map[string]*ProblemInfo, error)
	findByIDsFn           func(ctx context.Context, ids []string) (map[string]*ProblemBasicInfo, error)
	findByIDsWithLimitsFn func(ctx context.Context, ids []string) (map[string]*ProblemWithLimits, error)
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

func (m *mockProblemProvider) FindByIDsWithLimits(ctx context.Context, ids []string) (map[string]*ProblemWithLimits, error) {
	if m.findByIDsWithLimitsFn != nil {
		return m.findByIDsWithLimitsFn(ctx, ids)
	}
	result := make(map[string]*ProblemWithLimits, len(ids))
	for _, id := range ids {
		result[id] = &ProblemWithLimits{ID: id, Slug: "slug-" + id, Title: "title-" + id, TimeLimit: 1000, MemoryLimit: 256}
	}
	return result, nil
}

func defaultProblemProvider() *mockProblemProvider { return &mockProblemProvider{} }

// ── ContestParticipantProvider mock ──────────────────────────────────────────

type mockContestParticipantProvider struct {
	isRegisteredFn          func(ctx context.Context, contestID, userID string) (bool, error)
	countParticipantsFn     func(ctx context.Context, contestID string) (int, error)
	countParticipantsBulkFn func(ctx context.Context, contestIDs []string) (map[string]int, error)
	isRegisteredBulkFn      func(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error)
}

func (m *mockContestParticipantProvider) IsRegistered(ctx context.Context, contestID, userID string) (bool, error) {
	if m.isRegisteredFn != nil {
		return m.isRegisteredFn(ctx, contestID, userID)
	}
	return false, nil
}

func (m *mockContestParticipantProvider) CountParticipants(ctx context.Context, contestID string) (int, error) {
	if m.countParticipantsFn != nil {
		return m.countParticipantsFn(ctx, contestID)
	}
	return 0, nil
}

func (m *mockContestParticipantProvider) CountParticipantsBulk(ctx context.Context, contestIDs []string) (map[string]int, error) {
	if m.countParticipantsBulkFn != nil {
		return m.countParticipantsBulkFn(ctx, contestIDs)
	}
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	return result, nil
}

func (m *mockContestParticipantProvider) IsRegisteredBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error) {
	if m.isRegisteredBulkFn != nil {
		return m.isRegisteredBulkFn(ctx, contestIDs, userID)
	}
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = false
	}
	return result, nil
}

func mockParticipants() *mockContestParticipantProvider { return &mockContestParticipantProvider{} }

// ── RegistrationRepository mock ──────────────────────────────────────────────

type mockRegistrationRepository struct {
	saveFn                  func(ctx context.Context, r *domainContest.ContestRegistration) error
	findByContestAndUserFn  func(ctx context.Context, contestID, userID string) (*domainContest.ContestRegistration, error)
	deleteFn                func(ctx context.Context, contestID, userID string) error
	listByContestFn         func(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestRegistration, int, error)
	existsByContestAndUser  func(ctx context.Context, contestID, userID string) (bool, error)
	countByContestFn        func(ctx context.Context, contestID string) (int, error)
	countByContestBulkFn    func(ctx context.Context, contestIDs []string) (map[string]int, error)
	existsByUserBulkFn      func(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error)
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
	if m.existsByContestAndUser != nil {
		return m.existsByContestAndUser(ctx, contestID, userID)
	}
	return false, nil
}

func (m *mockRegistrationRepository) CountByContest(ctx context.Context, contestID string) (int, error) {
	if m.countByContestFn != nil {
		return m.countByContestFn(ctx, contestID)
	}
	return 0, nil
}

func (m *mockRegistrationRepository) CountByContestBulk(ctx context.Context, contestIDs []string) (map[string]int, error) {
	if m.countByContestBulkFn != nil {
		return m.countByContestBulkFn(ctx, contestIDs)
	}
	result := make(map[string]int, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = 0
	}
	return result, nil
}

func (m *mockRegistrationRepository) ExistsByUserBulk(ctx context.Context, contestIDs []string, userID string) (map[string]bool, error) {
	if m.existsByUserBulkFn != nil {
		return m.existsByUserBulkFn(ctx, contestIDs, userID)
	}
	result := make(map[string]bool, len(contestIDs))
	for _, id := range contestIDs {
		result[id] = false
	}
	return result, nil
}

func mockRegistrations() *mockRegistrationRepository { return &mockRegistrationRepository{} }

// ── ParticipantNicknameProvider mock ─────────────────────────────────────────

type mockNicknameProvider struct {
	getFn func(ctx context.Context, userIDs []string) (map[string]string, error)
}

func (m *mockNicknameProvider) GetNicknamesByIDs(ctx context.Context, userIDs []string) (map[string]string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, userIDs)
	}
	result := make(map[string]string, len(userIDs))
	for _, id := range userIDs {
		result[id] = "nick_" + id
	}
	return result, nil
}

func mockNicknames() *mockNicknameProvider { return &mockNicknameProvider{} }

// ── OwnerProvider mock ───────────────────────────────────────────────────────

type mockOwnerProvider struct {
	getDisplayFn func(ctx context.Context, userID string) (*UserDisplay, error)
}

func (m *mockOwnerProvider) GetDisplay(ctx context.Context, userID string) (*UserDisplay, error) {
	if m.getDisplayFn != nil {
		return m.getDisplayFn(ctx, userID)
	}
	return &UserDisplay{ID: userID, Nickname: "coach", Name: "Coach Name"}, nil
}

func mockOwner() *mockOwnerProvider { return &mockOwnerProvider{} }

// ── StandingsCache mock ──────────────────────────────────────────────────────

type mockStandingsCache struct {
	getFn                func(ctx context.Context, contestID string) (*CachedStandings, error)
	setFn                func(ctx context.Context, contestID string, data *CachedStandings) error
	acquireRefreshLockFn func(ctx context.Context, contestID string, ttl time.Duration) (bool, error)
	releaseRefreshLockFn func(ctx context.Context, contestID string) error
	invalidateFn         func(ctx context.Context, contestID string) error
}

func (m *mockStandingsCache) Get(ctx context.Context, contestID string) (*CachedStandings, error) {
	if m.getFn != nil {
		return m.getFn(ctx, contestID)
	}
	return nil, nil
}
func (m *mockStandingsCache) Set(ctx context.Context, contestID string, data *CachedStandings) error {
	if m.setFn != nil {
		return m.setFn(ctx, contestID, data)
	}
	return nil
}
func (m *mockStandingsCache) AcquireRefreshLock(ctx context.Context, contestID string, ttl time.Duration) (bool, error) {
	if m.acquireRefreshLockFn != nil {
		return m.acquireRefreshLockFn(ctx, contestID, ttl)
	}
	return true, nil
}
func (m *mockStandingsCache) ReleaseRefreshLock(ctx context.Context, contestID string) error {
	if m.releaseRefreshLockFn != nil {
		return m.releaseRefreshLockFn(ctx, contestID)
	}
	return nil
}
func (m *mockStandingsCache) Invalidate(ctx context.Context, contestID string) error {
	if m.invalidateFn != nil {
		return m.invalidateFn(ctx, contestID)
	}
	return nil
}

// ── ContestSubmissionsProvider mock ──────────────────────────────────────────

type mockContestSubmissionsProvider struct {
	listByContestFn func(ctx context.Context, contestID string, filters ContestSubmissionFilters) ([]RichSubmissionData, error)
}

func (m *mockContestSubmissionsProvider) ListByContest(ctx context.Context, contestID string, filters ContestSubmissionFilters) ([]RichSubmissionData, error) {
	if m.listByContestFn != nil {
		return m.listByContestFn(ctx, contestID, filters)
	}
	return nil, nil
}

// ── CallerStandingProvider mock ───────────────────────────────────────────────

type mockCallerStandingProvider struct {
	fn func(contestID, userID string) (string, bool, error)
}

func (m *mockCallerStandingProvider) GetCallerStandingID(_ context.Context, contestID, userID string) (string, bool, error) {
	if m.fn != nil {
		return m.fn(contestID, userID)
	}
	return userID, true, nil // default: individual, standingID = userID
}

// ── StandingsSubmissionProvider mock ─────────────────────────────────────────

type mockStandingsSubmissionProvider struct {
	listByContestFn func(ctx context.Context, contestID string) ([]ContestSubmissionData, error)
}

func (m *mockStandingsSubmissionProvider) ListByContest(ctx context.Context, contestID string) ([]ContestSubmissionData, error) {
	if m.listByContestFn != nil {
		return m.listByContestFn(ctx, contestID)
	}
	return nil, nil
}

// ── TeamParticipantProvider mock ─────────────────────────────────────────────

type mockTeamParticipantProvider struct {
	listSelectedMembersFn func(ctx context.Context, contestID string) (map[string][]string, error)
}

func (m *mockTeamParticipantProvider) ListSelectedMembersByContest(ctx context.Context, contestID string) (map[string][]string, error) {
	if m.listSelectedMembersFn != nil {
		return m.listSelectedMembersFn(ctx, contestID)
	}
	return map[string][]string{}, nil
}

// ── TransactionManager mock ──────────────────────────────────────────────────

type mockTransactionManager struct {
	withTxFn func(ctx context.Context, fn func(txCtx context.Context) error) error
}

func (m *mockTransactionManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(ctx)
}

// ── TeamParticipationRepository mock ────────────────────────────────────────

type mockTeamParticipationRepository struct {
	listFn func(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestTeamParticipation, int, error)
}

func (m *mockTeamParticipationRepository) Save(_ context.Context, _ *domainContest.ContestTeamParticipation) error {
	return nil
}
func (m *mockTeamParticipationRepository) FindByContestAndTeam(_ context.Context, _, _ string) (*domainContest.ContestTeamParticipation, error) {
	return nil, nil
}
func (m *mockTeamParticipationRepository) Update(_ context.Context, _ *domainContest.ContestTeamParticipation) error {
	return nil
}
func (m *mockTeamParticipationRepository) Delete(_ context.Context, _, _ string) error { return nil }
func (m *mockTeamParticipationRepository) List(ctx context.Context, contestID string, page, limit int) ([]*domainContest.ContestTeamParticipation, int, error) {
	if m.listFn != nil {
		return m.listFn(ctx, contestID, page, limit)
	}
	return []*domainContest.ContestTeamParticipation{}, 0, nil
}
func (m *mockTeamParticipationRepository) AreUsersSelectedInAnyTeam(_ context.Context, _ string, userIDs []string, _ string) (map[string]bool, error) {
	result := make(map[string]bool, len(userIDs))
	for _, id := range userIDs {
		result[id] = false
	}
	return result, nil
}

func mockTeamParticipRepo() *mockTeamParticipationRepository {
	return &mockTeamParticipationRepository{}
}

// ── CurrentUser helpers ──────────────────────────────────────────────────────

var (
	asAdmin      = testutil.AsAdmin
	asCoach      = testutil.AsCoach
	asContestant = testutil.AsContestant
)

// ── Test constants ───────────────────────────────────────────────────────────

const (
	callerID      = "aaaaaaaa-0000-0000-0000-000000000001"
	otherID       = "aaaaaaaa-0000-0000-0000-000000000002"
	testGroupID   = "bbbbbbbb-0000-0000-0000-000000000001"
	testContestID = "cccccccc-0000-0000-0000-000000000001"
	testProblemID = "dddddddd-0000-0000-0000-000000000001"
)

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
		false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(ownerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow.Add(-time.Hour),
		nil,
	)
}

func newFinishedContest(ownerID string) *domainContest.Contest {
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Finished Contest"),
		nil,
		time.Now().Add(-48*time.Hour),
		time.Now().Add(-24*time.Hour),
		domainContest.RestorePenalty(20),
		0,
		false,
		false,
		false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(ownerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
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
