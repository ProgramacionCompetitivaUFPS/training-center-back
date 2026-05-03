package group

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type requestJoinBody struct {
	Message *string `json:"message"`
}

// @Summary      Request to join a group
// @Tags         groups
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        body body requestJoinBody false "Optional message"
// @Success      201 {object} joinRequestResp
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /groups/{groupId}/requests [post]
func (h *Handler) RequestJoin(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")

	var body requestJoinBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	out, err := h.requestJoin.Execute(r.Context(), appGroup.RequestJoinInput{
		GroupID:     groupID,
		Message:     body.Message,
		CurrentUser: *caller,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusCreated, buildJoinRequestResp(out.Request, nil))
}

func buildJoinRequestResp(req *domainGroup.JoinRequest, display *appGroup.UserDisplay) joinRequestResp {
	resp := joinRequestResp{
		ID:        req.ID(),
		GroupID:   req.GroupID(),
		Status:    req.Status().String(),
		Message:   req.Message(),
		CreatedAt: req.CreatedAt().Format("2006-01-02T15:04:05Z"),
	}
	if display != nil {
		resp.Requester = &requesterResp{
			UserID:   req.RequesterUserID().Value(),
			Nickname: display.Nickname,
			Name:     display.Name,
			Email:    display.Email,
		}
	}
	return resp
}
