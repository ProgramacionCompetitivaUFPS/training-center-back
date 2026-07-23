package group

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Change member role
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId  path string        true "Group ID"
// @Param        nickname path string        true "Member nickname"
// @Param        body     body changeRoleReq true "Role data"
// @Success      200 {object} changeRoleResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/members/{nickname} [patch]
func (h *Handler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	var req changeRoleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.changeRole.Execute(r.Context(), appGroup.ChangeRoleInput{
		GroupID:     r.PathValue("groupId"),
		Nickname:    r.PathValue("nickname"),
		Role:        req.Role,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, changeRoleResp{
		GroupID:       out.GroupID,
		UserID:        out.UserID,
		Nickname:      out.Nickname,
		Role:          out.Role,
		JoinedAt:      out.JoinedAt.UTC().Format(time.RFC3339),
		RoleChangedAt: out.RoleChangedAt.UTC().Format(time.RFC3339),
	})
}
