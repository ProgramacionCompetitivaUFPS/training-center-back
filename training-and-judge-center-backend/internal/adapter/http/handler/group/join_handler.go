package group

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

type joinGroupResponse struct {
	Role     string `json:"role"`
	JoinedAt string `json:"joinedAt"`
}

// @Summary      Join group
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      201 {object} joinGroupResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/join [post]
func (h *Handler) Join(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.joinGroup.Execute(r.Context(), appGroup.JoinGroupInput{
		GroupID:     r.PathValue("groupId"),
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	m := out.Member
	handler.WriteJSON(w, http.StatusCreated, joinGroupResponse{
		Role:     string(m.Role()),
		JoinedAt: m.JoinedAt().Format(timeutil.RFC3339UTC),
	})
}
