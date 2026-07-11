package group

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newHandlerWithRevokeInvitation(uc *appGroup.RevokeInvitationUseCase) *Handler {
	return &Handler{revokeInvitation: uc}
}

func TestRevokeInvitation_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/groups/g1/invitations/inv1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("invitationId", "inv1")
	wrapAuth(http.HandlerFunc(h.RevokeInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRevokeInvitation_NonLeadReturns403(t *testing.T) {
	uc := appGroup.NewRevokeInvitationUseCase(&mockMemberRepo{}, &mockInvitationRepo{})
	h := newHandlerWithRevokeInvitation(uc)

	r := authedRequest("DELETE", "/groups/g1/invitations/inv1")
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("invitationId", "inv1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RevokeInvitation)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevokeInvitation_NotFoundReturns404(t *testing.T) {
	uc := appGroup.NewRevokeInvitationUseCase(leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{})
	h := newHandlerWithRevokeInvitation(uc)

	r := authedRequest("DELETE", "/groups/g1/invitations/nonexistent")
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("invitationId", "nonexistent")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RevokeInvitation)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRevokeInvitation_SuccessReturns204(t *testing.T) {
	inv, err := domainGroup.NewGroupInvitation("inv1", "g1", nil, shared.RestoreUserID("lead1"), time.Now())
	if err != nil {
		t.Fatalf("NewGroupInvitation: %v", err)
	}
	var transitioned bool
	invRepo := &mockInvitationRepo{
		findByIDFn: func(id string) (*domainGroup.GroupInvitation, error) {
			if id == "inv1" {
				return inv, nil
			}
			return nil, nil
		},
		transitionStatusFn: func(id string, from, to domainGroup.InvitationStatus) error {
			transitioned = true
			return nil
		},
	}
	uc := appGroup.NewRevokeInvitationUseCase(leadMemberRepoHandler("g1", "u1"), invRepo)
	h := newHandlerWithRevokeInvitation(uc)

	r := authedRequest("DELETE", "/groups/g1/invitations/inv1")
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("invitationId", "inv1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RevokeInvitation)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
	if !transitioned {
		t.Error("expected TransitionStatus to be called")
	}
}
