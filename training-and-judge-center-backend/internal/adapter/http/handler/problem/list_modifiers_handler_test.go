package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListModifiers_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithListModifiers(repoReturning(draftProblemWithModifier("u2")), &mockUserProviderH{})

	r := httptest.NewRequest(http.MethodGet, "/problems/p/test-problem/modifiers", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.ListModifiers).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListModifiers_Success_ReturnsNicknameAndName(t *testing.T) {
	provider := &mockUserProviderH{}
	h := newHandlerWithListModifiers(repoReturning(draftProblemWithModifier("u2")), provider)

	r := authedRequest(http.MethodGet, "/problems/p/test-problem/modifiers", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ListModifiers)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listModifiersResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.Modifiers) != 1 {
		t.Fatalf("expected 1 modifier, got %d", len(resp.Modifiers))
	}
	if resp.Modifiers[0].Nickname == "" {
		t.Errorf("expected non-empty nickname, got %+v", resp.Modifiers[0])
	}
}
