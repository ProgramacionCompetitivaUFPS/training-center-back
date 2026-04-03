package middleware

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/user"
)

// WithCurrentUser injects a CurrentUser into the context. For use in tests only.
func WithCurrentUser(ctx context.Context, u user.CurrentUser) context.Context {
	return context.WithValue(ctx, currentUserKey, u)
}
