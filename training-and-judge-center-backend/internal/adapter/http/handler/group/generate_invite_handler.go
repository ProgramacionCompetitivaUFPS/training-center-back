package group

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
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
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.generateInvite.Execute(r.Context(), appGroup.GenerateInviteInput{
		GroupID:     r.PathValue("groupId"),
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, generateInviteResponse{Token: out.Token})
}
