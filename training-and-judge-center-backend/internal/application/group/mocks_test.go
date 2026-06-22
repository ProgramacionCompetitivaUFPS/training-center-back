package group

import (
	"context"
	"testing"
	"time"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/testutil"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Time fixture ─────────────────────────────────────────────────────────────

var testNow = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

// ── mockGroupRepository ───────────────────────────────────────────────────────

type mockGroupRepository struct {
	groups         []*domainGroup.Group
	total          int
	lastFilters    domainGroup.ListFilters
	returnErr      error
	existsByNameFn func(name domainGroup.GroupName) (bool, error)
	saveErr        error
}

func (m *mockGroupRepository) Save(ctx context.Context, g *domainGroup.Group) error   { return m.saveErr }
func (m *mockGroupRepository) Update(ctx context.Context, g *domainGroup.Group) error { return m.saveErr }
func (m *mockGroupRepository) FindByID(ctx context.Context, id string) (*domainGroup.Group, error) {
	for _, g := range m.groups {
		if g.ID() == id {
			return g, nil
		}
	}
	return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
}
func (m *mockGroupRepository) ExistsByName(ctx context.Context, n domainGroup.GroupName) (bool, error) {
	if m.existsByNameFn != nil {
		return m.existsByNameFn(n)
	}
	return false, nil
}
func (m *mockGroupRepository) FindDefault(ctx context.Context) (*domainGroup.Group, error) {
	return nil, nil
}
func (m *mockGroupRepository) Delete(ctx context.Context, id string) error { return nil }
func (m *mockGroupRepository) List(ctx context.Context, filters domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	m.lastFilters = filters
	if m.returnErr != nil {
		return nil, 0, m.returnErr
	}
	return m.groups, m.total, nil
}

// ── mockMemberRepository ──────────────────────────────────────────────────────

type mockMemberRepository struct {
	memberCounts          map[string]int
	memberships           map[string]*domainGroup.GroupMember // key: groupID+userID
	leadCounts            map[string]int
	leads                 map[string][]*domainGroup.GroupMember
	saveErr               error
	savedMember           *domainGroup.GroupMember
	findByGroupAndUserErr error
}

func (m *mockMemberRepository) Save(_ context.Context, mem *domainGroup.GroupMember) error {
	m.savedMember = mem
	return m.saveErr
}
func (m *mockMemberRepository) Update(_ context.Context, mem *domainGroup.GroupMember) error {
	m.savedMember = mem
	return m.saveErr
}
func (m *mockMemberRepository) SaveAll(ctx context.Context, members []*domainGroup.GroupMember) error {
	return nil
}
func (m *mockMemberRepository) FindByGroupAndUser(ctx context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	if m.findByGroupAndUserErr != nil {
		return nil, m.findByGroupAndUserErr
	}
	if mem, ok := m.memberships[keyOf(groupID, userID)]; ok {
		return mem, nil
	}
	return nil, nil
}
func (m *mockMemberRepository) FindByGroup(ctx context.Context, groupID string, filters domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	return nil, 0, nil
}
func (m *mockMemberRepository) Delete(ctx context.Context, groupID string, userID shared.UserID) error {
	return nil
}
func (m *mockMemberRepository) CountLeads(ctx context.Context, groupID string) (int, error) {
	return m.leadCounts[groupID], nil
}
func (m *mockMemberRepository) CountMembers(ctx context.Context, groupID string) (int, error) {
	return m.memberCounts[groupID], nil
}
func (m *mockMemberRepository) ListLeads(ctx context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	return m.leads[groupID], nil
}
func (m *mockMemberRepository) BulkStats(ctx context.Context, groupIDs []string, viewerID shared.UserID) (map[string]domainGroup.MemberStats, error) {
	result := make(map[string]domainGroup.MemberStats, len(groupIDs))
	for _, gid := range groupIDs {
		s := domainGroup.MemberStats{Count: m.memberCounts[gid]}
		if mem := m.memberships[keyOf(gid, viewerID)]; mem != nil {
			s.IsMember = true
			s.Role = mem.Role()
			s.JoinedAt = mem.JoinedAt()
		}
		result[gid] = s
	}
	return result, nil
}

// ── mockJoinRequestRepository ─────────────────────────────────────────────────

type mockJoinRequestRepository struct {
	requests      []*domainGroup.JoinRequest
	savedRequests []*domainGroup.JoinRequest
	deletedIDs    []string
	saveErr       error
	findErr       error
}

func (m *mockJoinRequestRepository) Save(_ context.Context, r *domainGroup.JoinRequest) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	for i, existing := range m.savedRequests {
		if existing.ID() == r.ID() {
			m.savedRequests[i] = r
			return nil
		}
	}
	m.savedRequests = append(m.savedRequests, r)
	return nil
}

func (m *mockJoinRequestRepository) FindByID(_ context.Context, id string) (*domainGroup.JoinRequest, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	for _, r := range m.requests {
		if r.ID() == id {
			return r, nil
		}
	}
	return nil, nil
}

func (m *mockJoinRequestRepository) FindByGroupAndUser(_ context.Context, groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	var latest *domainGroup.JoinRequest
	for _, r := range m.requests {
		if r.GroupID() == groupID && r.RequesterUserID().Value() == userID.Value() {
			if r.IsPending() {
				return r, nil
			}
			if latest == nil {
				latest = r
			}
		}
	}
	return latest, nil
}

