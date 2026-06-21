package submission

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetSubmission_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithGetSubmission(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/submissions/"+testSubmissionID, nil)
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.GetSubmission), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetSubmission_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithGetSubmission(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/submissions/"+testSubmissionID, nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.GetSubmission), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSubmission_Forbidden_Returns403(t *testing.T) {
	// Author is "other-user", viewer is testAuthorID, visibility PRIVATE
	otherAuthor := "other-user-999"
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(otherAuthor, "PRIVATE", nil), nil
		},
	}
	h := newHandlerWithGetSubmission(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/submissions/"+testSubmissionID, nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.GetSubmission), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeAccessDenied {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeAccessDenied, body.Code)
	}
}

func TestGetSubmission_AuthorViewsPrivate_Returns200(t *testing.T) {
	// Viewer is testAuthorID; submission author is also testAuthorID → allowed
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	h := newHandlerWithGetSubmission(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/submissions/"+testSubmissionID, nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.GetSubmission), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body getSubmissionResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID != testSubmissionID {
		t.Errorf("ID = %q, want %q", body.ID, testSubmissionID)
	}
	if body.Visibility != "PRIVATE" {
		t.Errorf("Visibility = %q, want PRIVATE", body.Visibility)
	}
	if body.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", body.Status)
	}
	if body.SubmittedAt == "" {
		t.Error("expected non-empty SubmittedAt")
	}
	if body.SourceCode != "int main(){}" {
		t.Errorf("SourceCode = %q", body.SourceCode)
	}
}

func TestGetSubmission_PublicSubmission_Returns200(t *testing.T) {
	otherAuthor := "other-user-999"
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(otherAuthor, "PUBLIC", nil), nil
		},
	}
	h := newHandlerWithGetSubmission(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/submissions/"+testSubmissionID, nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.GetSubmission), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestGetSubmission_ContestSubmission_ContestIncludedInResponse(t *testing.T) {
	cid := "contest-001"
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", &cid), nil
		},
	}
	h := newHandlerWithGetSubmission(repo)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/submissions/"+testSubmissionID, nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.GetSubmission), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body getSubmissionResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Contest == nil {
		t.Fatal("expected contest in response, got nil")
	}
}
