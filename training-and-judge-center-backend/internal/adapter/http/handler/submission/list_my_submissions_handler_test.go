package submission

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

func TestListMySubmissions_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithListMy(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me/submissions", nil)
	wrapWithAuth(http.HandlerFunc(h.ListMySubmissions), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListMySubmissions_ReturnsEmptyList(t *testing.T) {
	h := newHandlerWithListMy(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me/submissions", nil)
	r.Header.Set("Authorization", "Bearer tok")
	wrapWithAuth(http.HandlerFunc(h.ListMySubmissions), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body listSubmissionsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Submissions) != 0 {
		t.Errorf("expected 0 submissions, got %d", len(body.Submissions))
	}
	if body.Pagination.Total != 0 {
		t.Errorf("Pagination.Total = %d, want 0", body.Pagination.Total)
	}
}

func TestListMySubmissions_ReturnsList_WithPagination(t *testing.T) {
	sub := aSubmission(testAuthorID, "PRIVATE", nil)
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{sub}, 1, nil
		},
	}
	h := newHandlerWithListMy(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/users/me/submissions?page=1&limit=20", nil)
	r.Header.Set("Authorization", "Bearer tok")
	wrapWithAuth(http.HandlerFunc(h.ListMySubmissions), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body listSubmissionsResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Submissions) != 1 {
		t.Fatalf("expected 1 submission, got %d", len(body.Submissions))
	}
	if body.Submissions[0].ID != testSubmissionID {
		t.Errorf("ID = %q, want %q", body.Submissions[0].ID, testSubmissionID)
	}
	if body.Pagination.Page != 1 {
		t.Errorf("Pagination.Page = %d, want 1", body.Pagination.Page)
	}
	if body.Pagination.Limit != 20 {
		t.Errorf("Pagination.Limit = %d, want 20", body.Pagination.Limit)
	}
	if body.Pagination.Total != 1 {
		t.Errorf("Pagination.Total = %d, want 1", body.Pagination.Total)
	}
}
