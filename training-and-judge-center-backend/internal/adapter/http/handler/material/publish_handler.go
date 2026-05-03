package material

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
)

// @Summary      Publish material
// @Tags         materials
// @Produce      json
// @Security     BearerAuth
// @Param        groupId    path string true "Group ID"
// @Param        materialId path string true "Material ID"
// @Success      200 {object} materialResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/materials/{materialId}/publish [post]
func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.publishUC.Execute(r.Context(), appMaterial.PublishMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     r.PathValue("groupId"),
		MaterialID:  r.PathValue("materialId"),
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildResponse(out.Material))
}