func (m *mockJoinRequestRepository) FindByGroup(_ context.Context, groupID string, filters domainGroup.JoinRequestFilters) ([]*domainGroup.JoinRequest, int, error) {
	var out []*domainGroup.JoinRequest
	for _, r := range m.requests {
		if r.GroupID() != groupID {
			continue
		}
		if filters.Status != nil && r.Status() != *filters.Status {
			continue
		}
		out = append(out, r)
	}
	return out, len(out), nil
}

func (m *mockJoinRequestRepository) Delete(_ context.Context, id string) error {
	m.deletedIDs = append(m.deletedIDs, id)
	return nil
}

// ── mockTransactionManager ────────────────────────────────────────────────────

type mockTransactionManager struct {
	withTxFn func(ctx context.Context, fn func(txCtx context.Context) error) error
}

func (m *mockTransactionManager) WithTx(ctx context.Context, fn func(txCtx context.Context) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(ctx)
}

// ── mockUserProvider ──────────────────────────────────────────────────────────

type mockUserProvider struct {
	displays map[string]*UserDisplay
	err      error
}

func (m *mockUserProvider) GetDisplays(ctx context.Context, ids []string) (map[string]*UserDisplay, error) {
	if m.err != nil {
		return nil, m.err
	}
	out := make(map[string]*UserDisplay, len(ids))
	for _, id := range ids {
		if d, ok := m.displays[id]; ok {
			out[id] = d
		}
	}
	return out, nil
}

// ── mockInvitationTokenService ────────────────────────────────────────────────

type mockInvitationTokenService struct {
	token       string
	genErr      error
	claims      *InvitationClaims
	validateErr error
}

func (m *mockInvitationTokenService) GenerateInviteToken(groupID, inviterID string) (string, error) {
	return m.token, m.genErr
}

func (m *mockInvitationTokenService) ValidateInviteToken(token string) (*InvitationClaims, error) {
	return m.claims, m.validateErr
}

// ── mockPreferencesReader ─────────────────────────────────────────────────────

type mockPreferencesReader struct {
	hide bool
}

func (m *mockPreferencesReader) HideGlobalGroup(ctx context.Context, userID string) (bool, error) {
	return m.hide, nil
}

// ── CurrentUser helpers ───────────────────────────────────────────────────────

var (
	asAdmin      = testutil.AsAdmin
	asCoach      = testutil.AsCoach
	asContestant = testutil.AsContestant
)

// ── Shared domain fixtures ────────────────────────────────────────────────────

func keyOf(groupID string, userID shared.UserID) string { return groupID + "::" + userID.Value() }

func mustGroup(t *testing.T, id, name string, visibility domainGroup.Visibility, joinPolicy domainGroup.JoinPolicy) *domainGroup.Group {
	t.Helper()
	gn, err := domainGroup.NewGroupName(name)
	if err != nil {
		t.Fatalf("NewGroupName: %v", err)
	}
	g, err := domainGroup.NewGroup(id, gn, nil, visibility, joinPolicy, shared.RestoreUserID("creator-id"),
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("NewGroup: %v", err)
	}
	return g
}

func mustJoinRequest(t *testing.T, id, groupID, userID string) *domainGroup.JoinRequest {
	t.Helper()
	req, err := domainGroup.NewJoinRequest(
		id, groupID, shared.RestoreUserID(userID), nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("NewJoinRequest: %v", err)
	}
	return req
}

func leadMemberRepo(groupID, userID string) *mockMemberRepository {
	uid := shared.RestoreUserID(userID)
	lead, _ := domainGroup.NewGroupMember("m-lead", groupID, uid, domainGroup.MemberRoleLead, domainGroup.JoinMethodDirectAdd, nil, testNow)
	return &mockMemberRepository{memberships: map[string]*domainGroup.GroupMember{keyOf(groupID, uid): lead}}
}

func pendingRequest(id, groupID, requesterID string) *domainGroup.JoinRequest {
	req, _ := domainGroup.NewJoinRequest(id, groupID, shared.RestoreUserID(requesterID), nil,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	return req
}

func inviteGroup(t *testing.T) *domainGroup.Group {
	t.Helper()
	return mustGroup(t, "g1", "Invite Club", domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite)
}

func mustUID(id string) shared.UserID { return shared.RestoreUserID(id) }

// ── mockGroupDeletionProvider ─────────────────────────────────────────────────

type mockGroupDeletionProvider struct {
	counts DeletionCounts
	err    error
}

func (m *mockGroupDeletionProvider) GetDeletionCounts(_ context.Context, _ string) (DeletionCounts, error) {
	if m.err != nil {
		return DeletionCounts{}, m.err
	}
	return m.counts, nil
}

// ── mockGroupStandingsInvalidator ─────────────────────────────────────────────

type mockGroupStandingsInvalidator struct {
	err error
}

func (m *mockGroupStandingsInvalidator) Invalidate(_ context.Context, _ string) error {
	return m.err
}

// ── mockNicknameResolver ──────────────────────────────────────────────────────

type mockNicknameResolver struct {
	user *UserDisplay
	err  error
}

func (m *mockNicknameResolver) ResolveByNickname(_ context.Context, _ string) (*UserDisplay, error) {
	return m.user, m.err
}

// ── mockContestRegistrationCleaner ────────────────────────────────────────────

type mockContestRegistrationCleaner struct {
	deletedCount int
	err          error
	called       bool
}

func (m *mockContestRegistrationCleaner) DeleteScheduledByGroupAndUser(_ context.Context, _, _ string) (int, error) {
	m.called = true
	return m.deletedCount, m.err
}
