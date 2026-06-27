package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithGetGroup(uc *appGroup.GetGroupUseCase) *Handler {
	return &Handler{getGroup: uc}
}

func TestGetGroup_NotFoundReturns404(t *testing.T) {
	repo := &stubGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) {
			return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
		},
	}
	h := newHandlerWithGetGroup(appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}))

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
	h := newHandlerWithGetGroup(appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}))

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
	h := newHandlerWithGetGroup(appGroup.NewGetGroupUseCase(repo, &stubMemberRepo{}, &stubUserProvider{}))

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
