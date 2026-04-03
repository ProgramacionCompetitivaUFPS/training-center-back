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

func TestCreate_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems", strings.NewReader(`{}`))
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCreate_InvalidBody_Returns400(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems", strings.NewReader(`not-json`))
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_UCReturnsValidationError_Returns400(t *testing.T) {
	uc := &mockCreateUC{
		fn: func(_ context.Context, _ appProblem.CreateProblemInput) (*appProblem.CreateProblemResult, error) {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "slug", Message: "slug already exists"},
			})
		},
	}
	h := newTestHandler(uc, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems", strings.NewReader(`{"slug":"dup","title":"T"}`))
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCreate_HappyPath_Returns201WithSlug(t *testing.T) {
	p := testProblem("my-problem")
	uc := &mockCreateUC{
		fn: func(_ context.Context, _ appProblem.CreateProblemInput) (*appProblem.CreateProblemResult, error) {
			return &appProblem.CreateProblemResult{Problem: p}, nil
		},
	}
	h := newTestHandler(uc, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems", strings.NewReader(`{"slug":"my-problem","title":"My Problem"}`))
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Create(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected 201, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "my-problem") {
		t.Errorf("expected slug in response body, got: %s", w.Body.String())
	}
}

func TestUpdate_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/problems/my-problem", strings.NewReader(`{}`))
	req.SetPathValue("slug", "my-problem")
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUpdate_InvalidBody_Returns400(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/problems/my-problem", strings.NewReader(`not-json`))
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestUpdate_UCReturnsNotFound_Returns404(t *testing.T) {
	uc := &mockUpdateUC{
		fn: func(_ context.Context, _ appProblem.UpdateProblemInput) (*appProblem.UpdateProblemResult, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	h := newTestHandler(nil, uc, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/problems/missing", strings.NewReader(`{"title":"New"}`))
	req.SetPathValue("slug", "missing")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUpdate_HappyPath_Returns200WithSlug(t *testing.T) {
	p := testProblem("my-problem")
	uc := &mockUpdateUC{
		fn: func(_ context.Context, _ appProblem.UpdateProblemInput) (*appProblem.UpdateProblemResult, error) {
			return &appProblem.UpdateProblemResult{Problem: p}, nil
		},
	}
	h := newTestHandler(nil, uc, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/problems/my-problem", strings.NewReader(`{"title":"Updated"}`))
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Update(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "my-problem") {
		t.Errorf("expected slug in response body, got: %s", w.Body.String())
	}
}
