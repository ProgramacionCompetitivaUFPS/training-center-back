package group

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

// @Summary      Remove member from group
// @Tags         groups
// @Security     BearerAuth
// @Param        groupId  path string true "Group ID"
// @Param        nickname path string true "Member nickname"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/members/{nickname} [delete]
func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	err := h.removeMember.Execute(r.Context(), appGroup.RemoveMemberInput{
		GroupID:     r.PathValue("groupId"),
		Nickname:    r.PathValue("nickname"),
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
