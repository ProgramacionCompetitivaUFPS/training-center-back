package problem

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRemoveModifier_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithRemoveModifier(repoReturning(draftProblemWithModifier("u2")), userProviderResolvingH("u2"))

	r := httptest.NewRequest(http.MethodDelete, "/problems/p/test-problem/modifiers/coach_mary", nil)
	r.SetPathValue("slug", "test-problem")
	r.SetPathValue("nickname", "coach_mary")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.RemoveModifier).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestRemoveModifier_Forbidden_Returns403(t *testing.T) {
	h := newHandlerWithRemoveModifier(repoReturning(draftProblemWithAuthor("u2")), userProviderResolvingH("u3"))

	r := authedRequest(http.MethodDelete, "/problems/p/test-problem/modifiers/coach_mary", nil)
	r.SetPathValue("slug", "test-problem")
	r.SetPathValue("nickname", "coach_mary")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveModifier_NicknameNotFound_Returns404(t *testing.T) {
	provider := &mockUserProviderH{}
	h := newHandlerWithRemoveModifier(repoReturning(draftProblem()), provider)

	r := authedRequest(http.MethodDelete, "/problems/p/test-problem/modifiers/nonexistent", nil)
	r.SetPathValue("slug", "test-problem")
	r.SetPathValue("nickname", "nonexistent")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRemoveModifier_Success_Returns204(t *testing.T) {
	h := newHandlerWithRemoveModifier(repoReturning(draftProblemWithModifier("u2")), userProviderResolvingH("u2"))

	r := authedRequest(http.MethodDelete, "/problems/p/test-problem/modifiers/coach_mary", nil)
	r.SetPathValue("slug", "test-problem")
	r.SetPathValue("nickname", "coach_mary")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.RemoveModifier)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}
