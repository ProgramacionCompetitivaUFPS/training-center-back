package submission

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	submitSolution *appSubmission.SubmitSolutionUseCase
}

func NewHandler(submitSolution *appSubmission.SubmitSolutionUseCase) *Handler {
	return &Handler{submitSolution: submitSolution}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*appshared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "invalid or missing authentication token"))
		return nil, false
	}
	u := appshared.CurrentUser{ID: claims.UserID, Role: claims.Role}
	return &u, true
}
