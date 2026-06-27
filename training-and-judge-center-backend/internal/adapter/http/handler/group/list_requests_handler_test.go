package group

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListJoinRequests_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/groups/g1/requests", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.ListJoinRequests)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListJoinRequests_InvalidPageReturns400(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedRequest("GET", "/groups/g1/requests?page=abc")
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.ListJoinRequests)).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
