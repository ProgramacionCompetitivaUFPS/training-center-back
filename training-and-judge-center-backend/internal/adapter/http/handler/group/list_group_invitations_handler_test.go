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
)

func newHandlerWithListGroupInvitations(uc *appGroup.ListGroupInvitationsUseCase) *Handler {
	return &Handler{listGroupInvitations: uc}
}

func TestListGroupInvitations_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/groups/g1/invitations", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.ListGroupInvitations)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListGroupInvitations_NonLeadReturns403(t *testing.T) {
	uc := appGroup.NewListGroupInvitationsUseCase(&mockMemberRepo{}, &mockInvitationRepo{}, &mockUserProvider{})
	h := newHandlerWithListGroupInvitations(uc)

	r := authedRequest("GET", "/groups/g1/invitations")
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroupInvitations)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestListGroupInvitations_InvalidPageReturns400(t *testing.T) {
	uc := appGroup.NewListGroupInvitationsUseCase(leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, &mockUserProvider{})
	h := newHandlerWithListGroupInvitations(uc)

	r := authedRequest("GET", "/groups/g1/invitations?page=abc")
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroupInvitations)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListGroupInvitations_SuccessReturnsInvitations(t *testing.T) {
	inv, err := domainGroup.NewGroupInvitation("inv1", "g1", nil, shared.RestoreUserID("lead1"), time.Now())
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	invRepo := &mockInvitationRepo{
		findByGroupFn: func(_ string, _ domainGroup.InvitationFilters) ([]*domainGroup.GroupInvitation, int, error) {
			return []*domainGroup.GroupInvitation{inv}, 1, nil
		},
	}
	userProvider := &mockUserProvider{}
	uc := appGroup.NewListGroupInvitationsUseCase(leadMemberRepoHandler("g1", "u1"), invRepo, userProvider)
	h := newHandlerWithListGroupInvitations(uc)

	r := authedRequest("GET", "/groups/g1/invitations")
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListGroupInvitations)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body listGroupInvitationsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body.Invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(body.Invitations))
	}
	if body.Invitations[0].ID != "inv1" {
		t.Errorf("expected id inv1, got %s", body.Invitations[0].ID)
	}
	if body.Invitations[0].EffectiveStatus != domainGroup.InvitationStatusPending.String() {
		t.Errorf("expected effectiveStatus PENDING, got %s", body.Invitations[0].EffectiveStatus)
	}
}
