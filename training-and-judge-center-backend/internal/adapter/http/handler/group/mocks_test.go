package group

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// â”€â”€ mockGroupRepo â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type mockGroupRepo struct {
	findByIDFn     func(id string) (*domainGroup.Group, error)
	existsByNameFn func(name domainGroup.GroupName) (bool, error)
	saveErr        error
	savedGroup     *domainGroup.Group
}

func (s *mockGroupRepo) Save(_ context.Context, g *domainGroup.Group) error {
	s.savedGroup = g
	return s.saveErr
}
func (s *mockGroupRepo) Update(_ context.Context, _ *domainGroup.Group) error { return nil }
func (s *mockGroupRepo) FindByID(_ context.Context, id string) (*domainGroup.Group, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(id)
	}
	return nil, nil
}
func (s *mockGroupRepo) ExistsByName(_ context.Context, n domainGroup.GroupName) (bool, error) {
	if s.existsByNameFn != nil {
		return s.existsByNameFn(n)
	}
	return false, nil
}
func (s *mockGroupRepo) FindDefault(_ context.Context) (*domainGroup.Group, error) { return nil, nil }
func (s *mockGroupRepo) Delete(_ context.Context, _ string) error                  { return nil }
func (s *mockGroupRepo) List(_ context.Context, _ domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	return nil, 0, nil
}

// â”€â”€ mockMemberRepo â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type mockMemberRepo struct {
	findByGroupAndUserFn func(groupID string, userID shared.UserID) (*domainGroup.GroupMember, error)
	countMembersFn       func(groupID string) (int, error)
	countLeadsFn         func(groupID string) (int, error)
	listLeadsFn          func(groupID string) ([]*domainGroup.GroupMember, error)
	saveAllErr           error
	savedMembers         []*domainGroup.GroupMember
}

func (s *mockMemberRepo) Save(_ context.Context, _ *domainGroup.GroupMember) error   { return nil }
func (s *mockMemberRepo) Update(_ context.Context, _ *domainGroup.GroupMember) error { return nil }
func (s *mockMemberRepo) SaveAll(_ context.Context, members []*domainGroup.GroupMember) error {
	if s.saveAllErr != nil {
		return s.saveAllErr
	}
	s.savedMembers = append(s.savedMembers, members...)
	return nil
}
func (s *mockMemberRepo) FindByGroupAndUser(_ context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	if s.findByGroupAndUserFn != nil {
		return s.findByGroupAndUserFn(groupID, userID)
	}
	return nil, nil
}
func (s *mockMemberRepo) FindByGroup(_ context.Context, _ string, _ domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	return nil, 0, nil
}
func (s *mockMemberRepo) Delete(_ context.Context, _ string, _ shared.UserID) error { return nil }
func (s *mockMemberRepo) CountLeads(_ context.Context, groupID string) (int, error) {
	if s.countLeadsFn != nil {
		return s.countLeadsFn(groupID)
	}
	return 0, nil
}
func (s *mockMemberRepo) CountMembers(_ context.Context, groupID string) (int, error) {
	if s.countMembersFn != nil {
		return s.countMembersFn(groupID)
	}
	return 0, nil
}
func (s *mockMemberRepo) ListLeads(_ context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	if s.listLeadsFn != nil {
		return s.listLeadsFn(groupID)
	}
	return nil, nil
}
func (s *mockMemberRepo) BulkStats(_ context.Context, _ []string, _ shared.UserID) (map[string]domainGroup.MemberStats, error) {
	return nil, nil
}

// â”€â”€ mockJoinRequestRepo â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type mockJoinRequestRepo struct {
	findByIDFn           func(id string) (*domainGroup.JoinRequest, error)
	findByGroupAndUserFn func(groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error)
}

func (s *mockJoinRequestRepo) Save(_ context.Context, _ *domainGroup.JoinRequest) error { return nil }
func (s *mockJoinRequestRepo) FindByID(_ context.Context, id string) (*domainGroup.JoinRequest, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(id)
	}
	return nil, nil
}
func (s *mockJoinRequestRepo) FindByGroupAndUser(_ context.Context, groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error) {
	if s.findByGroupAndUserFn != nil {
		return s.findByGroupAndUserFn(groupID, userID)
	}
	return nil, nil
}
func (s *mockJoinRequestRepo) FindByGroup(_ context.Context, _ string, _ domainGroup.JoinRequestFilters) ([]*domainGroup.JoinRequest, int, error) {
	return nil, 0, nil
}
func (s *mockJoinRequestRepo) Delete(_ context.Context, _ string) error { return nil }

// ── mockInvitationRepo ───────────────────────────────────────────────────────

type mockInvitationRepo struct {
	findByIDFn                     func(id string) (*domainGroup.GroupInvitation, error)
	findPendingByGroupAndInviteeFn func(groupID string, inviteeID *shared.UserID) (*domainGroup.GroupInvitation, error)
	saveFn                         func(inv *domainGroup.GroupInvitation) error
	transitionStatusFn             func(id string, from, to domainGroup.InvitationStatus) error
	findByGroupFn                  func(groupID string, filters domainGroup.InvitationFilters) ([]*domainGroup.GroupInvitation, int, error)
}

