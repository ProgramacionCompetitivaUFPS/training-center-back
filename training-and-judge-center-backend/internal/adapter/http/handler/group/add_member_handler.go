package group

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Add member to group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body addMemberReq true "Member data"
// @Success      201 {object} addMemberResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /groups/{groupId}/members [post]
func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
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
		JoinedAt:   out.JoinedAt.UTC().Format(time.RFC3339),
		AddedBy:    out.AddedBy,
		JoinMethod: out.JoinMethod,
	})
}
