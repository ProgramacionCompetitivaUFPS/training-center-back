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

type submitContestResponse struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	SubmittedAt  string `json:"submittedAt"`
	ProblemSlug  string `json:"problemSlug"`
	ProblemTitle string `json:"problemTitle"`
	ContestID    string `json:"contestId"`
	ContestName  string `json:"contestName"`
	Language     string `json:"language"`
	Compiler     string `json:"compiler"`
	FileSize     int    `json:"fileSize"`
	FileHash     string `json:"fileHash"`
}

// SubmitContest handles POST /groups/{groupId}/contests/{contestId}/problems/{problemSlug}/submissions
func (h *Handler) SubmitContest(w http.ResponseWriter, r *http.Request) {
	// Capture submittedAt IMMEDIATELY — before any validation or processing
	submittedAt := time.Now().UTC()

	cu, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	contestID := r.PathValue("contestId")
	problemSlug := r.PathValue("problemSlug")
	if groupID == "" || contestID == "" || problemSlug == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "missing path parameter"))
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
		slog.ErrorContext(r.Context(), "submission: failed to read uploaded file for contest", "error", err)
		handler.WriteError(r.Context(), w, apperror.NewInternal())
		return
	}

	out, ucErr := h.submitContestSolution.Execute(r.Context(), appSubmission.SubmitContestSolutionInput{
		CurrentUser: *cu,
		GroupID:     groupID,
		ContestID:   contestID,
		ProblemSlug: problemSlug,
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

	handler.WriteJSON(r.Context(), w, http.StatusCreated, submitContestResponse{
		ID:           out.ID,
		Status:       out.Status,
		SubmittedAt:  out.SubmittedAt.UTC().Format(time.RFC3339),
		ProblemSlug:  out.ProblemSlug,
		ProblemTitle: out.ProblemTitle,
		ContestID:    out.ContestID,
		ContestName:  out.ContestName,
		Language:     out.Language,
		Compiler:     out.Compiler,
		FileSize:     out.FileSize,
		FileHash:     out.FileHash,
	})
}
