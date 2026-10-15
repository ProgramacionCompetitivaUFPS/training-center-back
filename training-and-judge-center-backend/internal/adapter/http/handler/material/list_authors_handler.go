package material

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
)

type listAuthorsResponse struct {
	Authors []authorResp `json:"authors"`
}

// ListAuthors handles GET /groups/{groupId}/materials/authors
//
// @Summary      List authors of the group's most recent materials
// @Description  Returns a sample (not an exhaustive registry) meant for filter suggestions —
// @Description  an author outside the sample is still a valid nickname for the `author` filter on List.
// @Tags         materials
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      200 {object} listAuthorsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/materials/authors [get]
func (h *Handler) ListAuthors(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")

	out, err := h.listMaterialAuthors.Execute(r.Context(), appMaterial.ListMaterialAuthorsInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	authors := make([]authorResp, len(out.Authors))
	for i, a := range out.Authors {
		authors[i] = authorResp{Nickname: a.Nickname, Name: a.Name}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listAuthorsResponse{Authors: authors})
}
