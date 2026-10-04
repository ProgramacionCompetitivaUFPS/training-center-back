package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// ValidateText answers 400 to text Postgres cannot store: invalid UTF-8 or NUL in the
// query string, and NUL (the \u0000 escape) in JSON bodies. Invalid bytes inside a JSON
// body are not rejected: encoding/json already turns them into U+FFFD.
// It expects the body to be bounded already (see LimitBody).
func ValidateText() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if field := invalidQueryField(r); field != "" {
				writeInvalidText(w, field)
				return
			}

			if r.Body != nil && r.Body != http.NoBody && !isMultipart(r) {
				body, err := io.ReadAll(r.Body)
				if err != nil {
					writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Could not read request body")
					return
				}
				r.Body = io.NopCloser(bytes.NewReader(body))

				if jsonHasNUL(body) {
					writeInvalidText(w, bodyNULField(body))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func invalidQueryField(r *http.Request) string {
	for key, values := range r.URL.Query() {
		if !validText(key) {
			return "query"
		}
		for _, v := range values {
			if !validText(v) {
				return key
			}
		}
	}
	return ""
}

func validText(s string) bool {
	return utf8.ValidString(s) && !strings.ContainsRune(s, 0)
}

// jsonHasNUL reports whether the JSON text holds a \u0000 escape. A backslash escapes
// the next byte, so an escaped backslash followed by u0000 is plain text, not a NUL.
func jsonHasNUL(b []byte) bool {
	for i := 0; i < len(b); i++ {
		if b[i] != '\\' {
			continue
		}
		if i+5 < len(b) && b[i+1] == 'u' && string(b[i+2:i+6]) == "0000" {
			return true
		}
		i++
	}
	return false
}

// bodyNULField names the top-level key whose value holds the NUL, or "body" if it
// cannot be told (nested path aside, the key is what the client needs to fix).
func bodyNULField(body []byte) string {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(body, &fields); err != nil {
		return "body"
	}
	for key, raw := range fields {
		if jsonHasNUL(raw) {
			return key
		}
	}
	return "body"
}

func writeInvalidText(w http.ResponseWriter, field string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	_ = json.NewEncoder(w).Encode(apperror.NewValidation([]apperror.FieldError{
		{Field: field, Message: "Must be valid UTF-8 text without NUL characters"},
	}))
}
