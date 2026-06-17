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

func newHandlerWithAccept(invRepo domainTeam.InvitationRepository, memberRepo domainTeam.MemberRepository) *Handler {
	return &Handler{acceptInvitation: appTeam.NewAcceptInvitationUseCase(invRepo, memberRepo, &mockTxManager{})}
}

func TestAcceptInvitation_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithAccept(&mockInvitationRepo{}, &mockMemberRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/team-invitations/inv1/accept", nil)
	wrapAuth(http.HandlerFunc(h.AcceptInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAcceptInvitation_NotFoundReturns404(t *testing.T) {
	h := newHandlerWithAccept(&mockInvitationRepo{}, &mockMemberRepo{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/team-invitations/missing/accept", "")
	wrapAuth(http.HandlerFunc(h.AcceptInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestAcceptInvitation_SuccessReturns200(t *testing.T) {
	invRepo := &mockInvitationRepo{
		findByIDFn: func(_ string) (*domainTeam.TeamInvitation, error) {
			return domainTeam.RestoreTeamInvitation("inv1", "t1",
				domainShared.RestoreUserID("u1"),
				domainShared.RestoreUserID("u2"),
				time.Now()), nil
		},
	}
	h := newHandlerWithAccept(invRepo, &mockMemberRepo{})
	w := httptest.NewRecorder()
	r := authedPostRequest("/team-invitations/inv1/accept", "")
	wrapAuth(http.HandlerFunc(h.AcceptInvitation)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
