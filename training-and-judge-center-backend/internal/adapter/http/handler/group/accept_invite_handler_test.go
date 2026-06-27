package group

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAcceptInvite_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/invitations/accept", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAcceptInvite_InvalidJSONReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedPostRequest("/groups/g1/invitations/accept", `{invalid json}`)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.AcceptInvite)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
