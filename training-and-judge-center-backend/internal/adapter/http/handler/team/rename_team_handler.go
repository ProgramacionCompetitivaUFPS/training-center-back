package team

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type renameTeamRequest struct {
	Name string `json:"name"`
}

type renameTeamResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// @Summary      Rename team
// @Description  Any member of the team can change its name. The name follows the same rules as when the team is created: it must be unique (ignoring case) and within the length limit.
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        teamId path string true "Team ID"
// @Param        body body renameTeamRequest true "New name"
// @Success      200 {object} renameTeamResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /teams/{teamId} [patch]
func (h *Handler) RenameTeam(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var body renameTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.renameTeam.Execute(r.Context(), appTeam.RenameTeamInput{
		CurrentUser: *currentUser,
		TeamID:      chi.URLParam(r, "teamId"),
		Name:        body.Name,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, renameTeamResponse{ID: out.ID, Name: out.Name})
}
