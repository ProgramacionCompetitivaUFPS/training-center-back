package middleware

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type contextKey string

const currentUserKey contextKey = "current_user"

func Auth(tokenService user.TokenService, sessionInvalidator user.SessionInvalidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or missing authentication token")
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := tokenService.ValidateToken(tokenString)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or missing authentication token")
				return
			}

			revoked, err := sessionInvalidator.IsAllUserSessionRevoked(r.Context(), claims.UserID, claims.IssuedAt)
			if err != nil {
				slog.ErrorContext(r.Context(), "session revocation check failed", "user_id", claims.UserID, "error", err)
				writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable")
				return
			}
			if revoked {
				writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or missing authentication token")
				return
			}

			if claims.SessionID != "" {
				sessionRevoked, err := sessionInvalidator.IsSessionInvalidated(r.Context(), claims.SessionID)
				if err != nil {
					slog.ErrorContext(r.Context(), "session revocation check failed", "session_id", claims.SessionID, "error", err)
					writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Service temporarily unavailable")
					return
				}
				if sessionRevoked {
					writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or missing authentication token")
					return
				}
			}

			cu := appshared.CurrentUser{ID: claims.UserID, Role: claims.Role}
			ctx := context.WithValue(r.Context(), currentUserKey, cu)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetCurrentUser(ctx context.Context) (appshared.CurrentUser, bool) {
	cu, ok := ctx.Value(currentUserKey).(appshared.CurrentUser)
	return cu, ok
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q,"message":%q}`, code, msg)
}
