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
	"github.com/training-judge-center/backend/pkg/apperror"
)

// The authenticated user in these tests is "u1".

func newHandlerWithRename(teamRepo domainTeam.Repository, memberRepo domainTeam.MemberRepository) *Handler {
	return &Handler{renameTeam: appTeam.NewRenameTeamUseCase(teamRepo, memberRepo)}
}

func existingTeamRepo(updateName func(t *domainTeam.Team) error) *mockTeamRepo {
	return &mockTeamRepo{
		findByIDFn: func(id string) (*domainTeam.Team, error) {
			return domainTeam.RestoreTeam(id, domainTeam.RestoreTeamName("Alpha"), domainShared.RestoreUserID("creator"), time.Now()), nil
		},
		updateNameFn: updateName,
	}
}

func u1IsMember() *mockMemberRepo {
	return &mockMemberRepo{
		findByTeamAndUserFn: func(teamID string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", teamID, userID.Value(), time.Now()), nil
		},
	}
}

func renameRequest(body string) *http.Request {
	r := authedPostRequest("/teams/t1", body)
	r.Method = http.MethodPatch
	return withChiParam(r, "teamId", "t1")
}

func TestRenameTeam_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithRename(&mockTeamRepo{}, &mockMemberRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/teams/t1", nil)
	wrapAuth(http.HandlerFunc(h.RenameTeam)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRenameTeam_InvalidJSONReturns400(t *testing.T) {
	h := newHandlerWithRename(existingTeamRepo(nil), u1IsMember())
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.RenameTeam)).ServeHTTP(w, renameRequest("{not json"))
	if w.Code != http.StatusBadRequest && w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected a 4xx validation error, got %d", w.Code)
	}
}

func TestRenameTeam_MemberRenamesAndGetsTheNewName(t *testing.T) {
	var saved string
	h := newHandlerWithRename(existingTeamRepo(func(team *domainTeam.Team) error {
		saved = team.Name().Value()
		return nil
	}), u1IsMember())
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RenameTeam)).ServeHTTP(w, renameRequest(`{"name":"  Beta Squad "}`))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.ID != "t1" || body.Name != "Beta Squad" || saved != "Beta Squad" {
		t.Errorf("expected t1 renamed to the trimmed name, got body=%+v saved=%q", body, saved)
	}
}

func TestRenameTeam_NonMemberReturns403(t *testing.T) {
	h := newHandlerWithRename(existingTeamRepo(nil), &mockMemberRepo{})
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RenameTeam)).ServeHTTP(w, renameRequest(`{"name":"Gamma"}`))

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRenameTeam_NameTakenReturns409(t *testing.T) {
	h := newHandlerWithRename(existingTeamRepo(func(_ *domainTeam.Team) error {
		return apperror.NewConflict(domainTeam.ErrCodeTeamNameExists, "A team with this name already exists")
	}), u1IsMember())
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RenameTeam)).ServeHTTP(w, renameRequest(`{"name":"Taken"}`))

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != domainTeam.ErrCodeTeamNameExists {
		t.Errorf("expected %s, got %s", domainTeam.ErrCodeTeamNameExists, body.Code)
	}
}
