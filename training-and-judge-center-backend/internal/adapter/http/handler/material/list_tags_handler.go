package material

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
)

type listTagsResponse struct {
	Tags []string `json:"tags"`
}

// ListTags handles GET /groups/{groupId}/materials/tags
//
// @Summary      List distinct material tags used in a group
// @Tags         materials
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      200 {object} listTagsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/materials/tags [get]
func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")

	out, err := h.listMaterialTags.Execute(r.Context(), appMaterial.ListMaterialTagsInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listTagsResponse{Tags: out.Tags})
}
