package material

import (
	"encoding/json"
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Create material
// @Tags         materials
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body createMaterialRequest true "Material data"
// @Success      201 {object} materialResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /groups/{groupId}/materials [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	if groupID == "" {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "groupId is required"))
		return
	}

	var body createMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeValidationError, "Invalid request body"))
		return
	}

	out, err := h.createUC.Execute(r.Context(), appMaterial.CreateMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
		Title:       body.Title,
		Content:     body.Content,
		Tags:        body.Tags,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusCreated, buildResponse(out.Material))
}
