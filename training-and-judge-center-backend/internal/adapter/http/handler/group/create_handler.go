package group

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type createGroupRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description"`
	JoinMode    string  `json:"joinMode"`
	Visibility  string  `json:"visibility"`
}

type groupResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	JoinPolicy  string  `json:"joinPolicy"`
	Visibility  string  `json:"visibility"`
	IsGlobal    bool    `json:"isGlobal"`
	CreatedBy   string  `json:"createdBy"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
}

// @Summary      Create group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createGroupRequest true "Group data"
// @Success      201 {object} groupResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /groups [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var body createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	out, ucErr := h.createGroup.Execute(r.Context(), appGroup.CreateGroupInput{
		Name:        body.Name,
		Description: body.Description,
		JoinMode:    body.JoinMode,
		Visibility:  body.Visibility,
		CurrentUser: *currentUser,
	})
	if ucErr != nil {
		handler.WriteError(r.Context(), w, ucErr)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, groupResponse{
		ID:          out.ID,
		Name:        out.Name,
		Description: out.Description,
		JoinPolicy:  out.JoinPolicy,
		Visibility:  out.Visibility,
		IsGlobal:    out.IsDefault,
		CreatedBy:   out.CreatedBy,
		CreatedAt:   out.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   out.UpdatedAt.UTC().Format(time.RFC3339),
	})
}
