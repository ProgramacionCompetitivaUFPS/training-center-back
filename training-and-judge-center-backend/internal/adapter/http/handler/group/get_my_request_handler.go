package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      Get my join request for a group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      200 {object} joinRequestResp
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/requests/me [get]
func (h *Handler) GetMyRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")

	out, err := h.getMyRequest.Execute(r.Context(), appGroup.GetMyRequestInput{
		GroupID:     groupID,
		CurrentUser: *caller,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildJoinRequestResp(out.Request, nil))
}
