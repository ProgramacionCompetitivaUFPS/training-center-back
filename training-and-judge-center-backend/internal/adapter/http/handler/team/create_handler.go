package team

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appTeam "github.com/training-judge-center/backend/internal/application/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type createTeamRequest struct {
	Name string `json:"name"`
}

type member struct {
	UserID   string `json:"userId"`
	Nickname string `json:"nickname"`
	JoinedAt string `json:"joinedAt"`
}

type createTeamResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	CreatedBy string   `json:"createdBy"`
	CreatedAt string   `json:"createdAt"`
	Members   []member `json:"members"`
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
	currentUser, ok := handler.RequireCurrentUser(w, r)
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

	members := make([]member, len(out.Members))
	for i, m := range out.Members {
		members[i] = member{
			UserID:   m.UserID,
			Nickname: m.Nickname,
			JoinedAt: m.JoinedAt.UTC().Format(time.RFC3339),
		}
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, createTeamResponse{
		ID:        out.ID,
		Name:      out.Name,
		CreatedBy: out.CreatedBy,
		CreatedAt: out.CreatedAt.UTC().Format(time.RFC3339),
		Members:   members,
	})
}
