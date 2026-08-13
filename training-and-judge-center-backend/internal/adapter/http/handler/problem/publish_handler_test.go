package problem

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newValidation(id string) *domainProblem.ProblemValidation {
	v, err := domainProblem.NewProblemValidation(id, "p1", shared.RestoreUserID("u1"), testNow)
	if err != nil {
		panic(err)
	}
	return v
}

func passedValidation(id, resultJSON string) *domainProblem.ProblemValidation {
	v := newValidation(id)
	if err := v.Start(testNow); err != nil {
		panic(err)
	}
	if err := v.MarkPassed(resultJSON, testNow); err != nil {
		panic(err)
	}
	return v
}

func failedValidation(id, resultJSON string) *domainProblem.ProblemValidation {
	v := newValidation(id)
	if err := v.Start(testNow); err != nil {
		panic(err)
	}
	if err := v.MarkFailed(resultJSON, testNow); err != nil {
		panic(err)
	}
	return v
}

func TestPublish_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithPublish(repoReturning(draftProblem()), &mockValidationRepositoryH{}, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := httptest.NewRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Publish).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestPublish_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithPublish(&mockProblemRepo{}, &mockValidationRepositoryH{}, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/missing/publish", nil)
	r.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPublish_AlreadyPublished_Returns409(t *testing.T) {
	h := newHandlerWithPublish(repoReturning(publishedProblem()), &mockValidationRepositoryH{}, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var errResp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("could not decode error response: %v", err)
	}
	if errResp.Code != domainProblem.ErrCodeAlreadyPublished {
		t.Errorf("expected %s, got %s", domainProblem.ErrCodeAlreadyPublished, errResp.Code)
	}
}

func TestPublish_MissingFields_Returns400(t *testing.T) {
	h := newHandlerWithPublish(repoReturning(draftProblem()), &mockValidationRepositoryH{}, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp publishFailureResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.MissingFields) == 0 {
		t.Error("expected MissingFields to be populated")
	}
}

func TestPublish_CompleteProblem_PassesImmediately_Returns200(t *testing.T) {
	validationRepo := &mockValidationRepositoryH{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
		findByIDFn: func(_ context.Context, id string) (*domainProblem.ProblemValidation, error) {
			return passedValidation(id, `{"validationLogs":["compiled ok","2/2 test cases passed"]}`), nil
		},
	}
	h := newHandlerWithPublish(repoReturning(completeDraftProblem()), validationRepo, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp publishResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Status != "PUBLISHED" {
		t.Errorf("Status: got %q, want PUBLISHED", resp.Status)
	}
	if len(resp.ValidationLogs) != 2 {
		t.Errorf("ValidationLogs: got %v, want 2 entries", resp.ValidationLogs)
	}
}

func TestPublish_CompleteProblem_FailsValidation_Returns400(t *testing.T) {
	validationRepo := &mockValidationRepositoryH{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
		findByIDFn: func(_ context.Context, id string) (*domainProblem.ProblemValidation, error) {
			return failedValidation(id, `{"validationLogs":["compiled ok"],"failedTestCases":[{"case":"secret/01","verdict":"WRONG_ANSWER"}]}`), nil
		},
	}
	h := newHandlerWithPublish(repoReturning(completeDraftProblem()), validationRepo, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp publishFailureResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if len(resp.FailedTestCases) != 1 || resp.FailedTestCases[0].Case != "secret/01" {
		t.Errorf("FailedTestCases: got %v, want one entry for secret/01", resp.FailedTestCases)
	}
	if resp.Message != "Solution failed test cases" {
		t.Errorf("Message: got %q, want %q", resp.Message, "Solution failed test cases")
	}
}

func TestPublish_StatusLookupFails_Returns500(t *testing.T) {
	validationRepo := &mockValidationRepositoryH{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return nil, apperror.NewInternal()
		},
	}
	h := newHandlerWithPublish(repoReturning(completeDraftProblem()), validationRepo, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d: %s", w.Code, w.Body.String())
	}
}

// TestPublish_ClientDisconnected_WritesNothing covers the branch that's new
// to the handler now that awaiting the validation is a single call instead
// of a loop it owns: if the request context is already canceled by the time
// the use case returns an error, the handler must not attempt to write a
// response — there is nobody left to receive it.
func TestPublish_ClientDisconnected_WritesNothing(t *testing.T) {
	validationRepo := &mockValidationRepositoryH{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return nil, false, nil
		},
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return nil, apperror.NewInternal()
		},
	}
	h := newHandlerWithPublish(repoReturning(completeDraftProblem()), validationRepo, &mockValidationQueueH{}, &mockProblemStatusProviderH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/publish", nil)
	r.SetPathValue("slug", "test-problem")
	ctx, cancel := context.WithCancel(r.Context())
	cancel() // simulate the client disconnecting before the handler responds
	r = r.WithContext(ctx)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Publish)).ServeHTTP(w, r)

	if w.Body.Len() != 0 {
		t.Errorf("expected no response body to be written, got %q", w.Body.String())
	}
}
