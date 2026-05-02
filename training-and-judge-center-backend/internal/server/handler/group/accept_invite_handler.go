package group

import (
	"encoding/json"
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

type acceptInviteRequest struct {
	Token string `json:"token"`
}

func (h *Handler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	var body acceptInviteRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "invalid request body"))
		return
	}

	out, err := h.acceptInvite.Execute(r.Context(), appGroup.AcceptInviteInput{
		Token:       body.Token,
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
