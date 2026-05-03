package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
)

// @Summary      Cancel my join request for a group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/requests/me [delete]
func (h *Handler) CancelMyRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")

	if err := h.cancelMyRequest.Execute(r.Context(), appGroup.CancelMyRequestInput{
		GroupID:     groupID,
		CurrentUser: *caller,
	}); err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
