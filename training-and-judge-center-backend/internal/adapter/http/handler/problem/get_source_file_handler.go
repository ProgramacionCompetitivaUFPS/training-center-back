package problem

import (
	"net/http"
	"strconv"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type sourceFileResponse struct {
	FileType  string `json:"fileType"`
	Filename  string `json:"filename"`
	Language  string `json:"language"`
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
	SizeBytes int    `json:"sizeBytes"`
}

// @Summary      Read a solution, checker or validator
// @Description  The source of one of the problem's judging files (256 KiB at most; the response says if it was cut). Only the author, modifiers and Admin.
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        fileType path string true "solution, checker or validator"
// @Param        fileName query string false "Which solution (required for fileType=solution)"
// @Param        full query bool false "Return the whole file instead of the first 256 KiB"
// @Success      200 {object} sourceFileResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/files/{fileType} [get]
func (h *Handler) GetSourceFile(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	full, _ := strconv.ParseBool(r.URL.Query().Get("full"))
	out, err := h.getSourceFile.Execute(r.Context(), appProblem.GetSourceFileInput{
		Slug:        r.PathValue("slug"),
		FileType:    r.PathValue("fileType"),
		FileName:    r.URL.Query().Get("fileName"),
		Full:        full,
		CurrentUser: *cu,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	// solutions, checker and validator are not for contestants: never to be cached
	w.Header().Set("Cache-Control", "no-store")
	handler.WriteJSON(r.Context(), w, http.StatusOK, sourceFileResponse{
		FileType:  out.FileType,
		Filename:  out.Filename,
		Language:  out.Language,
		Content:   out.Content,
		Truncated: out.Truncated,
		SizeBytes: out.SizeBytes,
	})
}
