package contest

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Reveal the final results of a contest
// @Description  Unfreezes the scoreboard for everyone once a contest with a freeze has finished. Idempotent.
// @Tags         contests
// @Security     BearerAuth
// @Param        groupId    path  string true "Group ID"
// @Param        contestId  path  string true "Contest ID"
// @Success      204
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/reveal-results [post]
func (h *Handler) RevealResults(w http.ResponseWriter, r *http.Request) {
	caller, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	if groupID == "" || contestID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
		return
	}

	err := h.revealContestResults.Execute(r.Context(), appContest.RevealContestResultsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
		Now:         time.Now().UTC(),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
