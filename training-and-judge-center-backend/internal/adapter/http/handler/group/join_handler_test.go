package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithJoin(uc *appGroup.JoinGroupUseCase) *Handler {
	return &Handler{joinGroup: uc}
}

func TestJoin_GroupNotFoundReturns404(t *testing.T) {
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		},
	}
	h := newHandlerWithJoin(appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}))

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
	h := newHandlerWithJoin(appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}))

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
	existingMember := domainGroup.RestoreGroupMember("m1", "g-open", userID, domainGroup.MemberRoleMember, testTime(), nil, domainGroup.JoinMethodOpenJoin)
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	memberRepo := &stubMemberRepo{
		findByGroupAndUserFn: func(_ string, _ shared.UserID) (*domainGroup.GroupMember, error) {
			return existingMember, nil
		},
	}
	h := newHandlerWithJoin(appGroup.NewJoinGroupUseCase(repo, memberRepo))

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
	h := newHandlerWithJoin(appGroup.NewJoinGroupUseCase(repo, &stubMemberRepo{}))

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
