package handler

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func RequireCurrentUser(w http.ResponseWriter, r *http.Request) (*appshared.CurrentUser, bool) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	return &cu, true
}
