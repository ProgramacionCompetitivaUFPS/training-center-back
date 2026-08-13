package problem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
)

func TestGetLatestValidation_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGetLatestValidation(repoReturning(draftProblem()), &mockValidationRepositoryH{}, &mockProblemStatusProviderH{})

	r := httptest.NewRequest(http.MethodGet, "/problems/p/test-problem/validation", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.GetLatestValidation).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetLatestValidation_ProblemNotFound_Returns404(t *testing.T) {
	h := newHandlerWithGetLatestValidation(&mockProblemRepo{}, &mockValidationRepositoryH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodGet, "/problems/p/missing/validation", nil)
	r.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetLatestValidation)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetLatestValidation_NoAttemptsYet_Returns200WithFoundFalse(t *testing.T) {
	validationRepo := &mockValidationRepositoryH{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
	}
	h := newHandlerWithGetLatestValidation(repoReturning(draftProblem()), validationRepo, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodGet, "/problems/p/test-problem/validation", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetLatestValidation)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp latestValidationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Found {
		t.Error("expected found=false")
	}
}

func TestGetLatestValidation_Found_Returns200WithDecodedStatus(t *testing.T) {
	v := passedValidation("validation-1", `{"validationLogs":["ok"],"validationSummary":{"sampleCases":1,"secretCases":2,"solutionsTested":1,"allPassed":true}}`)
	validationRepo := &mockValidationRepositoryH{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return v, true, nil
		},
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return v, nil
		},
	}
	statusProvider := &mockProblemStatusProviderH{
		getStatusFn: func(_ context.Context, _ string) (string, error) { return "PUBLISHED", nil },
	}
	h := newHandlerWithGetLatestValidation(repoReturning(draftProblem()), validationRepo, statusProvider)

	r := authedRequest(http.MethodGet, "/problems/p/test-problem/validation", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.GetLatestValidation)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp latestValidationResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if !resp.Found || !resp.Terminal || !resp.Passed || resp.Status != "PUBLISHED" {
		t.Errorf("resp: got %+v", resp)
	}
	if resp.ValidationSummary == nil || resp.ValidationSummary.SecretCases != 2 {
		t.Errorf("ValidationSummary: got %v", resp.ValidationSummary)
	}
}
