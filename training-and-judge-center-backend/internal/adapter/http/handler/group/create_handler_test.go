package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
)

func newHandlerWithCreate(uc *appGroup.CreateGroupUseCase) *Handler {
	return &Handler{createGroup: uc}
}

func TestCreate_InvalidJSONReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{invalid json}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_ContestantReturns403(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Test","joinMode":"OPEN","visibility":"VISIBLE"}`)
	wrapAuth(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestCreate_ValidAdminRequestReturns201(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Algorithms Club","joinMode":"OPEN","visibility":"VISIBLE"}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body groupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Name != "Algorithms Club" {
		t.Errorf("Name = %q, want %q", body.Name, "Algorithms Club")
	}
	if body.ID == "" {
		t.Error("expected non-empty ID in response")
	}
}

func TestCreate_DuplicateNameReturns409(t *testing.T) {
	repo := &mockGroupRepo{
		existsByNameFn: func(_ domainGroup.GroupName) (bool, error) { return true, nil },
	}
	h := newHandlerWithCreate(appGroup.NewCreateGroupUseCase(repo, &mockMemberRepo{}, &mockNicknameResolver{}, &mockTxManager{}))
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Existing Group","joinMode":"OPEN","visibility":"VISIBLE"}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", w.Code)
	}
}

func TestCreate_WithInitialMembersAndLeadsReturns201(t *testing.T) {
	repo := &mockGroupRepo{}
	memberRepo := &mockMemberRepo{}
	nicknameResolver := &mockNicknameResolver{users: map[string]*appGroup.UserDisplay{
		"alice":  {ID: "member-1", Nickname: "alice", Name: "Alice", SystemRole: "CONTESTANT"},
		"coach2": {ID: "lead-1", Nickname: "coach2", Name: "Coach Two", SystemRole: "COACH"},
	}}
	h := newHandlerWithCreate(appGroup.NewCreateGroupUseCase(repo, memberRepo, nicknameResolver, &mockTxManager{}))
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Algorithms Club","joinMode":"OPEN","visibility":"VISIBLE","memberNicknames":["alice"],"leadNicknames":["coach2"]}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body groupResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body.Members) != 2 {
		t.Fatalf("expected 2 members in response, got %d: %+v", len(body.Members), body.Members)
	}
	if len(memberRepo.savedMembers) != 2 {
		t.Errorf("expected SaveAll to persist 2 members, got %d", len(memberRepo.savedMembers))
	}
}

func TestCreate_NicknameNotFoundReturns404(t *testing.T) {
	h := newHandlerWithCreate(appGroup.NewCreateGroupUseCase(&mockGroupRepo{}, &mockMemberRepo{}, &mockNicknameResolver{users: map[string]*appGroup.UserDisplay{}}, &mockTxManager{}))
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Algorithms Club","joinMode":"OPEN","visibility":"VISIBLE","memberNicknames":["ghost"]}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestCreate_ContestantAsLeadReturns400(t *testing.T) {
	nicknameResolver := &mockNicknameResolver{users: map[string]*appGroup.UserDisplay{
		"contestant1": {ID: "c-1", Nickname: "contestant1", SystemRole: "CONTESTANT"},
	}}
	h := newHandlerWithCreate(appGroup.NewCreateGroupUseCase(&mockGroupRepo{}, &mockMemberRepo{}, nicknameResolver, &mockTxManager{}))
	w := httptest.NewRecorder()

	r := authedPostRequest("/groups", `{"name":"Algorithms Club","joinMode":"OPEN","visibility":"VISIBLE","leadNicknames":["contestant1"]}`)
	wrapAuthAsAdmin(http.HandlerFunc(h.Create)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
