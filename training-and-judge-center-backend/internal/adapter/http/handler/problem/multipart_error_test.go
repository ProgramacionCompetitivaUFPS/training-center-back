package problem

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithSettingsOnly() *Handler {
	s, err := domainProblem.NewPlatformSettings(
		10000, 512,
		map[string]domainProblem.LanguageLimit{},
		map[string]struct{}{"cpp": {}},
		map[string]string{".cpp": "cpp"},
		map[string]struct{}{"dp": {}},
		4, 10, 50, 100, 10,
	)
	if err != nil {
		panic(err)
	}
	return &Handler{settings: s}
}

func TestUploadRoutes_NonMultipartBody_Returns400(t *testing.T) {
	h := newHandlerWithSettingsOnly()
	routes := map[string]http.HandlerFunc{
		"/problems/import":       h.Import,
		"/problems/p/test/files": h.UploadFiles,
	}
	contentTypes := []string{"application/json", "application/x-www-form-urlencoded"}

	for target, handlerFn := range routes {
		for _, ct := range contentTypes {
			t.Run(target+" "+ct, func(t *testing.T) {
				r := authedRequest(http.MethodPost, target, mustJSON(`{"slug":"x"}`))
				r.Header.Set("Content-Type", ct)
				r.SetPathValue("slug", "test")
				w := httptest.NewRecorder()

				wrapAuth(handlerFn).ServeHTTP(w, r)

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
}

func TestUploadRoutes_BodyOverTheCap_Returns413(t *testing.T) {
	h := newHandlerWithSettingsOnly()
	routes := map[string]http.HandlerFunc{
		"/problems/import":       h.Import,
		"/problems/p/test/files": h.UploadFiles,
	}

	for target, handlerFn := range routes {
		t.Run(target, func(t *testing.T) {
			var buf bytes.Buffer
			mw := multipart.NewWriter(&buf)
			part, _ := mw.CreateFormFile("file", "big.zip")
			_, _ = part.Write([]byte(strings.Repeat("a", 4096)))
			_ = mw.Close()

			r := httptest.NewRequest(http.MethodPost, target, &buf)
			r.Header.Set("Content-Type", mw.FormDataContentType())
			r.SetPathValue("slug", "test")
			w := httptest.NewRecorder()
			r.Body = http.MaxBytesReader(w, r.Body, 512)
			r.Header.Set("Authorization", "Bearer tok")

			wrapAuth(handlerFn).ServeHTTP(w, r)

			if w.Code != http.StatusRequestEntityTooLarge {
				t.Fatalf("expected 413, got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}
