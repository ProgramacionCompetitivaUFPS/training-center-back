package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

// @Summary      Export a problem as an ICPC package
// @Description  problem.yaml, statement, test cases, solutions, checker and validator, in the same layout the import reads. Only the author, modifiers and Admin.
// @Tags         problems
// @Produce      application/zip
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {file} binary
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/export [get]
func (h *Handler) Export(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.exportProblem.Execute(r.Context(), appProblem.ExportProblemInput{
		Slug:        r.PathValue("slug"),
		CurrentUser: *cu,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}
	writeZipDownload(w, out.FileName, out.Data)
}
