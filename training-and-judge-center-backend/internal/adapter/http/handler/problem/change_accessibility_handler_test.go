package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChangeAccessibility_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithChangeAccessibility(repoReturning(publishedProblem()))

	r := httptest.NewRequest(http.MethodPatch, "/problems/p/test-problem/accessibility", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.ChangeAccessibility).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestChangeAccessibility_Success_Returns200(t *testing.T) {
	h := newHandlerWithChangeAccessibility(repoReturning(publishedProblem()))

	r := authedRequest(http.MethodPatch, "/problems/p/test-problem/accessibility", mustJSON(`{"accessibility":"PRIVATE"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ChangeAccessibility)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp changeAccessibilityResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Accessibility != "PRIVATE" {
		t.Errorf("expected accessibility PRIVATE, got %q", resp.Accessibility)
	}
}

func TestChangeAccessibility_InvalidValue_Returns400(t *testing.T) {
	h := newHandlerWithChangeAccessibility(repoReturning(publishedProblem()))

	r := authedRequest(http.MethodPatch, "/problems/p/test-problem/accessibility", mustJSON(`{"accessibility":"RESTRICTED"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ChangeAccessibility)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangeAccessibility_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithChangeAccessibility(&mockProblemRepo{})

	r := authedRequest(http.MethodPatch, "/problems/p/missing/accessibility", mustJSON(`{"accessibility":"PUBLIC"}`))
	r.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.ChangeAccessibility)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
