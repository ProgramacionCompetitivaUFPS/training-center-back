package group

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type stubGroupRepo struct {
	findByIDFn     func(id string) (*domainGroup.Group, error)
	existsByNameFn func(name domainGroup.GroupName) (bool, error)
}

func (s *stubGroupRepo) Save(_ context.Context, _ *domainGroup.Group) error { return nil }
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

type stubMemberRepo struct {
	findByGroupAndUserFn func(groupID string, userID shared.UserID) (*domainGroup.GroupMember, error)
	countMembersFn       func(groupID string) (int, error)
	countLeadsFn         func(groupID string) (int, error)
	listLeadsFn          func(groupID string) ([]*domainGroup.GroupMember, error)
}

func (s *stubMemberRepo) Save(_ context.Context, _ *domainGroup.GroupMember) error { return nil }
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

type stubUserProvider struct{}

func (s *stubUserProvider) GetDisplays(_ context.Context, _ []string) (map[string]*appGroup.UserDisplay, error) {
	return nil, nil
}

// stubPrefsReader is a no-op preferences reader for tests that don't reach the use case.
type stubPrefsReader struct{}

func (s *stubPrefsReader) HideGlobalGroup(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type stubJoinRequestRepo struct {
	findByIDFn          func(id string) (*domainGroup.JoinRequest, error)
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

type stubTxManager struct{}

func (s *stubTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func stubHandler() *Handler {
	repo := &stubGroupRepo{}
	memberRepo := &stubMemberRepo{}
	joinRequestRepo := &stubJoinRequestRepo{}
	txMgr := &stubTxManager{}
	return NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
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
	)
}

// mockTokenSvc always validates successfully with a contestant role.
type mockTokenSvc struct{}

func (m *mockTokenSvc) GenerateToken(_ *domainUser.User) (string, error) { return "tok", nil }
func (m *mockTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "u1", Role: domainUser.RoleContestant}, nil
}

// mockAdminTokenSvc always validates successfully with an admin role.
type mockAdminTokenSvc struct{}

func (m *mockAdminTokenSvc) GenerateToken(_ *domainUser.User) (string, error) { return "tok", nil }
func (m *mockAdminTokenSvc) ValidateToken(_ string) (*domainUser.TokenClaims, error) {
	return &domainUser.TokenClaims{UserID: "admin-1", Role: domainUser.RoleAdmin}, nil
}

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

func TestListGroups_NonIntegerPageReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroups)).ServeHTTP(w, authedRequest("GET", "/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

func TestListGroups_NonIntegerLimitReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroups)).ServeHTTP(w, authedRequest("GET", "/groups?limit=xyz"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}

func TestListMyGroups_NonIntegerPageReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyGroups)).ServeHTTP(w, authedRequest("GET", "/users/me/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetGroup_NotFoundReturns404(t *testing.T) {
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		},
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("GET", "/groups/nonexistent")
	r.SetPathValue("groupId", "nonexistent")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetGroup)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetGroup_NonMemberHasNilRoleAndJoinedAt(t *testing.T) {
	group := domainGroup.RestoreGroup(
		"g-1", domainGroup.RestoreGroupName("Test Group"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return group, nil },
	}
	// FindByGroupAndUser returns nil, nil — viewer is not a member
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("GET", "/groups/g-1")
	r.SetPathValue("groupId", "g-1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetGroup)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body getGroupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.UserMembership.IsMember {
		t.Error("expected IsMember=false for non-member")
	}
	if body.UserMembership.Role != nil {
		t.Errorf("expected Role=nil for non-member, got %v", body.UserMembership.Role)
	}
	if body.UserMembership.JoinedAt != nil {
		t.Errorf("expected JoinedAt=nil for non-member, got %v", body.UserMembership.JoinedAt)
	}
}

