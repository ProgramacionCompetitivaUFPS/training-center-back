package contest

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Get registration status
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId   path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Success      200 {object} registrationStatusResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/register/status [get]
func (h *Handler) GetRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	if groupID == "" || contestID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	out, err := h.getRegistrationStatus.Execute(r.Context(), appContest.GetRegistrationStatusInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	resp := registrationStatusResponse{Registered: out.Registered}
	if out.RegisteredAt != nil {
		s := out.RegisteredAt.UTC().Format(time.RFC3339)
		resp.RegisteredAt = &s
	}
	handler.WriteJSON(r.Context(), w, http.StatusOK, resp)
}
