package group

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCancelMyRequest_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("DELETE", "/groups/g1/requests/me", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.CancelMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestCancelMyRequest_NoRequestReturns404(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := authedRequest("DELETE", "/groups/g1/requests/me")
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.CancelMyRequest)).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
