package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/training-judge-center/backend/internal/domain/user"
)

type contextKey string

const claimsKey contextKey = "claims"

func Auth(tokenService user.TokenService, sessionInvalidator user.SessionInvalidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Invalid or missing authentication token"}`))
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			claims, err := tokenService.ValidateToken(tokenString)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Invalid or missing authentication token"}`))
				return
			}

			if sessionInvalidator != nil {
				revoked, err := sessionInvalidator.IsSessionRevoked(r.Context(), claims.UserID, claims.IssuedAt)
				if err != nil || revoked {
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte(`{"error":"UNAUTHORIZED","message":"Invalid or missing authentication token"}`))
					return
				}
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetClaims(ctx context.Context) *user.TokenClaims {
	claims, ok := ctx.Value(claimsKey).(*user.TokenClaims)
	if !ok {
		return nil
	}
	return claims
}

func RequireRole(required user.Role) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil || user.Role(claims.Role) != required {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"FORBIDDEN","message":"Admin privileges required"}`))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
