package group

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

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
		CreatedAt:   out.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   out.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z"),
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
