package problem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetProblem_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGetProblem(repoReturning(publishedProblem()), &mockUserProviderH{}, &mockFileStorageH{})

	r := httptest.NewRequest(http.MethodGet, "/problems/p/test-problem", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetProblem).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetProblem_IncludesSamplesInResponse(t *testing.T) {
	storage := &mockFileStorageH{
		listFilesFn: func(_ context.Context, _ string) ([]string, error) {
			return []string{
				"problems/test-problem/testcases/xyz/data/sample/1.in",
				"problems/test-problem/testcases/xyz/data/sample/1.ans",
			}, nil
		},
		downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
			return []byte(path), nil
		},
	}
	h := newHandlerWithGetProblem(repoReturning(publishedProblemWithTestCases()), &mockUserProviderH{}, storage)

	r := authedRequest(http.MethodGet, "/problems/p/test-problem", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetProblem)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getProblemResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.Samples) != 1 {
		t.Fatalf("expected 1 sample, got %d", len(resp.Samples))
	}
	if resp.Samples[0].Name != "1" {
		t.Errorf("expected sample name %q, got %q", "1", resp.Samples[0].Name)
	}
}

func TestGetProblem_NoTestCasesReturnsEmptySamples(t *testing.T) {
	h := newHandlerWithGetProblem(repoReturning(publishedProblem()), &mockUserProviderH{}, &mockFileStorageH{})

	r := authedRequest(http.MethodGet, "/problems/p/test-problem", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetProblem)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp getProblemResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.Samples) != 0 {
		t.Errorf("expected no samples, got %d", len(resp.Samples))
	}
}
