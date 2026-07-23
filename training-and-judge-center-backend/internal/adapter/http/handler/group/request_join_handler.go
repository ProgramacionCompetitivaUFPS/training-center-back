package group

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
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
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")

	var body requestJoinBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
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
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, buildJoinRequestResp(out.Request, nil))
}

func buildJoinRequestResp(req appGroup.JoinRequestDTO, display *appGroup.UserDisplay) joinRequestResp {
	resp := joinRequestResp{
		ID:        req.ID,
		GroupID:   req.GroupID,
		Status:    req.Status,
		Message:   req.Message,
		CreatedAt: req.CreatedAt.UTC().Format(time.RFC3339),
	}
	if display != nil {
		resp.Requester = &requesterResp{
			UserID:   req.RequesterUserID,
			Nickname: display.Nickname,
			Name:     display.Name,
			Email:    display.Email,
		}
	}
	return resp
}
