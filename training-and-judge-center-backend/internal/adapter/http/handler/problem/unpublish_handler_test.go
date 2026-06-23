package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUnpublish_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUnpublish(repoReturning(publishedProblem()), &mockActiveContestCheckerH{})

	r := httptest.NewRequest(http.MethodPost, "/problems/p/test-problem/unpublish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Unpublish).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUnpublish_Success_Returns200(t *testing.T) {
	h := newHandlerWithUnpublish(repoReturning(publishedProblem()), &mockActiveContestCheckerH{})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/unpublish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpublish)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp unpublishResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Status != "DRAFT" {
		t.Errorf("expected status DRAFT, got %q", resp.Status)
	}
}

func TestUnpublish_ProblemInActiveContest_Returns409(t *testing.T) {
	h := newHandlerWithUnpublish(repoReturning(publishedProblem()), &mockActiveContestCheckerH{inActive: true})

	r := authedRequest(http.MethodPost, "/problems/p/test-problem/unpublish", nil)
	r.SetPathValue("slug", "test-problem")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpublish)).ServeHTTP(w, r)

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

func TestUnpublish_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithUnpublish(&mockProblemRepo{}, &mockActiveContestCheckerH{})

	r := authedRequest(http.MethodPost, "/problems/p/missing/unpublish", nil)
	r.SetPathValue("slug", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Unpublish)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}
