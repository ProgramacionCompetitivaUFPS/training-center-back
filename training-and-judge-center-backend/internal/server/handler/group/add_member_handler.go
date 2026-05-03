package group

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type addMemberRequest struct {
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
}

// @Summary      Add member to group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body addMemberRequest true "Member data"
// @Success      201 {object} addMemberResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /groups/{groupId}/members [post]
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}
	groupID := r.PathValue("groupId")

	var body addMemberRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeValidationError, "Invalid request body"))
		return
	}
	if strings.TrimSpace(body.Nickname) == "" || body.Role == "" {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeValidationError, "nickname and role are required"))
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

	handler.WriteJSON(w, http.StatusCreated, addMemberResp{
		GroupID:    out.Member.GroupID(),
		UserID:     out.Member.UserID().Value(),
		Nickname:   out.Nickname,
		Role:       string(out.Member.Role()),
		JoinedAt:   out.Member.JoinedAt().Format(time.RFC3339),
		AddedBy:    userIDPtrToStringPtr(out.Member.AddedBy()),
		JoinMethod: string(out.Member.JoinMethod()),
	})
}
