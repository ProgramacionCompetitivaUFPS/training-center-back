package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newLimitBodyHandler(limit int64) (http.Handler, *int) {
	read := 0
	h := LimitBody(limit)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		read = len(b)
		w.WriteHeader(http.StatusOK)
	}))
	return h, &read
}

func TestLimitBody_UnderLimit_Passes(t *testing.T) {
	h, read := newLimitBodyHandler(10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK || *read != 10 {
		t.Fatalf("expected 200 with 10 bytes read, got %d with %d", w.Code, *read)
	}
}

func TestLimitBody_OverLimit_Returns413(t *testing.T) {
	h, _ := newLimitBodyHandler(10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("0123456789A"))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "PAYLOAD_TOO_LARGE") {
		t.Errorf("expected PAYLOAD_TOO_LARGE, got %s", w.Body.String())
	}
}

func TestLimitBody_OverLimitWithoutContentLength_Returns413(t *testing.T) {
	h, _ := newLimitBodyHandler(10)
	r := httptest.NewRequest(http.MethodPost, "/", io.MultiReader(strings.NewReader("0123456789A")))
	r.ContentLength = -1
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413, got %d", w.Code)
	}
}

func TestLimitBody_Multipart_IsNotLimited(t *testing.T) {
	h, read := newLimitBodyHandler(10)
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(strings.Repeat("x", 100)))
	r.Header.Set("Content-Type", "multipart/form-data; boundary=abc")
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK || *read != 100 {
		t.Fatalf("expected 200 with 100 bytes read, got %d with %d", w.Code, *read)
	}
}

func TestLimitBody_NoBody_Passes(t *testing.T) {
	h, _ := newLimitBodyHandler(10)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/", nil))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
