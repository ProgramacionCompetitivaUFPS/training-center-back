package middleware

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/domain/shared"
)

// RequireAnyRole allows the request through when the caller's role matches
// any of the given roles. Unlike RequireRole (exact single-role match), this
// is for endpoints shared by more than one privileged role.
func RequireAnyRole(roles ...shared.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cu, ok := GetCurrentUser(r.Context())
			if !ok {
				writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
				return
			}
			for _, role := range roles {
				if cu.Role == role {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeError(w, http.StatusForbidden, "FORBIDDEN", "insufficient permissions")
		})
	}
}
