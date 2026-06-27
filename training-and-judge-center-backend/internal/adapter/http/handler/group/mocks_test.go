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
)

// ── stubGroupRepo ─────────────────────────────────────────────────────────────

type stubGroupRepo struct {
	findByIDFn     func(id string) (*domainGroup.Group, error)
	existsByNameFn func(name domainGroup.GroupName) (bool, error)
}

func (s *stubGroupRepo) Save(_ context.Context, _ *domainGroup.Group) error   { return nil }
func (s *stubGroupRepo) Update(_ context.Context, _ *domainGroup.Group) error { return nil }
func (s *stubGroupRepo) FindByID(_ context.Context, id string) (*domainGroup.Group, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(id)
	}
	return nil, nil
}
func (s *stubGroupRepo) ExistsByName(_ context.Context, n domainGroup.GroupName) (bool, error) {
	if s.existsByNameFn != nil {
		return s.existsByNameFn(n)
	}
	return false, nil
}
func (s *stubGroupRepo) FindDefault(_ context.Context) (*domainGroup.Group, error) { return nil, nil }
func (s *stubGroupRepo) Delete(_ context.Context, _ string) error                  { return nil }
func (s *stubGroupRepo) List(_ context.Context, _ domainGroup.ListFilters) ([]*domainGroup.Group, int, error) {
	return nil, 0, nil
}

// ── stubMemberRepo ────────────────────────────────────────────────────────────

type stubMemberRepo struct {
	findByGroupAndUserFn func(groupID string, userID shared.UserID) (*domainGroup.GroupMember, error)
	countMembersFn       func(groupID string) (int, error)
	countLeadsFn         func(groupID string) (int, error)
	listLeadsFn          func(groupID string) ([]*domainGroup.GroupMember, error)
}

func (s *stubMemberRepo) Save(_ context.Context, _ *domainGroup.GroupMember) error   { return nil }
func (s *stubMemberRepo) Update(_ context.Context, _ *domainGroup.GroupMember) error { return nil }
func (s *stubMemberRepo) SaveAll(_ context.Context, _ []*domainGroup.GroupMember) error {
	return nil
}
func (s *stubMemberRepo) FindByGroupAndUser(_ context.Context, groupID string, userID shared.UserID) (*domainGroup.GroupMember, error) {
	if s.findByGroupAndUserFn != nil {
		return s.findByGroupAndUserFn(groupID, userID)
	}
	return nil, nil
}
func (s *stubMemberRepo) FindByGroup(_ context.Context, _ string, _ domainGroup.MemberFilters) ([]*domainGroup.GroupMember, int, error) {
	return nil, 0, nil
}
func (s *stubMemberRepo) Delete(_ context.Context, _ string, _ shared.UserID) error { return nil }
func (s *stubMemberRepo) CountLeads(_ context.Context, groupID string) (int, error) {
	if s.countLeadsFn != nil {
		return s.countLeadsFn(groupID)
	}
	return 0, nil
}
func (s *stubMemberRepo) CountMembers(_ context.Context, groupID string) (int, error) {
	if s.countMembersFn != nil {
		return s.countMembersFn(groupID)
	}
	return 0, nil
}
func (s *stubMemberRepo) ListLeads(_ context.Context, groupID string) ([]*domainGroup.GroupMember, error) {
	if s.listLeadsFn != nil {
		return s.listLeadsFn(groupID)
	}
	return nil, nil
}
func (s *stubMemberRepo) BulkStats(_ context.Context, _ []string, _ shared.UserID) (map[string]domainGroup.MemberStats, error) {
	return nil, nil
}

// ── stubJoinRequestRepo ───────────────────────────────────────────────────────

type stubJoinRequestRepo struct {
	findByIDFn           func(id string) (*domainGroup.JoinRequest, error)
	findByGroupAndUserFn func(groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error)
}

func (s *stubJoinRequestRepo) Save(_ context.Context, _ *domainGroup.JoinRequest) error { return nil }
func (s *stubJoinRequestRepo) FindByID(_ context.Context, id string) (*domainGroup.JoinRequest, error) {
	if s.findByIDFn != nil {
		return s.findByIDFn(id)
	}
	return nil, nil
}
func (s *stubJoinRequestRepo) FindByGroupAndUser(_ context.Context, groupID string, userID shared.UserID) (*domainGroup.JoinRequest, error) {
	if s.findByGroupAndUserFn != nil {
		return s.findByGroupAndUserFn(groupID, userID)
	}
	return nil, nil
}
func (s *stubJoinRequestRepo) FindByGroup(_ context.Context, _ string, _ domainGroup.JoinRequestFilters) ([]*domainGroup.JoinRequest, int, error) {
	return nil, 0, nil
}
func (s *stubJoinRequestRepo) Delete(_ context.Context, _ string) error { return nil }

// ── other stubs ───────────────────────────────────────────────────────────────

type stubUserProvider struct{}

func (s *stubUserProvider) GetDisplays(_ context.Context, _ []string) (map[string]*appGroup.UserDisplay, error) {
	return nil, nil
}

type stubPrefsReader struct{}

func (s *stubPrefsReader) HideGlobalGroup(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type stubTxManager struct{}

func (s *stubTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type stubInvitationSvc struct{}

func (s *stubInvitationSvc) GenerateInviteToken(_, _ string) (string, error) {
	return "stub.invite.token", nil
}
func (s *stubInvitationSvc) ValidateInviteToken(_ string) (*appGroup.InvitationClaims, error) {
	return &appGroup.InvitationClaims{GroupID: "g1"}, nil
}

// ── auth mocks ────────────────────────────────────────────────────────────────

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

// ── request helpers ───────────────────────────────────────────────────────────

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

// ── shared fixtures ───────────────────────────────────────────────────────────

func testTime() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

// stubHandler wires a Handler with all default (no-op) dependencies.
func stubHandler() *Handler {
	repo := &stubGroupRepo{}
	memberRepo := &stubMemberRepo{}
	joinRequestRepo := &stubJoinRequestRepo{}
	txMgr := &stubTxManager{}
	inviteSvc := &stubInvitationSvc{}
	return NewHandler(
		appGroup.NewCreateGroupUseCase(repo, memberRepo, txMgr),
		appGroup.NewListGroupsUseCase(repo, memberRepo),
		appGroup.NewGetGroupUseCase(repo, memberRepo, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, memberRepo, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, memberRepo),
		appGroup.NewRequestJoinUseCase(repo, memberRepo, joinRequestRepo),
		appGroup.NewApproveRequestUseCase(memberRepo, joinRequestRepo, txMgr),
		appGroup.NewRejectRequestUseCase(memberRepo, joinRequestRepo),
		appGroup.NewListJoinRequestsUseCase(memberRepo, joinRequestRepo, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(joinRequestRepo),
		appGroup.NewCancelMyRequestUseCase(joinRequestRepo),
		appGroup.NewGenerateInviteUseCase(repo, memberRepo, inviteSvc),
		appGroup.NewAcceptInviteUseCase(repo, memberRepo, inviteSvc),
		nil, /* addMember */
		nil, /* removeMember */
		nil, /* changeRole */
		nil, /* leaveGroup */
		nil, /* listMembers */
		nil, /* deleteGroup */
		nil, /* updateGroup */
	)
}
