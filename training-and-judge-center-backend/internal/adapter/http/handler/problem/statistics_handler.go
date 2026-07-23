package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type statisticsResponse struct {
	TotalSubmissions         int                   `json:"totalSubmissions"`
	UniqueUsers              uniqueUsersResp       `json:"uniqueUsers"`
	AcceptanceRateByLanguage []languageStatResp    `json:"acceptanceRateByLanguage"`
	VerdictDistribution      []verdictStatResp     `json:"verdictDistribution"`
}

type uniqueUsersResp struct {
	Attempted int `json:"attempted"`
	Solved    int `json:"solved"`
}

type languageStatResp struct {
	Language       string `json:"language"`
	UsersAccepted  int    `json:"usersAccepted"`
	UsersAttempted int    `json:"usersAttempted"`
}

type verdictStatResp struct {
	Verdict string `json:"verdict"`
	Count   int    `json:"count"`
}

type noSubmissionsResponse struct {
	Message          string `json:"message"`
	TotalSubmissions int    `json:"totalSubmissions"`
}

// @Summary      Get problem statistics
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} statisticsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/statistics [get]
func (h *Handler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetCurrentUser(r.Context()); !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	out, err := h.getProblemStatistics.Execute(r.Context(), appProblem.GetProblemStatisticsInput{Slug: slug})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	if !out.HasSubmissions {
		handler.WriteJSON(r.Context(), w, http.StatusOK, noSubmissionsResponse{
			Message:          "No submissions yet for this problem",
			TotalSubmissions: 0,
		})
		return
	}

	langs := make([]languageStatResp, len(out.ByLanguage))
	for i, ls := range out.ByLanguage {
		langs[i] = languageStatResp{
			Language:       ls.Language,
			UsersAccepted:  ls.UsersAccepted,
			UsersAttempted: ls.UsersAttempted,
		}
	}

	verdicts := make([]verdictStatResp, len(out.ByVerdict))
	for i, vs := range out.ByVerdict {
		verdicts[i] = verdictStatResp{Verdict: vs.Verdict, Count: vs.Count}
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, statisticsResponse{
		TotalSubmissions: out.TotalSubmissions,
		UniqueUsers: uniqueUsersResp{
			Attempted: out.UniqueAttempted,
			Solved:    out.UniqueSolved,
		},
		AcceptanceRateByLanguage: langs,
		VerdictDistribution:      verdicts,
	})
}
