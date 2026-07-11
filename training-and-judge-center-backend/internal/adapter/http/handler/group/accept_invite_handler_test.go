package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"time"
)

func newHandlerWithAcceptInvite(uc *appGroup.AcceptInviteUseCase) *Handler {
	return &Handler{acceptInvite: uc}
}

func TestAcceptInvite_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/invitations/accept", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAcceptInvite_InvalidJSONReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/invitations/accept", `{invalid json}`)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestAcceptInvite_InvitationNotFoundReturns404(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewAcceptInviteUseCase(repo, &mockMemberRepo{}, &mockInvitationRepo{}, &mockTxManager{})
	h := newHandlerWithAcceptInvite(uc)

	r := authedPostRequest("/groups/g1/invitations/accept", `{"invitationId":"nonexistent"}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAcceptInvite_SuccessReturns201WithRoleMember(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	inv, err := domainGroup.NewGroupInvitation("inv1", "g1", nil, shared.RestoreUserID("lead1"), time.Now())
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	invRepo := &mockInvitationRepo{
		findByIDFn: func(id string) (*domainGroup.GroupInvitation, error) {
			if id == "inv1" {
				// Return a fresh copy each call, matching real repository
				// semantics: an in-memory-only mutation from a prior read
				// (e.g. inv.Accept()) must not leak into a later read.
				return domainGroup.RestoreGroupInvitation(inv.ID(), inv.GroupID(), inv.InviteeID(), inv.InvitedBy(), inv.Status(), inv.ExpiresAt(), inv.CreatedAt()), nil
			}
			return nil, nil
		},
	}
	uc := appGroup.NewAcceptInviteUseCase(repo, &mockMemberRepo{}, invRepo, &mockTxManager{})
	h := newHandlerWithAcceptInvite(uc)

	r := authedPostRequest("/groups/g1/invitations/accept", `{"invitationId":"inv1"}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var body joinGroupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Role != domainGroup.MemberRoleMember.String() {
		t.Errorf("expected role MEMBER, got %s", body.Role)
	}
}

func TestAcceptInvite_WrongInviteeReturns403(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	inviteeID := shared.RestoreUserID("someone-else")
	inv, err := domainGroup.NewGroupInvitation("inv1", "g1", &inviteeID, shared.RestoreUserID("lead1"), time.Now())
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	invRepo := &mockInvitationRepo{
		findByIDFn: func(id string) (*domainGroup.GroupInvitation, error) { return inv, nil },
	}
	uc := appGroup.NewAcceptInviteUseCase(repo, &mockMemberRepo{}, invRepo, &mockTxManager{})
	h := newHandlerWithAcceptInvite(uc)

	r := authedPostRequest("/groups/g1/invitations/accept", `{"invitationId":"inv1"}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}
