package contest

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type getStandingsFilterOptionsResponse struct {
	Countries    []string `json:"countries"`
	Cities       []string `json:"cities"`
	Institutions []string `json:"institutions"`
}

// @Summary      List distinct country/city/institution values across a contest's participants
// @Tags         contests
// @Produce      json
// @Security     BearerAuth
// @Param        groupId   path string true "Group ID"
// @Param        contestId path string true "Contest ID"
// @Success      200 {object} getStandingsFilterOptionsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /groups/{groupId}/contests/{contestId}/standings/filters [get]
func (h *Handler) GetStandingsFilterOptions(w http.ResponseWriter, r *http.Request) {
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

	out, err := h.getStandings.ExecuteFilterOptions(r.Context(), appContest.GetStandingsFilterOptionsInput{
		CurrentUser: *caller,
		GroupID:     groupID,
		ContestID:   contestID,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, getStandingsFilterOptionsResponse{
		Countries:    out.Countries,
		Cities:       out.Cities,
		Institutions: out.Institutions,
	})
}
