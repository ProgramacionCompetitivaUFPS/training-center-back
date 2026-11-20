package group

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetMyRequest_UnauthenticatedReturns401(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/groups/g1/requests/me", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.GetMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetMyRequest_NoRequestReturns200WithNullRequest(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()
	r := authedRequest("GET", "/groups/g1/requests/me")
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.GetMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"request":null`) {
		t.Errorf("expected null request in body, got %s", w.Body.String())
	}
}
