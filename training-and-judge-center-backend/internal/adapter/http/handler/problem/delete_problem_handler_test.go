package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestDeleteProblem_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithDeleteProblem(repoReturning(draftProblem()), &mockActiveContestCheckerH{})

	r := httptest.NewRequest(http.MethodDelete, "/problems/p/test-problem", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.DeleteProblem).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestDeleteProblem_Success_Returns204(t *testing.T) {
	h := newHandlerWithDeleteProblem(repoReturning(draftProblem()), &mockActiveContestCheckerH{})

	r := authedRequest(http.MethodDelete, "/problems/p/test-problem", mustJSON(`{"confirmSlug":"test-problem"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.DeleteProblem)).ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteProblem_ProblemInActiveContest_Returns409(t *testing.T) {
	h := newHandlerWithDeleteProblem(repoReturning(publishedProblem()), &mockActiveContestCheckerH{inActive: true})

	r := authedRequest(http.MethodDelete, "/problems/p/test-problem", mustJSON(`{"confirmSlug":"test-problem"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.DeleteProblem)).ServeHTTP(w, r)

	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d: %s", w.Code, w.Body.String())
	}
	var errResp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("could not decode error response: %v", err)
	}
	if errResp.Code != domainProblem.ErrCodeProblemInActiveContest {
		t.Errorf("expected %s, got %s", domainProblem.ErrCodeProblemInActiveContest, errResp.Code)
	}
}

func TestDeleteProblem_SlugMismatch_Returns400(t *testing.T) {
	h := newHandlerWithDeleteProblem(repoReturning(draftProblem()), &mockActiveContestCheckerH{})

	r := authedRequest(http.MethodDelete, "/problems/p/test-problem", mustJSON(`{"confirmSlug":"wrong-slug"}`))
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.DeleteProblem)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var errResp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&errResp); err != nil {
		t.Fatalf("could not decode error response: %v", err)
	}
	if errResp.Code != domainProblem.ErrCodeSlugMismatch {
		t.Errorf("expected %s, got %s", domainProblem.ErrCodeSlugMismatch, errResp.Code)
	}
}
