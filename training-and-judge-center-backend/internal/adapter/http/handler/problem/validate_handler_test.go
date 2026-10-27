package problem

import (
	"net/http"
	"net/http/httptest"
	"testing"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

func newHandlerWithValidate(repo *mockProblemRepo, validationRepo *mockValidationRepositoryH, queue *mockValidationQueueH) *Handler {
	return &Handler{requestProblemValidation: appProblem.NewRequestProblemValidationUseCase(repo, validationRepo, queue)}
}

func TestValidate_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithValidate(repoReturning(completeDraftProblem()), &mockValidationRepositoryH{}, &mockValidationQueueH{})

	r := httptest.NewRequest(http.MethodPost, "/problems/p/test-problem/validate", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Validate).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestValidate_CompleteDraft_Returns202WithoutWaiting(t *testing.T) {
	queue := &mockValidationQueueH{}
	h := newHandlerWithValidate(repoReturning(completeDraftProblem()), &mockValidationRepositoryH{}, queue)

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/validate", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Validate)).ServeHTTP(w, r)

	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidate_IncompleteDraft_Returns400WithMissingFields(t *testing.T) {
	h := newHandlerWithValidate(repoReturning(draftProblem()), &mockValidationRepositoryH{}, &mockValidationQueueH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/validate", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Validate)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestValidate_AlreadyPublished_Returns409(t *testing.T) {
	h := newHandlerWithValidate(repoReturning(publishedProblem()), &mockValidationRepositoryH{}, &mockValidationQueueH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/validate", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Validate)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
}
