package group

import (
	"encoding/json"
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
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
	IsDefault   bool    `json:"isDefault"`
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
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{
			Code:    apperror.ErrCodeUnauthorized,
			Message: "Invalid or missing authentication token",
		})
		return
	}

	var body createGroupRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	result, ucErr := h.createGroup.Execute(r.Context(), appGroup.CreateGroupInput{
		Name:        body.Name,
		Description: body.Description,
		JoinMode:    body.JoinMode,
		Visibility:  body.Visibility,
		CurrentUser: currentUser,
	})
	if ucErr != nil {
		handler.WriteError(w, ucErr)
		return
	}

	g := result.Group
	handler.WriteJSON(w, http.StatusCreated, groupResponse{
		ID:          g.ID(),
		Name:        g.Name().Value(),
		Description: g.Description(),
		JoinPolicy:  g.JoinPolicy().String(),
		Visibility:  g.Visibility().String(),
		IsDefault:   g.IsDefault(),
		CreatedBy:   g.CreatedBy().Value(),
		CreatedAt:   g.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   g.UpdatedAt().Format("2006-01-02T15:04:05Z"),
	})
}
