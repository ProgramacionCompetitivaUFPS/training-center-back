package problem

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	multipartMemoryPaddingBytes = 50 << 20 // 50 MB extra for form fields and metadata
	uploadContextTimeout        = 2 * time.Minute
)

// @Summary      Upload problem files
// @Tags         problems
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        fileType formData string true "File type (testCases, solution, checker, validator)"
// @Param        file formData file true "File to upload"
// @Param        confirmSlug formData string false "The problem slug; required to replace testCases or checker of a published problem that is in an active contest"
// @Success      200 {object} getProblemResponse
// @Success      202 {object} judgingUpdateResponse "Live update of a published problem accepted; it takes effect once validated"
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      413 {object} apperror.AppError
// @Router       /problems/p/{slug}/files [post]
func (h *Handler) UploadFiles(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing token"})
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Problem slug is required"})
		return
	}

	maxTestCaseBytes := int64(h.settings.MaxFileSizeTestCaseMB()) << 20
	maxDefaultBytes := int64(h.settings.MaxFileSizeDefaultMB()) << 20

	// Parse form with dynamic memory logic (TestCase limit + 50MB padding)
	maxMemory := maxTestCaseBytes + multipartMemoryPaddingBytes
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		handler.WriteError(r.Context(), w, multipartParseError(err))
		return
	}

	fileType := r.FormValue("fileType")
	if fileType == "" {
		fileType = r.FormValue("type")
	}
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "File missing in form-data"})
		return
	}
	defer file.Close()

	limitBytes := maxDefaultBytes
	if strings.EqualFold(fileType, appProblem.FileTypeTestCases) {
		limitBytes = maxTestCaseBytes
	}

	if fileHeader.Size > limitBytes {
		handler.WriteError(r.Context(), w, apperror.NewPayloadTooLarge("File size exceeds allowed limit"))
		return
	}

	fileData, err := io.ReadAll(io.LimitReader(file, limitBytes+1))
	if err != nil {
		slog.ErrorContext(r.Context(), "Failed to read uploaded file", "error", err, "slug", slug)
		handler.WriteJSON(r.Context(), w, http.StatusInternalServerError, apperror.NewInternal())
		return
	}
	if int64(len(fileData)) > limitBytes {
		handler.WriteError(r.Context(), w, apperror.NewPayloadTooLarge("File size exceeds allowed limit"))
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), uploadContextTimeout)
	defer cancel()

	currentUser := shared.CurrentUser{ID: cu.ID, Role: cu.Role}

	out, ucErr := h.uploadProblemFiles.Execute(ctx, appProblem.UploadProblemFilesInput{
		Slug:        slug,
		FileType:    fileType,
		FileName:    fileHeader.Filename,
		FileData:    fileData,
		ConfirmSlug: r.FormValue("confirmSlug"),
		CurrentUser: currentUser,
	})

	if ucErr != nil {
		handler.WriteError(r.Context(), w, ucErr)
		return
	}

	authorDisplay, _ := h.userProvider.GetDisplay(r.Context(), out.Problem.AuthorID)
	if out.ValidationID != "" {
		handler.WriteJSON(r.Context(), w, http.StatusAccepted, judgingUpdateResponse{
			Message:          out.Message,
			ValidationID:     out.ValidationID,
			AffectedContests: toAffectedContests(out.AffectedContests),
			Problem:          buildResponse(out.Problem, authorDisplay),
		})
		return
	}
	handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]interface{}{
		"message": out.Message,
		"problem": buildResponse(out.Problem, authorDisplay),
	})
}
