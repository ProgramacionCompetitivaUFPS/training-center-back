package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
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
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")

	out, err := h.getMyRequest.Execute(r.Context(), appGroup.GetMyRequestInput{
		GroupID:     groupID,
		CurrentUser: *caller,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildJoinRequestResp(out.Request, nil))
}
