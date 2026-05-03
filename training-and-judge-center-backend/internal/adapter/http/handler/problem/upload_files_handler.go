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
	"github.com/training-judge-center/backend/internal/domain/shared"
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
// @Success      200 {object} getProblemResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /problems/p/{slug}/files [post]
func (h *Handler) UploadFiles(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing token"})
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Problem slug is required"})
		return
	}

	maxTestCaseBytes := int64(h.settings.GetMaxFileSizeTestCaseMB()) << 20
	maxDefaultBytes := int64(h.settings.GetMaxFileSizeDefaultMB()) << 20

	// Parse form with dynamic memory logic (TestCase limit + 50MB padding)
	maxMemory := maxTestCaseBytes + multipartMemoryPaddingBytes
	err := r.ParseMultipartForm(maxMemory)
	if err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Failed to parse multipart form"})
		return
	}

	fileType := r.FormValue("fileType")
	if fileType == "" {
		fileType = r.FormValue("type")
	}
	file, fileHeader, err := r.FormFile("file")
	if err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "File missing in form-data"})
		return
	}
	defer file.Close()

	limitBytes := maxDefaultBytes
	if strings.EqualFold(fileType, appProblem.FileTypeTestCases) {
		limitBytes = maxTestCaseBytes
	}

	if fileHeader.Size > limitBytes {
		handler.WriteJSON(w, http.StatusRequestEntityTooLarge, apperror.AppError{Code: apperror.ErrCodePayloadTooLarge, Message: "File size exceeds allowed limit"})
		return
	}

	fileData, err := io.ReadAll(io.LimitReader(file, limitBytes+1))
	if err != nil {
		slog.Error("Failed to read uploaded file", "error", err, "slug", slug)
		handler.WriteJSON(w, http.StatusInternalServerError, apperror.NewInternal())
		return
	}
	if int64(len(fileData)) > limitBytes {
		handler.WriteJSON(w, http.StatusRequestEntityTooLarge, apperror.AppError{Code: apperror.ErrCodePayloadTooLarge, Message: "File size exceeds allowed limit"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), uploadContextTimeout)
	defer cancel()

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	res, ucErr := h.uploadUC.Execute(ctx, appProblem.UploadProblemFilesInput{
		Slug:        slug,
		FileType:    fileType,
		FileName:    fileHeader.Filename,
		FileData:    fileData,
		CurrentUser: currentUser,
	})

	if ucErr != nil {
		handler.WriteError(w, ucErr)
		return
	}

	authorDisplay, _ := h.userProvider.GetDisplay(r.Context(), res.Problem.AuthorID().Value())
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": res.Message,
		"problem": buildResponse(res.Problem, authorDisplay),
	})
}
