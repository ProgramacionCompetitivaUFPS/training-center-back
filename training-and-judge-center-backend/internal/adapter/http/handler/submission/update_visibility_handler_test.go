package submission

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUpdateVisibility_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUpdateVisibility(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility",
		strings.NewReader(`{"visibility":"PUBLIC"}`))
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateVisibility_NotFound_Returns404(t *testing.T) {
	h := newHandlerWithUpdateVisibility(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	body := mustJSON(t, updateVisibilityRequest{Visibility: "PUBLIC"})
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility", body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateVisibility_InvalidBody_Returns400(t *testing.T) {
	h := newHandlerWithUpdateVisibility(&mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility",
		strings.NewReader("not json"))
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateVisibility_InvalidVisibilityValue_Returns400(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	h := newHandlerWithUpdateVisibility(repo)
	w := httptest.NewRecorder()
	body := mustJSON(t, updateVisibilityRequest{Visibility: "BANANA"})
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility", body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateVisibility_NonAuthor_Returns403(t *testing.T) {
	otherAuthor := "other-user-999"
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(otherAuthor, "PRIVATE", nil), nil
		},
	}
	h := newHandlerWithUpdateVisibility(repo)
	w := httptest.NewRecorder()
	body := mustJSON(t, updateVisibilityRequest{Visibility: "PUBLIC"})
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility", body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var errBody apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&errBody); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if errBody.Code != domainsubmission.ErrCodeAccessDenied {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeAccessDenied, errBody.Code)
	}
}

func TestUpdateVisibility_SetPublic_Returns200(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(testAuthorID, "PRIVATE", nil), nil
		},
	}
	h := newHandlerWithUpdateVisibility(repo)
	w := httptest.NewRecorder()
	body := mustJSON(t, updateVisibilityRequest{Visibility: "PUBLIC"})
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility", body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestUpdateVisibility_SetPrivate_Returns200(t *testing.T) {
	repo := &mockSubmissionRepo{
		findByIDFn: func(_ string) (*domainsubmission.Submission, error) {
			return aSubmission(testAuthorID, "PUBLIC", nil), nil
		},
	}
	h := newHandlerWithUpdateVisibility(repo)
	w := httptest.NewRecorder()
	body := mustJSON(t, updateVisibilityRequest{Visibility: "PRIVATE"})
	r := httptest.NewRequest(http.MethodPatch, "/submissions/"+testSubmissionID+"/visibility", body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", "application/json")
	r.SetPathValue("submissionId", testSubmissionID)
	wrapWithAuth(http.HandlerFunc(h.UpdateVisibility), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func mustJSON(t *testing.T, v any) *bytes.Reader {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}
