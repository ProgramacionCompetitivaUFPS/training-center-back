package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func serveValidateText(r *http.Request) (*httptest.ResponseRecorder, string) {
	var seen string
	h := ValidateText()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = "reached"
		w.WriteHeader(http.StatusOK)
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w, seen
}

func TestValidateText_Query(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantField string
	}{
		{"stray byte", "search=%F1", "search"},
		{"truncated sequence", "search=%C3", "search"},
		{"NUL alone", "search=%00", "search"},
		{"NUL in the middle", "author=a%00b", "author"},
		{"NUL at the start", "tags=%00ab", "tags"},
		{"NUL at the end", "searchTerm=ab%00", "searchTerm"},
		{"invalid key", "%F1=x", "query"},
		{"valid accents", "search=%C3%B1", ""},
		{"valid CJK and emoji", "search=%E6%97%A5%F0%9F%98%80", ""},
		{"literal wildcards and quotes", "search=100%25_%5C%27%22", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w, seen := serveValidateText(httptest.NewRequest(http.MethodGet, "/problems?"+tt.query, nil))

			if tt.wantField == "" {
				if w.Code != http.StatusOK || seen != "reached" {
					t.Fatalf("expected the request to pass, got %d", w.Code)
				}
				return
			}
			assertInvalidText(t, w, tt.wantField)
		})
	}
}

func TestValidateText_JSONBody(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantField string
	}{
		{"NUL alone", `{"title":"\u0000"}`, "title"},
		{"NUL at the start", `{"title":"\u0000ab"}`, "title"},
		{"NUL in the middle", `{"statement":"a\u0000b"}`, "statement"},
		{"NUL at the end", `{"name":"ab\u0000"}`, "name"},
		{"NUL after an escaped quote", `{"name":"\"\u0000"}`, "name"},
		{"escaped backslash before u0000 is text", `{"statement":"\\u0000"}`, ""},
		{"NUL after escaped backslash", `{"statement":"\\\u0000"}`, "statement"},
		{"other escapes", `{"title":"añ\n\t\u0001"}`, ""},
		{"accents and emoji", `{"title":"Camión 😀"}`, ""},
		{"stray byte inside JSON is not rejected", "{\"title\":\"a\xf1b\"}", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/problems", strings.NewReader(tt.body))
			r.Header.Set("Content-Type", "application/json")

			w, seen := serveValidateText(r)

			if tt.wantField == "" {
				if w.Code != http.StatusOK || seen != "reached" {
					t.Fatalf("expected the request to pass, got %d", w.Code)
				}
				return
			}
			assertInvalidText(t, w, tt.wantField)
		})
	}
}

func TestValidateText_BodyStillReadableByHandler(t *testing.T) {
	var got string
	h := ValidateText()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sb strings.Builder
		buf := make([]byte, 64)
		for {
			n, err := r.Body.Read(buf)
			sb.Write(buf[:n])
			if err != nil {
				break
			}
		}
		got = sb.String()
	}))
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"a":"b"}`))
	r.Header.Set("Content-Type", "application/json")

	h.ServeHTTP(httptest.NewRecorder(), r)

	if got != `{"a":"b"}` {
		t.Fatalf("handler did not receive the body, got %q", got)
	}
}

func TestValidateText_MultipartBodyIsNotInspected(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`\u0000`))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=abc")

	w, seen := serveValidateText(r)

	if w.Code != http.StatusOK || seen != "reached" {
		t.Fatalf("expected the request to pass, got %d", w.Code)
	}
}

func assertInvalidText(t *testing.T, w *httptest.ResponseRecorder, wantField string) {
	t.Helper()
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Code != apperror.ErrCodeValidationError || len(resp.Details) != 1 || resp.Details[0].Field != wantField {
		t.Errorf("expected VALIDATION_ERROR on field %q, got %+v", wantField, resp)
	}
}
