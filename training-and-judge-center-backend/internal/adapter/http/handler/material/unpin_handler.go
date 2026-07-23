package material

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
)

// @Summary      Unpin material
// @Tags         materials
// @Produce      json
// @Security     BearerAuth
// @Param        groupId    path string true "Group ID"
// @Param        materialId path string true "Material ID"
// @Success      200 {object} materialResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/materials/{materialId}/unpin [post]
func (h *Handler) Unpin(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.unpinMaterial.Execute(r.Context(), appMaterial.UnpinMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     r.PathValue("groupId"),
		MaterialID:  r.PathValue("materialId"),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildResponse(out.Material))
}
