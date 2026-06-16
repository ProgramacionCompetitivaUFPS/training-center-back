package team

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	createTeam *appTeam.CreateTeamUseCase
}

func NewHandler(createTeam *appTeam.CreateTeamUseCase) *Handler {
	return &Handler{createTeam: createTeam}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}
	return &u, true
}
