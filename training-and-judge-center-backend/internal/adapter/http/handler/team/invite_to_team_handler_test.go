package team

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func withChiParam(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.NewRouteContext()
	chiCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

func newHandlerWithInvite(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository, invRepo domainTeam.InvitationRepository, up appTeam.UserProvider) *Handler {
	return &Handler{inviteToTeam: appTeam.NewInviteToTeamUseCase(teamRepo, memberRepo, invRepo, up)}
}

func TestInviteToTeam_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithInvite(&mockTeamRepo{}, &mockMemberRepo{}, &mockInvitationRepo{}, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/teams/t1/invitations", nil)
	wrapAuth(http.HandlerFunc(h.InviteToTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestInviteToTeam_NotMemberReturns403(t *testing.T) {
	teamRepo := &mockTeamRepo{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeHandlerTeam("t1", "Alpha", "other", time.Now()), nil
		},
	}
	h := newHandlerWithInvite(teamRepo, &mockMemberRepo{}, &mockInvitationRepo{}, &mockUserProvider{})
	w := httptest.NewRecorder()
	r := withChiParam(authedPostRequest("/teams/t1/invitations", `{"nickname":"alice"}`), "teamId", "t1")
	wrapAuth(http.HandlerFunc(h.InviteToTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestInviteToTeam_SuccessReturns201(t *testing.T) {
	teamRepo := &mockTeamRepo{
		findByIDFn: func(_ string) (*domainTeam.Team, error) {
			return makeHandlerTeam("t1", "Alpha", "u1", time.Now()), nil
		},
	}
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			if userID.Value() == "u1" {
				return makeHandlerMember("m1", "t1", "u1", time.Now()), nil
			}
			return nil, apperror.NewNotFound(domainTeam.ErrCodeNotTeamMember, "not a member")
		},
	}
	up := &mockUserProvider{
		findByNicknameFn: func(_ string) (*appTeam.UserDisplay, error) {
			return &appTeam.UserDisplay{ID: "u2", Nickname: "alice"}, nil
		},
	}
	h := newHandlerWithInvite(teamRepo, memberRepo, &mockInvitationRepo{}, up)
	w := httptest.NewRecorder()
	r := withChiParam(authedPostRequest("/teams/t1/invitations", `{"nickname":"alice"}`), "teamId", "t1")
	wrapAuth(http.HandlerFunc(h.InviteToTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
