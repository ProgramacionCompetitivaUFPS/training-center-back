package problem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestDeleteProblem_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/problems/my-problem", strings.NewReader(`{"confirmSlug":"my-problem"}`))
	req.SetPathValue("slug", "my-problem")
	w := httptest.NewRecorder()

	h.DeleteProblem(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDeleteProblem_InvalidBody_Returns400(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/problems/my-problem", strings.NewReader(`not-json`))
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, adminUser)
	w := httptest.NewRecorder()

	h.DeleteProblem(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDeleteProblem_NotFound_Returns404(t *testing.T) {
	uc := &mockDeleteProblemUC{
		fn: func(_ context.Context, _ appProblem.DeleteProblemInput) error {
			return apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	h := newTestHandler(nil, nil, nil, nil, nil, uc, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/problems/missing", strings.NewReader(`{"confirmSlug":"missing"}`))
	req.SetPathValue("slug", "missing")
	req = withUser(req, adminUser)
	w := httptest.NewRecorder()

	h.DeleteProblem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestDeleteProblem_HappyPath_Returns204(t *testing.T) {
	uc := &mockDeleteProblemUC{
		fn: func(_ context.Context, _ appProblem.DeleteProblemInput) error {
			return nil
		},
	}
	h := newTestHandler(nil, nil, nil, nil, nil, uc, nil, nil)

	req := httptest.NewRequest(http.MethodDelete, "/problems/my-problem", strings.NewReader(`{"confirmSlug":"my-problem"}`))
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, adminUser)
	w := httptest.NewRecorder()

	h.DeleteProblem(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204, got %d", w.Code)
	}
}
