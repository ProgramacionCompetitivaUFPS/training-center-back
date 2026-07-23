package submission

import (
	"net/http"
	"strconv"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// ListProblemSubmissions handles GET /problems/p/{slug}/submissions
//
// @Summary      List submissions for a problem
// @Tags         submissions
// @Security     BearerAuth
// @Produce      json
// @Param        slug     path   string  true   "Problem slug"
// @Param        page     query  int     false  "Page number (default 1)"
// @Param        limit    query  int     false  "Items per page (default 20, max 100)"
// @Param        verdict  query  string  false  "Filter by verdict"
// @Param        language query  string  false  "Filter by language"
// @Param        mine     query  bool    false  "Only my submissions (regardless of visibility)"
// @Success      200 {object} listSubmissionsResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Failure      500 {object} apperror.AppError
// @Router       /problems/p/{slug}/submissions [get]
func (h *Handler) ListProblemSubmissions(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	slug := r.PathValue("slug")
	mine, _ := strconv.ParseBool(r.URL.Query().Get("mine"))
	page, limit := parsePagination(r)

	in := appsubmission.ListProblemSubmissionsInput{
		CurrentUser: *cu,
		ProblemSlug: slug,
		Status:      queryParamPtr(r, "verdict"),
		Language:    queryParamPtr(r, "language"),
		OnlyMine:    mine,
		Page:        page,
		Limit:       limit,
	}

	out, err := h.listProblemSubmissions.Execute(r.Context(), in)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toListResponse(out.Submissions, out.Total, out.Page, out.Limit))
}
