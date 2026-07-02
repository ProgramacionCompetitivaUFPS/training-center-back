package problem

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAddModifier_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithAddModifier(repoReturning(draftProblem()), userProviderResolvingH("u2"))

	r := httptest.NewRequest(http.MethodPost, "/problems/p/test-problem/modifiers", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.AddModifier).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestAddModifier_Forbidden_Returns403(t *testing.T) {
	h := newHandlerWithAddModifier(repoReturning(draftProblemWithAuthor("u2")), userProviderResolvingH("u3"))

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/modifiers", mustJSON(`{"userNickname":"coach_mary"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AddModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddModifier_MissingNickname_Returns400(t *testing.T) {
	h := newHandlerWithAddModifier(repoReturning(draftProblem()), userProviderResolvingH("u2"))

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/modifiers", mustJSON(`{}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AddModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddModifier_NicknameNotFound_Returns404(t *testing.T) {
	provider := &mockUserProviderH{}
	h := newHandlerWithAddModifier(repoReturning(draftProblem()), provider)

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/modifiers", mustJSON(`{"userNickname":"nonexistent"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AddModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestAddModifier_Success_Returns204(t *testing.T) {
	h := newHandlerWithAddModifier(repoReturning(draftProblem()), userProviderResolvingH("u2"))

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/modifiers", mustJSON(`{"userNickname":"coach_mary"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.AddModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
