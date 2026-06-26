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

type contestSummaryResp struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	StartTime       string `json:"startTime"`
	DurationMinutes int    `json:"durationMinutes"`
}

type materialResp struct {
	ID             string `json:"id"`
	Title          string `json:"title"`
	GroupID        string `json:"groupId"`
	GroupName      string `json:"groupName"`
	PublishedAt    string `json:"publishedAt"`
	AuthorNickname string `json:"authorNickname"`
}

type streakResp struct {
	Current int `json:"current"`
	Maximum int `json:"maximum"`
}

type rankingResp struct {
	Position   *int `json:"position"`
	TotalUsers int  `json:"totalUsers"`
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
	UpcomingContests     []contestSummaryResp `json:"upcomingContests"`
	ActiveContests       []contestSummaryResp `json:"activeContests"`
	ProblemsSolved       int                  `json:"problemsSolved"`
	RecentMaterials      []materialResp       `json:"recentMaterials"`
	Streak               streakResp           `json:"streak"`
	Ranking              rankingResp          `json:"ranking"`
	RecentContestResults []contestResultResp  `json:"recentContestResults"`
}

// @Summary      Get my dashboard
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} dashboardResponse
// @Failure      401 {object} apperror.AppError
// @Router       /users/me/dashboard [get]
func (h *Handler) GetDashboard(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getDashboard.Execute(r.Context(), appuser.GetDashboardInput{
		CurrentUser: *currentUser,
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

	upcoming := make([]contestSummaryResp, 0, len(out.UpcomingContests))
	for _, c := range out.UpcomingContests {
		upcoming = append(upcoming, contestSummaryResp{
			ID:              c.ID,
			Name:            c.Name,
			StartTime:       c.StartTime.UTC().Format(time.RFC3339),
			DurationMinutes: c.DurationMinutes,
		})
	}

	active := make([]contestSummaryResp, 0, len(out.ActiveContests))
	for _, c := range out.ActiveContests {
		active = append(active, contestSummaryResp{
			ID:              c.ID,
			Name:            c.Name,
			StartTime:       c.StartTime.UTC().Format(time.RFC3339),
			DurationMinutes: c.DurationMinutes,
		})
	}

	materials := make([]materialResp, 0, len(out.RecentMaterials))
	for _, m := range out.RecentMaterials {
		materials = append(materials, materialResp{
			ID:             m.ID,
			Title:          m.Title,
			GroupID:        m.GroupID,
			GroupName:      m.GroupName,
			PublishedAt:    m.PublishedAt.UTC().Format(time.RFC3339),
			AuthorNickname: m.AuthorNickname,
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
		RecentMaterials:      materials,
		Streak:               streakResp{Current: out.CurrentStreak, Maximum: out.MaximumStreak},
		Ranking:              rankingResp{Position: out.RankingPosition, TotalUsers: out.RankingTotalUsers},
		RecentContestResults: results,
	}
}
