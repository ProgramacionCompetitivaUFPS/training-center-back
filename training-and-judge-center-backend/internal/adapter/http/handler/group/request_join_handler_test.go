package group

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newHandlerWithRequestJoin(uc *appGroup.RequestJoinUseCase) *Handler {
	return &Handler{requestJoin: uc}
}

func TestRequestJoin_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/requests", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.RequestJoin)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRequestJoin_InvalidJSONReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/requests", `{invalid}`)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.RequestJoin)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestRequestJoin_EmptyBodyIsValid(t *testing.T) {
	g := domainGroup.RestoreGroup(
		"g-req", domainGroup.RestoreGroupName("Req Club"), nil,
		domainGroup.VisibilityVisible, domainGroup.JoinPolicyRequest,
		false, shared.RestoreUserID("author-1"), testTime(), testTime(),
	)
	repo := &mockGroupRepo{
		findByIDFn: func(_ string) (*domainGroup.Group, error) { return g, nil },
	}
	h := newHandlerWithRequestJoin(appGroup.NewRequestJoinUseCase(repo, &mockMemberRepo{}, &mockJoinRequestRepo{}))

	r := httptest.NewRequest("POST", "/groups/g-req/requests", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("groupId", "g-req")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RequestJoin)).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("empty body should return 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
