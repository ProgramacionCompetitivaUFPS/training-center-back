package group

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type addMemberRequest struct {
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}
	groupID := chi.URLParam(r, "groupId")

	var body addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeValidationError, "Invalid request body"))
		return
	}

	out, err := h.addMember.Execute(r.Context(), appGroup.AddMemberInput{
		GroupID:     groupID,
		Nickname:    body.Nickname,
		Role:        body.Role,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	addedBy := ""
	if ab := out.Member.AddedBy(); ab != nil {
		addedBy = ab.Value()
	}
	handler.WriteJSON(w, http.StatusCreated, addMemberResp{
		GroupID:    out.Member.GroupID(),
		UserID:     out.Member.UserID().Value(),
		Nickname:   out.Nickname,
		Role:       string(out.Member.Role()),
		JoinedAt:   out.Member.JoinedAt().Format(time.RFC3339),
		AddedBy:    addedBy,
		JoinMethod: string(out.Member.JoinMethod()),
	})
}
