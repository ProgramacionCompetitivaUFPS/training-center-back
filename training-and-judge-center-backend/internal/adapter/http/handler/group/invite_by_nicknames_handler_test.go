package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
)

func newHandlerWithInviteByNicknames(uc *appGroup.InviteByNicknamesUseCase) *Handler {
	return &Handler{inviteByNicknames: uc}
}

func TestInviteByNicknames_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/invitations/targeted", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.InviteByNicknames)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestInviteByNicknames_NonLeadReturns403(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewInviteByNicknamesUseCase(repo, &mockMemberRepo{}, &mockInvitationRepo{}, &mockNicknameResolver{}, &mockTxManager{}, &mockEmailSender{}, "http://localhost:5173")
	h := newHandlerWithInviteByNicknames(uc)

	r := authedPostRequest("/groups/g1/invitations/targeted", `{"nicknames":["bob"]}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.InviteByNicknames)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInviteByNicknames_InvalidJSONReturns400(t *testing.T) {
	h := mockHandler()
	r := authedPostRequest("/groups/g1/invitations/targeted", `{invalid`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.InviteByNicknames)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestInviteByNicknames_EmptyNicknamesReturns400(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewInviteByNicknamesUseCase(repo, leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, &mockNicknameResolver{}, &mockTxManager{}, &mockEmailSender{}, "http://localhost:5173")
	h := newHandlerWithInviteByNicknames(uc)

	r := authedPostRequest("/groups/g1/invitations/targeted", `{"nicknames":[]}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.InviteByNicknames)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestInviteByNicknames_MixedResultsSuccess(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	resolver := &mockNicknameResolver{users: map[string]*appGroup.UserDisplay{
		"bob": {ID: "invitee-1", Nickname: "bob", Name: "Bob", Email: "bob@example.com"},
	}}
	uc := appGroup.NewInviteByNicknamesUseCase(repo, leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, resolver, &mockTxManager{}, &mockEmailSender{}, "http://localhost:5173")
	h := newHandlerWithInviteByNicknames(uc)

	r := authedPostRequest("/groups/g1/invitations/targeted", `{"nicknames":["bob","ghost"]}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.InviteByNicknames)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body inviteByNicknamesResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(body.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(body.Results))
	}
	byNickname := make(map[string]inviteByNicknamesResultResp, len(body.Results))
	for _, r := range body.Results {
		byNickname[r.Nickname] = r
	}
	if byNickname["bob"].Status != "invited" || byNickname["bob"].Invitation == nil {
		t.Errorf("expected bob to be invited with an invitation, got %+v", byNickname["bob"])
	}
	if byNickname["ghost"].Status != "failed" {
		t.Errorf("expected ghost to be failed, got %+v", byNickname["ghost"])
	}
}