func TestGetGroup_ResponseShape(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-2", domainGroup.RestoreGroupName("My Group"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("GET", "/groups/g-2")
	r.SetPathValue("groupId", "g-2")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetGroup)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body getGroupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.ID != "g-2" {
		t.Errorf("expected ID=g-2, got %s", body.ID)
	}
	if body.Name != "My Group" {
		t.Errorf("expected Name='My Group', got %s", body.Name)
	}
	if body.Visibility != "VISIBLE" {
		t.Errorf("expected Visibility=VISIBLE, got %s", body.Visibility)
	}
	if body.CreatedAt == "" || body.UpdatedAt == "" {
		t.Error("expected non-empty CreatedAt and UpdatedAt")
	}
}

func testTime() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

// --- buildPagination tests ---

func TestBuildPagination_FirstPageOfMany_HasNextNoPrev(t *testing.T) {
	p := buildPagination(30, 1, 3, 10)
	if !p.HasNextPage {
		t.Error("expected HasNextPage=true on page 1 of 3")
	}
	if p.HasPrevPage {
		t.Error("expected HasPrevPage=false on page 1")
	}
}

func TestBuildPagination_LastPage_HasPrevNoNext(t *testing.T) {
	p := buildPagination(30, 3, 3, 10)
	if p.HasNextPage {
		t.Error("expected HasNextPage=false on last page")
	}
	if !p.HasPrevPage {
		t.Error("expected HasPrevPage=true on page 3 of 3")
	}
}

func TestBuildPagination_MiddlePage_HasBoth(t *testing.T) {
	p := buildPagination(30, 2, 3, 10)
	if !p.HasNextPage {
		t.Error("expected HasNextPage=true on middle page")
	}
	if !p.HasPrevPage {
		t.Error("expected HasPrevPage=true on middle page")
	}
}

func TestBuildPagination_SinglePage_HasNeither(t *testing.T) {
	p := buildPagination(5, 1, 1, 10)
	if p.HasNextPage {
		t.Error("expected HasNextPage=false on single page")
	}
	if p.HasPrevPage {
		t.Error("expected HasPrevPage=false on single page")
	}
}

func TestBuildPagination_ZeroResults_HasNeither(t *testing.T) {
	p := buildPagination(0, 1, 0, 10)
	if p.HasNextPage {
		t.Error("expected HasNextPage=false with 0 results")
	}
	if p.HasPrevPage {
		t.Error("expected HasPrevPage=false with 0 results")
	}
}

// --- Create handler tests ---

func TestCreate_InvalidJSONReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{invalid json}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_ContestantReturns403(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Test","joinMode":"OPEN","visibility":"VISIBLE"}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCreate_ValidAdminRequestReturns201(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Algorithms Club","joinMode":"OPEN","visibility":"VISIBLE"}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body groupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Name != "Algorithms Club" {
		t.Errorf("Name = %q, want %q", body.Name, "Algorithms Club")
	}
	if body.ID == "" {
		t.Error("expected non-empty ID in response")
	}
}

