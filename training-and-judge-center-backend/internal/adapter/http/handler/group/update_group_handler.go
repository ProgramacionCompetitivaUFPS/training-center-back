package group

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Update group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string        true "Group ID"
// @Param        body    body updateGroupReq true "Fields to update (all optional)"
// @Success      200 {object} updateGroupResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /groups/{groupId} [patch]
func (h *Handler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	var req updateGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "invalid JSON"},
		}))
		return
	}

	out, err := h.updateGroup.Execute(r.Context(), appGroup.UpdateGroupInput{
		GroupID:     r.PathValue("groupId"),
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
		JoinPolicy:  req.JoinPolicy,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildUpdateGroupResp(out))
}

func buildUpdateGroupResp(out *appGroup.UpdateGroupOutput) updateGroupResp {
	resp := updateGroupResp{
		ID:          out.ID,
		Name:        out.Name,
		Description: out.Description,
		Visibility:  out.Visibility,
		JoinPolicy:  out.JoinPolicy,
		CreatedBy:   out.CreatedBy,
		CreatedAt:   out.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   out.UpdatedAt.UTC().Format(time.RFC3339),
		MembersCount: out.MembersCount,
	}
	if out.RequestsAutoApproved > 0 || out.RequestsAutoRejected > 0 {
		resp.PolicyChangeEffects = &policyChangeEffectsResp{
			RequestsAutoApproved: out.RequestsAutoApproved,
			RequestsAutoRejected: out.RequestsAutoRejected,
		}
	}
	return resp
}
