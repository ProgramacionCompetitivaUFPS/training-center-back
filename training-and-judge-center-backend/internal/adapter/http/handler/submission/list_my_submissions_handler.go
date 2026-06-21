package submission

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// ListMySubmissions handles GET /users/me/submissions
//
// @Summary      List my submissions
// @Tags         submissions
// @Security     BearerAuth
// @Produce      json
// @Param        page        query  int     false  "Page number (default 1)"
// @Param        limit       query  int     false  "Items per page (default 20, max 100)"
// @Param        verdict     query  string  false  "Filter by status"
// @Param        language    query  string  false  "Filter by language"
// @Param        problemSlug query  string  false  "Filter by problem slug"
// @Success      200 {object} listSubmissionsResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      500 {object} apperror.AppError
// @Router       /users/me/submissions [get]
func (h *Handler) ListMySubmissions(w http.ResponseWriter, r *http.Request) {
	cu, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	page, limit := parsePagination(r)
	in := appsubmission.ListMySubmissionsInput{
		CurrentUser: *cu,
		Status:      queryParamPtr(r, "verdict"),
		Language:    queryParamPtr(r, "language"),
		ProblemSlug: queryParamPtr(r, "problemSlug"),
		Page:        page,
		Limit:       limit,
	}

	out, err := h.listMySubmissions.Execute(r.Context(), in)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, toListResponse(out.Submissions, out.Total, out.Page, out.Limit))
}
