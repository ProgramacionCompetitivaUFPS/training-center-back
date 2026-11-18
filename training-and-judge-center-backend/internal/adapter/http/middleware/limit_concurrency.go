package middleware

import (
	"net/http"
	"strconv"
	"time"
)

// LimitConcurrency lets at most limit requests run the wrapped handlers at once. One that
// cannot get a slot within wait is answered 503 with Retry-After, instead of piling up in memory.
// Build it once and share it: the budget is per instance, not per route.
func LimitConcurrency(limit int, wait time.Duration) func(http.Handler) http.Handler {
	slots := make(chan struct{}, limit)
	retryAfter := strconv.Itoa(max(1, int(wait/time.Second)))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			timer := time.NewTimer(wait)
			defer timer.Stop()

			select {
			case slots <- struct{}{}:
				defer func() { <-slots }()
				next.ServeHTTP(w, r)
			case <-timer.C:
				w.Header().Set("Retry-After", retryAfter)
				writeError(w, http.StatusServiceUnavailable, "SERVER_BUSY", "Too many requests in progress, try again shortly")
			case <-r.Context().Done():
			}
		})
	}
}
