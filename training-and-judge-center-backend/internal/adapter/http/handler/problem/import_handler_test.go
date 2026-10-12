package problem

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestImport_NonMultipartBody_Returns400(t *testing.T) {
	h := newHandlerWithImport()

	for _, contentType := range []string{"application/json", "application/x-www-form-urlencoded"} {
		t.Run(contentType, func(t *testing.T) {
			r := authedRequest(http.MethodPost, "/problems/import", mustJSON(`{"slug":"x"}`))
			r.Header.Set("Content-Type", contentType)
			w := httptest.NewRecorder()

			wrapAuth(http.HandlerFunc(h.Import)).ServeHTTP(w, r)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
			}
			var body apperror.AppError
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("invalid JSON: %v", err)
			}
			if body.Code != apperror.ErrCodeValidationError {
				t.Errorf("code: got %q, want %q", body.Code, apperror.ErrCodeValidationError)
			}
		})
	}
}

func TestImport_BodyOverTheCap_Returns413(t *testing.T) {
	h := newHandlerWithImport()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	part, _ := mw.CreateFormFile("file", "big.zip")
	_, _ = part.Write([]byte(strings.Repeat("a", 4096)))
	_ = mw.Close()

	r := httptest.NewRequest(http.MethodPost, "/problems/import", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	r.Header.Set("Authorization", "Bearer tok")
	w := httptest.NewRecorder()
	r.Body = http.MaxBytesReader(w, r.Body, 512)

	wrapAuth(http.HandlerFunc(h.Import)).ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d: %s", w.Code, w.Body.String())
	}
}
