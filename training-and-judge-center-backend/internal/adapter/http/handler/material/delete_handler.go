package material

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
)

// @Summary      Delete material
// @Tags         materials
// @Security     BearerAuth
// @Param        groupId    path string true "Group ID"
// @Param        materialId path string true "Material ID"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/materials/{materialId} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	err := h.deleteMaterial.Execute(r.Context(), appMaterial.DeleteMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     r.PathValue("groupId"),
		MaterialID:  r.PathValue("materialId"),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
