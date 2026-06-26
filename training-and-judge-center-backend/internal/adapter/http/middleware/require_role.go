package middleware

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

func RequireRole(required shared.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cu, ok := GetCurrentUser(r.Context())
			if !ok || cu.Role != required {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "Admin privileges required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
