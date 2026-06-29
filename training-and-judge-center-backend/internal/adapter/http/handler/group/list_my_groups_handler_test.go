package group

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListMyGroups_NonIntegerPageReturns400(t *testing.T) {
	h := mockHandler()
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListMyGroups)).ServeHTTP(w, authedRequest("GET", "/users/me/groups?page=abc"))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
