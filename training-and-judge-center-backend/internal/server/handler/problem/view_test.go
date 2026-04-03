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

func TestGetProblem_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/problems/p/my-problem", nil)
	req.SetPathValue("slug", "my-problem")
	w := httptest.NewRecorder()

	h.GetProblem(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGetProblem_NotFound_Returns404(t *testing.T) {
	uc := &mockGetProblemUC{
		fn: func(_ context.Context, _ appProblem.GetProblemInput) (*appProblem.GetProblemOutput, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	h := newTestHandler(nil, nil, nil, uc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/problems/p/missing", nil)
	req.SetPathValue("slug", "missing")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.GetProblem(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestGetProblem_HappyPath_Returns200WithSlug(t *testing.T) {
	p := testProblem("my-problem")
	uc := &mockGetProblemUC{
		fn: func(_ context.Context, _ appProblem.GetProblemInput) (*appProblem.GetProblemOutput, error) {
			return &appProblem.GetProblemOutput{
				Problem:   p,
				Author:    appProblem.ModifierDisplay{Nickname: "author", Name: "Author"},
				Modifiers: []appProblem.ModifierDisplay{{Nickname: "author", Name: "Author"}},
				Files:     nil,
			}, nil
		},
	}
	h := newTestHandler(nil, nil, nil, uc, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/problems/p/my-problem", nil)
	req.SetPathValue("slug", "my-problem")
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.GetProblem(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "my-problem") {
		t.Errorf("expected slug in response body, got: %s", w.Body.String())
	}
}

func TestListProblems_NoUser_Returns401(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/problems?page=1&limit=20", nil)
	w := httptest.NewRecorder()

	h.ListProblems(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestListProblems_InvalidPage_Returns400(t *testing.T) {
	h := newTestHandler(nil, nil, nil, nil, nil, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/problems?page=abc", nil)
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.ListProblems(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestListProblems_HappyPath_Returns200WithPagination(t *testing.T) {
	p := testProblem("my-problem")
	uc := &mockListProblemsUC{
		fn: func(_ context.Context, _ appProblem.ListProblemsInput) (*appProblem.ListProblemsOutput, error) {
			return &appProblem.ListProblemsOutput{
				Problems: []appProblem.ProblemSummary{
					{Problem: p, Author: appProblem.ModifierDisplay{Nickname: "author", Name: "Author"}},
				},
				TotalCount: 1,
				TotalPages: 1,
				Page:       1,
				Limit:      20,
			}, nil
		},
	}
	h := newTestHandler(nil, nil, nil, nil, uc, nil, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/problems?page=1&limit=20", nil)
	req = withUser(req, coachUser)
	w := httptest.NewRecorder()

	h.ListProblems(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "pagination") {
		t.Errorf("expected pagination in response, got: %s", body)
	}
	if !strings.Contains(body, "my-problem") {
		t.Errorf("expected slug in response, got: %s", body)
	}
}
