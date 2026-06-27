package group

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type updateRequestBody struct {
	Status string `json:"status"`
}

// @Summary      Approve or reject a join request
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        requestId path string true "Request ID"
// @Param        body body updateRequestBody true "New status (APPROVED or REJECTED)"
// @Success      200 {object} joinRequestResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/requests/{requestId} [patch]
func (h *Handler) UpdateJoinRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")
	requestID := chi.URLParam(r, "requestId")

	var body updateRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	switch body.Status {
	case domainGroup.JoinRequestStatusApproved.String():
		out, err := h.approveRequest.Execute(r.Context(), appGroup.ApproveRequestInput{
			GroupID:     groupID,
			RequestID:   requestID,
			CurrentUser: *caller,
		})
		if err != nil {
			handler.WriteError(r.Context(), w, err)
			return
		}
		handler.WriteJSON(r.Context(), w, http.StatusOK, buildJoinRequestResp(out.Request, nil))

	case domainGroup.JoinRequestStatusRejected.String():
		out, err := h.rejectRequest.Execute(r.Context(), appGroup.RejectRequestInput{
			GroupID:     groupID,
			RequestID:   requestID,
			CurrentUser: *caller,
		})
		if err != nil {
			handler.WriteError(r.Context(), w, err)
			return
		}
		handler.WriteJSON(r.Context(), w, http.StatusOK, buildJoinRequestResp(out.Request, nil))

	default:
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "status must be APPROVED or REJECTED",
			Details: []apperror.FieldError{{Field: "status", Message: "must be APPROVED or REJECTED"}},
		})
	}
}
