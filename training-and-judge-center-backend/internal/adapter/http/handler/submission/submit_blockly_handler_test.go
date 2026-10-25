package submission

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// blocklyRequest is a python310 submission that also carries a workspaceXml part.
func blocklyRequest(workspace string) *http.Request {
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	_ = w.WriteField("language", "python310")
	_ = w.WriteField("compiler", "py")
	part, _ := w.CreateFormFile("file", "solution.py")
	_, _ = part.Write([]byte("print(1)\n"))
	ws, _ := w.CreateFormFile("workspaceXml", "workspace.xml")
	_, _ = ws.Write([]byte(workspace))
	w.Close()

	r := httptest.NewRequest(http.MethodPost, "/problems/p/two-sum/submissions", body)
	r.Header.Set("Authorization", "Bearer tok")
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.SetPathValue("slug", "two-sum")
	return r
}

func TestSubmit_BlocklyWorkspaceReturns201(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := blocklyRequest(`<xml xmlns="https://developers.google.com/blockly/xml"></xml>`)
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body submitResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !body.IsBlockly {
		t.Error("expected isBlocklySubmission = true")
	}
	if body.Language != "python310" {
		t.Errorf("Language = %q, want python310", body.Language)
	}
}

func TestSubmit_InvalidBlocklyWorkspaceReturns400(t *testing.T) {
	h := newHandlerWithSubmit(&mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := blocklyRequest(`<svg onload="alert(1)"></svg>`)
	wrapWithAuth(http.HandlerFunc(h.Submit), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != appsubmission.ErrCodeInvalidBlocklyWorkspace {
		t.Errorf("expected %s, got %s", appsubmission.ErrCodeInvalidBlocklyWorkspace, body.Code)
	}
}
