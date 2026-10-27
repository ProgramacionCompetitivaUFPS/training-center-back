package middleware

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
)

// LimitBody caps the request body at maxBytes and answers 413 beyond it.
// Multipart requests pass untouched: their handlers enforce their own, larger limits.
func LimitBody(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil || r.Body == http.NoBody || isMultipart(r) {
				next.ServeHTTP(w, r)
				return
			}

			if r.ContentLength > maxBytes {
				writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body is too large")
				return
			}

			// Buffered so an oversized chunked body is also answered 413, not as a decode error in the handler.
			body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes))
			if err != nil {
				var tooLarge *http.MaxBytesError
				if errors.As(err, &tooLarge) {
					writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body is too large")
					return
				}
				writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Could not read request body")
				return
			}

			r.Body = io.NopCloser(bytes.NewReader(body))
			next.ServeHTTP(w, r)
		})
	}
}

func isMultipart(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && strings.HasPrefix(mediaType, "multipart/")
}
