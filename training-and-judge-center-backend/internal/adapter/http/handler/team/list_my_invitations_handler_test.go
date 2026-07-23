package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

func newHandlerWithListInvitations(invRepo domainTeam.InvitationRepository, teamRepo domainTeam.Repository, up appTeam.UserProvider) *Handler {
	return &Handler{listMyInvitations: appTeam.NewListMyInvitationsUseCase(invRepo, teamRepo, up)}
}

func TestListMyInvitations_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithListInvitations(&mockInvitationRepo{}, &mockTeamRepo{}, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/users/me/team-invitations", nil)
	wrapAuth(http.HandlerFunc(h.ListMyInvitations)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListMyInvitations_EmptyListReturns200(t *testing.T) {
	h := newHandlerWithListInvitations(&mockInvitationRepo{}, &mockTeamRepo{}, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := authedGetRequest("/users/me/team-invitations")
	wrapAuth(http.HandlerFunc(h.ListMyInvitations)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body listMyInvitationsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(body.Invitations) != 0 {
		t.Errorf("expected empty list, got %d", len(body.Invitations))
	}
}

func TestListMyInvitations_ReturnsList(t *testing.T) {
	now := time.Now()
	inv := domainTeam.RestoreTeamInvitation("inv1", "t1",
		domainShared.RestoreUserID("u1"),
		domainShared.RestoreUserID("u2"),
		now)

	invRepo := &mockInvitationRepo{
		findByInviteeFn: func(_ domainShared.UserID) ([]*domainTeam.TeamInvitation, error) {
			return []*domainTeam.TeamInvitation{inv}, nil
		},
	}
	teamRepo := &mockTeamRepo{
		findByIDsFn: func(_ []string) ([]*domainTeam.Team, error) {
			return []*domainTeam.Team{makeHandlerTeam("t1", "Alpha", "u2", now)}, nil
		},
	}

	h := newHandlerWithListInvitations(invRepo, teamRepo, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := authedGetRequest("/users/me/team-invitations")
	wrapAuth(http.HandlerFunc(h.ListMyInvitations)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body listMyInvitationsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(body.Invitations) != 1 {
		t.Fatalf("expected 1 invitation, got %d", len(body.Invitations))
	}
	if body.Invitations[0].Team.Name != "Alpha" {
		t.Errorf("Team.Name = %q, want Alpha", body.Invitations[0].Team.Name)
	}
}
