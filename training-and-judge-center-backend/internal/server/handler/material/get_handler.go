package material

import (
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Get material
// @Tags         materials
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        materialId path string true "Material ID"
// @Success      200 {object} materialResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/materials/{materialId} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	materialID := r.PathValue("materialId")
	if groupID == "" || materialID == "" {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "groupId and materialId are required"))
		return
	}

	out, err := h.getUC.Execute(r.Context(), appMaterial.GetMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
		MaterialID:  materialID,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildResponse(out.Material))
}
