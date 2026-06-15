package group

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	var req addMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.addMember.Execute(r.Context(), appGroup.AddMemberInput{
		GroupID:     r.PathValue("groupId"),
		Nickname:    req.Nickname,
		Role:        req.Role,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, addMemberResp{
		GroupID:    out.GroupID,
		UserID:     out.UserID,
		Nickname:   out.Nickname,
		Role:       out.Role,
		JoinedAt:   out.JoinedAt.Format(timeutil.RFC3339UTC),
		AddedBy:    out.AddedBy,
		JoinMethod: out.JoinMethod,
	})
}
