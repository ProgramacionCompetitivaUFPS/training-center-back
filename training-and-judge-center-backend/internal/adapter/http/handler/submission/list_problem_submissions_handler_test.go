package submission

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestListProblemSubmissions_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithListProblem(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/problems/p/two-sum/submissions", nil)
	r.SetPathValue("slug", "two-sum")
	wrapWithAuth(http.HandlerFunc(h.ListProblemSubmissions), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestListProblemSubmissions_ProblemNotFound_Returns404(t *testing.T) {
	pp := &mockProblemProvider{
		fn: func(_ string) (*appsubmission.ProblemInfo, error) {
			return nil, apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "problem not found")
		},
	}
	uc := appsubmission.NewListProblemSubmissionsUseCase(&mockSubmissionRepo{}, pp, &mockUserDisplayProvider{})
	h := NewHandler(nil, nil, nil, nil, uc)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/problems/p/ghost/submissions", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("slug", "ghost")
	wrapWithAuth(http.HandlerFunc(h.ListProblemSubmissions), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestListProblemSubmissions_ReturnsPublicSubmissions(t *testing.T) {
	sub := aSubmission(testAuthorID, "PUBLIC", nil)
	repo := &mockSubmissionRepo{
		listFn: func(_ domainsubmission.ListFilters) ([]*domainsubmission.Submission, int, error) {
			return []*domainsubmission.Submission{sub}, 1, nil
		},
	}
	h := newHandlerWithListProblem(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/problems/p/two-sum/submissions", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("slug", "two-sum")
	wrapWithAuth(http.HandlerFunc(h.ListProblemSubmissions), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
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
	if body.Submissions[0].Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", body.Submissions[0].Status)
	}
	if body.Submissions[0].Contest != nil {
		t.Error("contest should be nil for problem submissions")
	}
}
