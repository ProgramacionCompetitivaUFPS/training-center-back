package submission

import (
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appSubmission "github.com/training-judge-center/backend/internal/application/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	maxMultipartMemory = 10 << 20 // 10 MB multipart parse budget; controls memory vs disk split
	maxReadBytes       = 2 << 20  // 2 MB read cap; use case enforces the actual 1 MB limit
)

type submitResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	SubmittedAt  string `json:"submittedAt"`
	ProblemSlug  string `json:"problemSlug"`
	ProblemTitle string `json:"problemTitle"`
	Language     string `json:"language"`
	Compiler     string `json:"compiler"`
	FileSize     int    `json:"fileSize"`
	FileHash     string `json:"fileHash"`
}

// @Summary      Submit a solution
// @Tags         submissions
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        slug     path     string true "Problem slug"
// @Param        language formData string true "Programming language"
// @Param        compiler formData string true "Compiler identifier"
// @Param        file     formData file   true "Solution source file"
// @Success      201 {object} submitResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Failure      429 {object} apperror.AppError
// @Router       /problems/p/{slug}/submissions [post]
func (h *Handler) Submit(w http.ResponseWriter, r *http.Request) {
	// Capture submittedAt IMMEDIATELY — before any validation or processing
	submittedAt := time.Now().UTC()

	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "problem slug is required"))
		return
	}

	if err := r.ParseMultipartForm(maxMultipartMemory); err != nil {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "failed to parse multipart form"))
		return
	}

	language := r.FormValue("language")
	compiler := r.FormValue("compiler")

	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "file field is required in multipart form"))
		return
	}
	defer file.Close()

	fileData, err := io.ReadAll(io.LimitReader(file, maxReadBytes))
	if err != nil {
		slog.ErrorContext(r.Context(), "submission: failed to read uploaded file", "error", err)
		handler.WriteError(r.Context(), w, apperror.NewInternal())
		return
	}

	out, ucErr := h.submitSolution.Execute(r.Context(), appSubmission.SubmitSolutionInput{
		CurrentUser: *cu,
		ProblemSlug: slug,
		Language:    language,
		Compiler:    compiler,
		FileName:    fileHeader.Filename,
		FileData:    fileData,
		SubmittedAt: submittedAt,
	})
	if ucErr != nil {
		handler.WriteError(r.Context(), w, ucErr)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusCreated, submitResponse{
		ID:           out.ID,
		Status:       out.Status,
		SubmittedAt:  out.SubmittedAt.UTC().Format(time.RFC3339),
		ProblemSlug:  out.ProblemSlug,
		ProblemTitle: out.ProblemTitle,
		Language:     out.Language,
		Compiler:     out.Compiler,
		FileSize:     out.FileSize,
		FileHash:     out.FileHash,
	})
}