func TestCreate_DuplicateNameReturns409(t *testing.T) {
	repo := &stubGroupRepo{
		existsByNameFn: func(_ domainGroup.GroupName) (bool, error) { return true, nil },
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Existing Group","joinMode":"OPEN","visibility":"VISIBLE"}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestJoin_GroupNotFoundReturns404(t *testing.T) {
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		},
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("POST", "/groups/nonexistent/join")
	r.SetPathValue("groupId", "nonexistent")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Join)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestJoin_NonOpenPolicyReturns403(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-invite", domainGroup.RestoreGroupName("Invite Club"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("POST", "/groups/g-invite/join")
	r.SetPathValue("groupId", "g-invite")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Join)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestJoin_AlreadyMemberReturns409(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-open", domainGroup.RestoreGroupName("Open Club"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	userID := shared.RestoreUserID("u1")
	existingMember := domainGroup.RestoreGroupMember("m1", "g-open", userID, domainGroup.MemberRoleMember, testTime())
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	memberRepo := &stubMemberRepo{
		findByGroupAndUserFn: func(_ string, _ shared.UserID) (*domainGroup.GroupMember, error) {
			return existingMember, nil
		},
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, memberRepo),
		appGroup.NewGetGroupUseCase(repo, memberRepo, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, memberRepo, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, memberRepo),
		appGroup.NewRequestJoinUseCase(repo, memberRepo, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(memberRepo, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(memberRepo, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(memberRepo, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("POST", "/groups/g-open/join")
	r.SetPathValue("groupId", "g-open")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Join)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestJoin_SuccessReturns201WithRoleAndJoinedAt(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-open", domainGroup.RestoreGroupName("Open Club"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyOpen,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := authedRequest("POST", "/groups/g-open/join")
	r.SetPathValue("groupId", "g-open")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Join)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body joinGroupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Role != "MEMBER" {
		t.Errorf("Role = %q, want %q", body.Role, "MEMBER")
	}
	if body.JoinedAt == "" {
		t.Error("expected non-empty JoinedAt")
	}
	if _, parseErr := time.Parse("2006-01-02T15:04:05Z", body.JoinedAt); parseErr != nil {
		t.Errorf("JoinedAt %q is not RFC3339 UTC format: %v", body.JoinedAt, parseErr)
	}
}

func TestJoin_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g-open/join", nil)
	r.SetPathValue("groupId", "g-open")
	wrapAuth(http.HandlerFunc(h.Join)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

// --- RequestJoin handler tests ---

func TestRequestJoin_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/requests", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.RequestJoin)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequestJoin_InvalidJSONReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/requests", `{invalid}`)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.RequestJoin)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRequestJoin_EmptyBodyIsValid(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-req", domainGroup.RestoreGroupName("Req Club"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	h := NewHandler(
		appGroup.NewCreateGroupUseCase(repo),
		appGroup.NewListGroupsUseCase(repo, &stubMemberRepo{}),
		appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}),
		appGroup.NewListMyGroupsUseCase(repo, &stubMemberRepo{}, &stubPrefsReader{}),
		appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}),
		appGroup.NewRequestJoinUseCase(repo, &stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewApproveRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubTxManager{}),
		appGroup.NewRejectRequestUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}),
		appGroup.NewListJoinRequestsUseCase(&stubMemberRepo{}, &stubJoinRequestRepo{}, &stubUserProvider{}),
		appGroup.NewGetMyRequestUseCase(&stubJoinRequestRepo{}),
		appGroup.NewCancelMyRequestUseCase(&stubJoinRequestRepo{}),
	)

	r := httptest.NewRequest("POST", "/groups/g-req/requests", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("groupId", "g-req")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RequestJoin)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("empty body should return 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// --- ListJoinRequests handler tests ---

func TestListJoinRequests_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/groups/g1/requests", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.ListJoinRequests)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListJoinRequests_InvalidPageReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedRequest("GET", "/groups/g1/requests?page=abc")
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.ListJoinRequests)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

// --- GetMyRequest handler tests ---

func TestGetMyRequest_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/groups/g1/requests/me", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.GetMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetMyRequest_NoRequestReturns404(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedRequest("GET", "/groups/g1/requests/me")
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.GetMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- CancelMyRequest handler tests ---

func TestCancelMyRequest_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/groups/g1/requests/me", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.CancelMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCancelMyRequest_NoRequestReturns404(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedRequest("DELETE", "/groups/g1/requests/me")
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.CancelMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

// --- UpdateJoinRequest handler tests ---

func TestUpdateJoinRequest_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("PATCH", "/groups/g1/requests/r1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("requestId", "r1")
	wrapAuth(http.HandlerFunc(h.UpdateJoinRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateJoinRequest_InvalidJSONReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/requests/r1", `{bad json}`)
	r.Method = "PATCH"
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("requestId", "r1")
	wrapAuth(http.HandlerFunc(h.UpdateJoinRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateJoinRequest_InvalidStatusReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/requests/r1", `{"status":"PENDING"}`)
	r.Method = "PATCH"
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("requestId", "r1")
	wrapAuth(http.HandlerFunc(h.UpdateJoinRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid status, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", body.Code)
	}
}