func (s *mockInvitationRepo) Save(_ context.Context, inv *domainGroup.GroupInvitation) error {
	if s.saveFn != nil {
		return s.saveFn(inv)
	}
	return nil
}
func (s *mockInvitationRepo) FindByID(_ context.Context, id string) (*domainGroup.GroupInvitation, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(id)
	}
	return nil, apperror.NewNotFound(domainGroup.ErrCodeInvitationNotFound, "invitation not found")
}
func (s *mockInvitationRepo) FindPendingByGroupAndInvitee(_ context.Context, groupID string, inviteeID *shared.UserID) (*domainGroup.GroupInvitation, error) {
	if s.findPendingByGroupAndInviteeFn != nil {
		return s.findPendingByGroupAndInviteeFn(groupID, inviteeID)
	}
	return nil, nil
}
func (s *mockInvitationRepo) FindByGroup(_ context.Context, groupID string, filters domainGroup.InvitationFilters) ([]*domainGroup.GroupInvitation, int, error) {
	if s.findByGroupFn != nil {
		return s.findByGroupFn(groupID, filters)
	}
	return nil, 0, nil
}
func (s *mockInvitationRepo) TransitionStatus(_ context.Context, id string, from, to domainGroup.InvitationStatus) error {
	if s.transitionStatusFn != nil {
		return s.transitionStatusFn(id, from, to)
	}
	return nil
}

// ── mockNicknameResolver / mockEmailResolver ────────────────────────────────

type mockNicknameResolver struct {
	user  *appGroup.UserDisplay
	err   error
	users map[string]*appGroup.UserDisplay
}

func (s *mockNicknameResolver) ResolveByNickname(_ context.Context, nickname string) (*appGroup.UserDisplay, error) {
	if s.users != nil {
		return s.users[nickname], nil
	}
	if s.err != nil {
		return nil, s.err
	}
	return s.user, nil
}

type mockEmailResolver struct {
	user *appGroup.UserDisplay
	err  error
}

func (s *mockEmailResolver) ResolveByEmail(_ context.Context, _ string) (*appGroup.UserDisplay, error) {
	return s.user, s.err
}

// â”€â”€ other stubs â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type mockUserProvider struct{}

func (s *mockUserProvider) GetDisplays(_ context.Context, _ []string) (map[string]*appGroup.UserDisplay, error) {
	return nil, nil
}

type mockPrefsReader struct{}

func (s *mockPrefsReader) HideGlobalGroup(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type mockTxManager struct{}

func (s *mockTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// â”€â”€ auth mocks â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User) (string, error) {
	return "tok", nil
}
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "u1", Role: shared.RoleContestant}, nil
}

type mockAdminTokenSvc struct{}

func (m *mockAdminTokenSvc) GenerateToken(_ context.Context, _ *domainUser.User) (string, error) {
	return "tok", nil
}
func (m *mockAdminTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "admin-1", Role: shared.RoleAdmin}, nil
}

// â”€â”€ request helpers â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func authedRequest(method, target string) *http.Request {
	r := httptest.NewRequest(method, target, nil)
	r.Header.Set("Authorization", "Bearer tok")
	return r
}

func authedPostRequest(target, body string) *http.Request {
	r := httptest.NewRequest("POST", target, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	return r
}

func wrapAuth(h http.Handler) http.Handler {
	return middleware.Auth(&mockTokenSvc{}, nil)(h)
}

func wrapAuthAsAdmin(h http.Handler) http.Handler {
	return middleware.Auth(&mockAdminTokenSvc{}, nil)(h)
}

// â”€â”€ shared fixtures â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€â”€

func testTime() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

// mockHandler wires a Handler with all default (no-op) dependencies.
func mockHandler() *Handler {
	repo := &mockGroupRepo{}
	memberRepo := &mockMemberRepo{}
	joinRequestRepo := &mockJoinRequestRepo{}
	invitationRepo := &mockInvitationRepo{}
	txMgr := &mockTxManager{}
	return NewHandler(
		appGroup.NewCreateGroupUseCase(repo, memberRepo, &mockNicknameResolver{}, txMgr),
		appGroup.NewListGroupsUseCase(repo, memberRepo),
		appGroup.NewGetGroupUseCase(repo, memberRepo, &mockUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, memberRepo, &mockPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, memberRepo),
		appGroup.NewRequestJoinUseCase(repo, memberRepo, joinRequestRepo),
		appGroup.NewApproveRequestUseCase(memberRepo, joinRequestRepo, txMgr),
		appGroup.NewRejectRequestUseCase(memberRepo, joinRequestRepo),
		appGroup.NewListJoinRequestsUseCase(memberRepo, joinRequestRepo, &mockUserProvider{}),
		appGroup.NewGetMyRequestUseCase(joinRequestRepo),
		appGroup.NewCancelMyRequestUseCase(joinRequestRepo),
		appGroup.NewGenerateInviteUseCase(repo, memberRepo, invitationRepo, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, txMgr),
		appGroup.NewAcceptInviteUseCase(repo, memberRepo, invitationRepo, txMgr),
		appGroup.NewListGroupInvitationsUseCase(memberRepo, invitationRepo, &mockUserProvider{}),
		appGroup.NewRevokeInvitationUseCase(memberRepo, invitationRepo),
		nil, /* addMember */
		nil, /* removeMember */
		nil, /* changeRole */
		nil, /* leaveGroup */
		nil, /* listMembers */
		nil, /* deleteGroup */
		nil, /* updateGroup */
	)
}
