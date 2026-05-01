package group

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type updateRequestBody struct {
	Status string `json:"status"`
}

func (h *Handler) UpdateJoinRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")
	requestID := chi.URLParam(r, "requestId")

	var body updateRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	switch domainGroup.JoinRequestStatus(body.Status) {
	case domainGroup.JoinRequestStatusApproved:
		out, err := h.approveRequest.Execute(r.Context(), appGroup.ApproveRequestInput{
			GroupID:     groupID,
			RequestID:   requestID,
			CurrentUser: *caller,
		})
		if err != nil {
			handler.WriteError(w, err)
			return
		}
		handler.WriteJSON(w, http.StatusOK, buildJoinRequestResp(out.Request, nil))

	case domainGroup.JoinRequestStatusRejected:
		out, err := h.rejectRequest.Execute(r.Context(), appGroup.RejectRequestInput{
			GroupID:     groupID,
			RequestID:   requestID,
			CurrentUser: *caller,
		})
		if err != nil {
			handler.WriteError(w, err)
			return
		}
		handler.WriteJSON(w, http.StatusOK, buildJoinRequestResp(out.Request, nil))

	default:
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "status must be APPROVED or REJECTED",
			Details: []apperror.FieldError{{Field: "status", Message: "must be APPROVED or REJECTED"}},
		})
	}
}
