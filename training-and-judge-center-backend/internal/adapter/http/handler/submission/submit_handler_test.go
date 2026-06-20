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

func TestSubmit_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/problems/p/two-sum/submissions", nil)
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSubmit_MissingMultipartReturns400(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/problems/p/two-sum/submissions", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("slug", "two-sum")
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_InvalidLanguageReturns400(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/two-sum/submissions", "two-sum",
		"cobol85", "cob", "solution.cbl", []byte("IDENTIFICATION DIVISION."))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected %s, got %s", apperror.ErrCodeValidationError, body.Code)
	}
}

func TestSubmit_CompilerMismatchReturns400(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/two-sum/submissions", "two-sum",
		"cpp20", "javac", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeCompilerMismatch {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeCompilerMismatch, body.Code)
	}
}

func TestSubmit_ProblemNotFoundReturns404(t *testing.T) {
	pp := &mockProblemProvider{
		fn: func(_ string) (*appsubmission.ProblemInfo, error) {
			return nil, apperror.NewNotFound(domainsubmission.ErrCodeSubmissionNotFound, "problem not found")
		},
	}
	h := newHandlerWithSubmit(pp, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/ghost/submissions", "ghost",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_UnpublishedProblemReturns400(t *testing.T) {
	pp := &mockProblemProvider{
		fn: func(slug string) (*appsubmission.ProblemInfo, error) {
			return &appsubmission.ProblemInfo{
				ID: "prob-001", Slug: slug, Title: "Draft",
				IsPublished: false, IsPublic: true,
			}, nil
		},
	}
	h := newHandlerWithSubmit(pp, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/draft/submissions", "draft",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeProblemNotPublished {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeProblemNotPublished, body.Code)
	}
}

func TestSubmit_PrivateProblemNonModifierReturns403(t *testing.T) {
	pp := &mockProblemProvider{
		fn: func(slug string) (*appsubmission.ProblemInfo, error) {
			return &appsubmission.ProblemInfo{
				ID: "prob-002", Slug: slug, Title: "Private",
				IsPublished: true, IsPublic: false,
				ModifierIDs: []string{"other-user"},
			}, nil
		},
	}
	h := newHandlerWithSubmit(pp, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/private/submissions", "private",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeProblemNotAccessible {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeProblemNotAccessible, body.Code)
	}
}

func TestSubmit_DuplicateFileReturns409(t *testing.T) {
	repo := &mockSubmissionRepo{
		existsByHashFn: func(_, _, _ string) (bool, error) { return true, nil },
	}
	h := newHandlerWithSubmit(&mockProblemProvider{}, repo)
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/two-sum/submissions", "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeDuplicateSubmission {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeDuplicateSubmission, body.Code)
	}
}

func TestSubmit_ValidCppReturns201(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	fileData := []byte("int main(){}")
	r := multipartRequest("/problems/p/two-sum/submissions", "two-sum",
		"cpp20", "g++", "solution.cpp", fileData)
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body submitResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID == "" {
		t.Error("expected non-empty ID")
	}
	if body.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", body.Status)
	}
	if body.Language != "cpp20" {
		t.Errorf("Language = %q, want cpp20", body.Language)
	}
	if body.Compiler != "g++" {
		t.Errorf("Compiler = %q, want g++", body.Compiler)
	}
	if body.FileHash == "" {
		t.Error("expected non-empty FileHash")
	}
	if body.FileSize != len(fileData) {
		t.Errorf("FileSize = %d, want %d", body.FileSize, len(fileData))
	}
	if body.ProblemSlug != "two-sum" {
		t.Errorf("ProblemSlug = %q, want two-sum", body.ProblemSlug)
	}
	if body.SubmittedAt == "" {
		t.Error("expected non-empty SubmittedAt")
	}
}

func TestSubmit_ValidJavaReturns201(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/two-sum/submissions", "two-sum",
		"java17", "javac", "Solution.java", []byte("class Solution {}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_ValidPythonReturns201(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/two-sum/submissions", "two-sum",
		"python310", "py", "solution.py", []byte("print('hello')"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSubmit_PrivateProblemModifierCanSubmit(t *testing.T) {
	// testAuthorID is the viewer (from wrapWithAuth) and is listed as a modifier.
	pp := &mockProblemProvider{
		fn: func(slug string) (*appsubmission.ProblemInfo, error) {
			return &appsubmission.ProblemInfo{
				ID: "prob-003", Slug: slug, Title: "Private",
				IsPublished: true, IsPublic: false,
				ModifierIDs: []string{testAuthorID},
			}, nil
		},
	}
	h := newHandlerWithSubmit(pp, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := multipartRequest("/problems/p/private/submissions", "private",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
