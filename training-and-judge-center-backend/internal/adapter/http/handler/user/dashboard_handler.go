package user

import (
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type submissionResp struct {
	ID            string `json:"id"`
	ProblemSlug   string `json:"problemSlug"`
	ProblemTitle  string `json:"problemTitle"`
	Verdict       string `json:"verdict"`
	Language      string `json:"language"`
	SubmittedAt   string `json:"submittedAt"`
	ExecutionTime *int   `json:"executionTime"`
	MemoryKb      *int   `json:"memoryKb"`
}

type contestSummary struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	StartTime       string `json:"startTime"`
	DurationMinutes int    `json:"durationMinutes"`
	GroupID         string `json:"groupId"`
	GroupName       string `json:"groupName"`
}

type streakResp struct {
	Current int `json:"current"`
	Maximum int `json:"maximum"`
}

type contestResultResp struct {
	ContestID      string `json:"contestId"`
	ContestName    string `json:"contestName"`
	Position       int    `json:"position"`
	ProblemsSolved int    `json:"problemsSolved"`
	Penalty        int    `json:"penalty"`
}

type dashboardResponse struct {
	RecentSubmissions    []submissionResp    `json:"recentSubmissions"`
	UpcomingContests     []contestSummary    `json:"upcomingContests"`
	ActiveContests       []contestSummary    `json:"activeContests"`
	ProblemsSolved       int                 `json:"problemsSolved"`
	MaterialsCount       int                 `json:"materialsCount"`
	Streak               streakResp          `json:"streak"`
	RecentContestResults []contestResultResp `json:"recentContestResults"`
}

// @Summary      Get my dashboard
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dashboardResponse
// @Failure      401 {object} apperror.AppError
// @Router       /users/me/dashboard [get]
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getDashboard.Execute(r.Context(), appuser.GetDashboardInput{
		CurrentUser: *currentUser,
		Now:         time.Now(),
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildDashboardResponse(out))
}

func buildDashboardResponse(out *appuser.GetDashboardOutput) dashboardResponse {
	submissions := make([]submissionResp, 0, len(out.RecentSubmissions))
	for _, s := range out.RecentSubmissions {
		submissions = append(submissions, submissionResp{
			ID:            s.ID,
			ProblemSlug:   s.ProblemSlug,
			ProblemTitle:  s.ProblemTitle,
			Verdict:       s.Verdict,
			Language:      s.Language,
			SubmittedAt:   s.SubmittedAt.UTC().Format(time.RFC3339),
			ExecutionTime: s.ExecutionTime,
			MemoryKb:      s.MemoryKb,
		})
	}

	upcoming := make([]contestSummary, 0, len(out.UpcomingContests))
	for _, c := range out.UpcomingContests {
		upcoming = append(upcoming, contestSummary{
			ID:              c.ID,
			Name:            c.Name,
			StartTime:       c.StartTime.UTC().Format(time.RFC3339),
			DurationMinutes: c.DurationMinutes,
			GroupID:         c.GroupID,
			GroupName:       c.GroupName,
		})
	}

	active := make([]contestSummary, 0, len(out.ActiveContests))
	for _, c := range out.ActiveContests {
		active = append(active, contestSummary{
			ID:              c.ID,
			Name:            c.Name,
			StartTime:       c.StartTime.UTC().Format(time.RFC3339),
			DurationMinutes: c.DurationMinutes,
			GroupID:         c.GroupID,
			GroupName:       c.GroupName,
		})
	}

	results := make([]contestResultResp, 0, len(out.RecentContestResults))
	for _, cr := range out.RecentContestResults {
		results = append(results, contestResultResp{
			ContestID:      cr.ContestID,
			ContestName:    cr.ContestName,
			Position:       cr.Position,
			ProblemsSolved: cr.ProblemsSolved,
			Penalty:        cr.Penalty,
		})
	}

	return dashboardResponse{
		RecentSubmissions:    submissions,
		UpcomingContests:     upcoming,
		ActiveContests:       active,
		ProblemsSolved:       out.ProblemsSolved,
		MaterialsCount:       out.MaterialsCount,
		Streak:               streakResp{Current: out.CurrentStreak, Maximum: out.MaximumStreak},
		RecentContestResults: results,
	}
}
