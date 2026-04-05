package problem

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUnpublish_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems/my-problem/unpublish", nil)
	req.SetPathValue("slug", "my-problem")
	w := httptest.NewRecorder()

	h.Unpublish(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestUnpublish_NotFound_Returns404(t *testing.T) {
	uc := &mockUnpublishUC{
		fn: func(_ context.Context, _ appProblem.UnpublishProblemInput) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	h := newTestHandler(nil, nil, nil, nil, nil, nil, uc, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems/missing/unpublish", nil)
	req.SetPathValue("slug", "missing")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Unpublish(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestUnpublish_HappyPath_Returns200(t *testing.T) {
	now := time.Now()
	uc := &mockUnpublishUC{
		fn: func(_ context.Context, _ appProblem.UnpublishProblemInput) (*domainProblem.Problem, error) {
			return domainProblem.RestoreProblem(
				"test-id", "my-problem", "My Problem",
				nil, nil, nil, []string{}, "DRAFT", "PRIVATE",
				domainProblem.RestoreUserID("author-id"),
				nil, nil, nil, nil, nil, nil, nil,
				now, now,
			), nil
		},
	}
	h := newTestHandler(nil, nil, nil, nil, nil, nil, uc, nil)

	req := httptest.NewRequest(http.MethodPost, "/problems/my-problem/unpublish", nil)
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.Unpublish(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "DRAFT") {
		t.Errorf("expected DRAFT status in response, got: %s", w.Body.String())
	}
}

func TestChangeAccessibility_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/problems/my-problem/accessibility", strings.NewReader(`{"accessibility":"PUBLIC"}`))
	req.SetPathValue("slug", "my-problem")
	w := httptest.NewRecorder()

	h.ChangeAccessibility(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestChangeAccessibility_InvalidBody_Returns400(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodPatch, "/problems/my-problem/accessibility", strings.NewReader(`not-json`))
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.ChangeAccessibility(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestChangeAccessibility_HappyPath_Returns200(t *testing.T) {
	now := time.Now()
	uc := &mockChangeAccessibilityUC{
		fn: func(_ context.Context, _ appProblem.ChangeAccessibilityInput) (*domainProblem.Problem, error) {
			return domainProblem.RestoreProblem(
				"test-id", "my-problem", "My Problem",
				nil, nil, nil, []string{}, "DRAFT", "PUBLIC",
				domainProblem.RestoreUserID("author-id"),
				nil, nil, nil, nil, nil, nil, nil,
				now, now,
			), nil
		},
	}
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, uc)

	req := httptest.NewRequest(http.MethodPatch, "/problems/my-problem/accessibility", strings.NewReader(`{"accessibility":"PUBLIC"}`))
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.ChangeAccessibility(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "PUBLIC") {
		t.Errorf("expected PUBLIC accessibility in response, got: %s", w.Body.String())
	}
}
