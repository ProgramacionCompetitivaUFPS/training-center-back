package user

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type ranking struct {
	Position   *int `json:"position"`
	TotalUsers int  `json:"totalUsers"`
}

type topicStat struct {
	Tag    string `json:"tag"`
	Solved int    `json:"solved"`
}

type statsResponse struct {
	ProblemsSolved       int         `json:"problemsSolved"`
	TotalSubmissions     int         `json:"totalSubmissions"`
	AcceptedSubmissions  int         `json:"acceptedSubmissions"`
	ContestsParticipated int         `json:"contestsParticipated"`
	Ranking              ranking     `json:"ranking"`
	TopicStats           []topicStat `json:"topicStats"`
}

// @Summary      Get my profile statistics
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} statsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /users/me/stats [get]
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getProfileStats.Execute(r.Context(), appuser.GetProfileStatsInput{
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildStatsResponse(out))
}

func buildStatsResponse(out *appuser.GetProfileStatsOutput) statsResponse {
	topicStats := make([]topicStat, 0, len(out.TopicStats))
	for _, t := range out.TopicStats {
		topicStats = append(topicStats, topicStat{Tag: t.Tag, Solved: t.Solved})
	}

	return statsResponse{
		ProblemsSolved:       out.ProblemsSolved,
		TotalSubmissions:     out.TotalSubmissions,
		AcceptedSubmissions:  out.AcceptedSubmissions,
		ContestsParticipated: out.ContestsParticipated,
		Ranking:              ranking{Position: out.RankingPosition, TotalUsers: out.RankingTotalUsers},
		TopicStats:           topicStats,
	}
}
