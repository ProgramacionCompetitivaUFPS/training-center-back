package group

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGenerateInvite_UnauthenticatedReturns401(t *testing.T) {
	h := stubHandler()
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/groups/g1/invitations", nil)
	r.SetPathValue("groupId", "g1")
	wrapAuth(http.HandlerFunc(h.GenerateInvite)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}
