package team

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type createTeamRequest struct {
	Name string `json:"name"`
}

type memberResponse struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	JoinedAt string `json:"joinedAt"`
}

type createTeamResponse struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	CreatedBy string           `json:"createdBy"`
	CreatedAt string           `json:"createdAt"`
	Members   []memberResponse `json:"members"`
}

// @Summary      Create team
// @Tags         teams
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createTeamRequest true "Team data"
// @Success      201 {object} createTeamResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /teams [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	var body createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.createTeam.Execute(r.Context(), appTeam.CreateTeamInput{
		Name:        body.Name,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	members := make([]memberResponse, len(out.Members))
	for i, m := range out.Members {
		members[i] = memberResponse{
			UserID:   m.UserID,
			Nickname: m.Nickname,
			JoinedAt: m.JoinedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, createTeamResponse{
		ID:        out.ID,
		Name:      out.Name,
		CreatedBy: out.CreatedBy,
		CreatedAt: out.CreatedAt.Format("2006-01-02T15:04:05Z"),
		Members:   members,
	})
}
