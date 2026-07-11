package group

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newHandlerWithGenerateInvite(uc *appGroup.GenerateInviteUseCase) *Handler {
	return &Handler{generateInvite: uc}
}

func inviteGroupFixture() *domainGroup.Group {
	return domainGroup.RestoreGroup(
		"g1", domainGroup.RestoreGroupName("Invite Club"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyInvite,
		false, shared.RestoreUserID("creator-id"), testTime(), testTime(),
	)
}

func leadMemberRepoHandler(groupID, userID string) *mockMemberRepo {
	lead := domainGroup.RestoreGroupMember("m-lead", groupID, shared.RestoreUserID(userID), domainGroup.MemberRoleLead, testTime(), nil, domainGroup.JoinMethodDirectAdd)
	return &mockMemberRepo{
		findByGroupAndUserFn: func(gid string, uid shared.UserID) (*domainGroup.GroupMember, error) {
			if gid == groupID && uid.Value() == userID {
				return lead, nil
			}
			return nil, nil
		},
	}
}

func TestGenerateInvite_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/invitations", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGenerateInvite_NonLeadReturns403(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewGenerateInviteUseCase(repo, &mockMemberRepo{}, &mockInvitationRepo{}, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, &mockTxManager{})
	h := newHandlerWithGenerateInvite(uc)

	r := authedPostRequest("/groups/g1/invitations", `{}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGenerateInvite_InvalidJSONReturns400(t *testing.T) {
	h := mockHandler()
	r := authedPostRequest("/groups/g1/invitations", `{invalid`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGenerateInvite_GeneralInvitationSuccess(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewGenerateInviteUseCase(repo, leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, &mockTxManager{})
	h := newHandlerWithGenerateInvite(uc)

	r := authedPostRequest("/groups/g1/invitations", `{}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var body generateInviteResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Invitee != nil {
		t.Errorf("expected nil invitee for general invitation, got %+v", body.Invitee)
	}
	if body.Status != domainGroup.InvitationStatusPending.String() {
		t.Errorf("expected status PENDING, got %s", body.Status)
	}
	if body.ExpiresAt == "" {
		t.Error("expected non-empty expiresAt")
	}
}

func TestGenerateInvite_PersonalInvitationByNicknameSuccess(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	display := &appGroup.UserDisplay{ID: "invitee-1", Nickname: "bob", Name: "Bob", Email: "bob@example.com"}
	uc := appGroup.NewGenerateInviteUseCase(repo, leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, &mockNicknameResolver{user: display}, &mockEmailResolver{}, &mockUserProvider{}, &mockTxManager{})
	h := newHandlerWithGenerateInvite(uc)

	r := authedPostRequest("/groups/g1/invitations", `{"userNickname":"bob"}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var body generateInviteResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if body.Invitee == nil || body.Invitee.Nickname != "bob" {
		t.Errorf("expected invitee nickname 'bob', got %+v", body.Invitee)
	}
}

func TestGenerateInvite_NicknameNotFoundReturns404(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewGenerateInviteUseCase(repo, leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, &mockTxManager{})
	h := newHandlerWithGenerateInvite(uc)

	r := authedPostRequest("/groups/g1/invitations", `{"userNickname":"ghost"}`)
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGenerateInvite_EmptyBodyIsValidGeneralInvitation(t *testing.T) {
	repo := &mockGroupRepo{findByIDFn: func(_ string) (*domainGroup.Group, error) { return inviteGroupFixture(), nil }}
	uc := appGroup.NewGenerateInviteUseCase(repo, leadMemberRepoHandler("g1", "u1"), &mockInvitationRepo{}, &mockNicknameResolver{}, &mockEmailResolver{}, &mockUserProvider{}, &mockTxManager{})
	h := newHandlerWithGenerateInvite(uc)

	r := authedRequest("POST", "/groups/g1/invitations")
	r.SetPathValue("groupId", "g1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for empty body, got %d: %s", w.Code, w.Body.String())
	}
}
