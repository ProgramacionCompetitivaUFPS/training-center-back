package group

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

func (h *Handler) ChangeRole(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
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
		JoinedAt:      out.JoinedAt.Format(timeutil.RFC3339UTC),
		RoleChangedAt: out.RoleChangedAt.Format(timeutil.RFC3339UTC),
	})
}
