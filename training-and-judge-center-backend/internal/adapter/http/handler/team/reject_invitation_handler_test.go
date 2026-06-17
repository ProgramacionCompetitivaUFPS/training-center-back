package team

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

func newHandlerWithReject(invRepo domainTeam.InvitationRepository) *Handler {
	return &Handler{rejectInvitation: appTeam.NewRejectInvitationUseCase(invRepo)}
}

func TestRejectInvitation_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithReject(&mockInvitationRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/team-invitations/inv1", nil)
	wrapAuth(http.HandlerFunc(h.RejectInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRejectInvitation_NotFoundReturns404(t *testing.T) {
	h := newHandlerWithReject(&mockInvitationRepo{})
	w := httptest.NewRecorder()
	r := authedGetRequest("/team-invitations/missing")
	r.Method = "DELETE"
	wrapAuth(http.HandlerFunc(h.RejectInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestRejectInvitation_SuccessReturns204(t *testing.T) {
	invRepo := &mockInvitationRepo{
		findByIDFn: func(_ string) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1",
				domainShared.RestoreUserID("u1"),
				domainShared.RestoreUserID("u2"),
				time.Now()), nil
		},
	}
	h := newHandlerWithReject(invRepo)
	w := httptest.NewRecorder()
	r := authedGetRequest("/team-invitations/inv1")
	r.Method = "DELETE"
	wrapAuth(http.HandlerFunc(h.RejectInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
