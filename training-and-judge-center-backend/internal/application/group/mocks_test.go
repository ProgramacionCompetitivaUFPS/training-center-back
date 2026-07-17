package group

import (
	"context"
	"testing"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
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
	savedGroup     *domainGroup.Group
}

func (m *mockGroupRepository) Save(ctx context.Context, g *domainGroup.Group) error {
	m.savedGroup = g
	return m.saveErr
}
func (m *mockGroupRepository) Update(ctx context.Context, g *domainGroup.Group) error {
	return m.saveErr
}
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
	savedMembers          []*domainGroup.GroupMember
	saveAllErr            error
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
	if m.saveAllErr != nil {
		return m.saveAllErr
	}
	m.savedMembers = append(m.savedMembers, members...)
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
	displays    map[string]*UserDisplay
	err         error
	lastCallIDs []string
}

func (m *mockUserProvider) GetDisplays(ctx context.Context, ids []string) (map[string]*UserDisplay, error) {
	m.lastCallIDs = ids
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

// ── mockInvitationRepository ──────────────────────────────────────────────────

type findPendingCall struct {
	groupID   string
	inviteeID *shared.UserID
}

type transitionCall struct {
	id   string
	from domainGroup.InvitationStatus
	to   domainGroup.InvitationStatus
}

type mockInvitationRepository struct {
	byID              map[string]*domainGroup.GroupInvitation
	findByIDErr       error
	savedInvitations  []*domainGroup.GroupInvitation
	saveErr           error
	pendingResult     *domainGroup.GroupInvitation
	findPendingErr    error
	findPendingCalls  []findPendingCall
	transitions       []transitionCall
	transitionErr     error
	findByGroupResult []*domainGroup.GroupInvitation
	findByGroupTotal  int
	findByGroupErr    error
	lastFilters       domainGroup.InvitationFilters
}

func (m *mockInvitationRepository) Save(_ context.Context, inv *domainGroup.GroupInvitation) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedInvitations = append(m.savedInvitations, inv)
	if m.byID == nil {
		m.byID = map[string]*domainGroup.GroupInvitation{}
	}
	m.byID[inv.ID()] = inv
	return nil
}

func (m *mockInvitationRepository) FindByID(_ context.Context, id string) (*domainGroup.GroupInvitation, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	if inv, ok := m.byID[id]; ok {
		// Return a fresh copy, not the stored pointer: a real repository read
		// is an independent DB round-trip and must not reflect in-memory-only
		// mutations (e.g. inv.Accept()) the caller made to a previous read
		// before persisting them.
		return domainGroup.RestoreGroupInvitation(inv.ID(), inv.GroupID(), inv.InviteeID(), inv.InvitedBy(), inv.Status(), inv.ExpiresAt(), inv.CreatedAt()), nil
	}
	return nil, apperror.NewNotFound(domainGroup.ErrCodeInvitationNotFound, "invitation not found")
}

func (m *mockInvitationRepository) FindPendingByGroupAndInvitee(_ context.Context, groupID string, inviteeID *shared.UserID) (*domainGroup.GroupInvitation, error) {
	m.findPendingCalls = append(m.findPendingCalls, findPendingCall{groupID: groupID, inviteeID: inviteeID})
	if m.findPendingErr != nil {
		return nil, m.findPendingErr
	}
	return m.pendingResult, nil
}

func (m *mockInvitationRepository) FindByGroup(_ context.Context, _ string, filters domainGroup.InvitationFilters) ([]*domainGroup.GroupInvitation, int, error) {
	m.lastFilters = filters
	if m.findByGroupErr != nil {
		return nil, 0, m.findByGroupErr
	}
	return m.findByGroupResult, m.findByGroupTotal, nil
}

func (m *mockInvitationRepository) TransitionStatus(_ context.Context, id string, from, to domainGroup.InvitationStatus) error {
	m.transitions = append(m.transitions, transitionCall{id: id, from: from, to: to})
	return m.transitionErr
}

// ── mockEmailResolver ─────────────────────────────────────────────────────────

type mockEmailResolver struct {
	user *UserDisplay
	err  error
}

func (m *mockEmailResolver) ResolveByEmail(_ context.Context, _ string) (*UserDisplay, error) {
	return m.user, m.err
}

// ── mockEmailSender ────────────────────────────────────────────────────────────

type mockEmailSender struct {
	sendFn   func(ctx context.Context, msg appshared.EmailMessage) error
	sentMsgs []appshared.EmailMessage
}

func (m *mockEmailSender) Send(ctx context.Context, msg appshared.EmailMessage) error {
	m.sentMsgs = append(m.sentMsgs, msg)
	if m.sendFn != nil {
		return m.sendFn(ctx, msg)
	}
	return nil
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

// mustInvitation builds a fresh, non-expired invitation anchored to the real
// wall clock — AcceptInviteUseCase computes expiry against time.Now() (D10:
// the use case owns the time source, not an injected fixture), so a fixture
// built from the fixed testNow (2026-01-01) would already read as expired.
func mustInvitation(t *testing.T, id, groupID string, inviteeID *shared.UserID, invitedBy string) *domainGroup.GroupInvitation {
	t.Helper()
	inv, err := domainGroup.NewGroupInvitation(id, groupID, inviteeID, shared.RestoreUserID(invitedBy), time.Now())
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	return inv
}

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
	user  *UserDisplay
	err   error
	users map[string]*UserDisplay
}

func (m *mockNicknameResolver) ResolveByNickname(_ context.Context, nickname string) (*UserDisplay, error) {
	if m.users != nil {
		return m.users[nickname], nil
	}
	if m.err != nil {
		return nil, m.err
	}
	return m.user, nil
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

// ── mockTeamSelectionCleaner ──────────────────────────────────────────────────

type mockTeamSelectionCleaner struct {
	err    error
	called bool
}

func (m *mockTeamSelectionCleaner) RemoveFromScheduledByGroupAndUser(_ context.Context, _, _ string) error {
	m.called = true
	return m.err
}
