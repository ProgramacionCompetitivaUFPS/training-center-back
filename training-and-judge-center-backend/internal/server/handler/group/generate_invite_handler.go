package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
)

type generateInviteResponse struct {
	Token string `json:"token"`
}

// @Summary      Generate invite token
// @Tags         groups
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Success      201 {object} generateInviteResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/invitations [post]
func (h *Handler) GenerateInvite(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.generateInvite.Execute(r.Context(), appGroup.GenerateInviteInput{
		GroupID:     r.PathValue("groupId"),
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusCreated, generateInviteResponse{Token: out.Token})
}
