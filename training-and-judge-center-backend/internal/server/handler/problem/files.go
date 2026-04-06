package problem

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	multipartMemoryPaddingBytes = 50 << 20 // 50 MB extra for form fields and metadata
	uploadContextTimeout        = 2 * time.Minute
)

func (h *Handler) UploadFiles(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
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

	res, ucErr := h.uploadUC.Execute(ctx, appProblem.UploadProblemFilesInput{
		Slug:        slug,
		FileType:    fileType,
		FileName:    fileHeader.Filename,
		FileData:    fileData,
		CurrentUser: *currentUser,
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

func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing token"})
		return
	}

	slug := r.PathValue("slug")
	fileType := r.PathValue("fileType")
	fileName := r.URL.Query().Get("fileName")

	if slug == "" || fileType == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Slug and fileType are required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	_, ucErr := h.deleteFileUC.Execute(ctx, appProblem.DeleteProblemFileInput{
		Slug:        slug,
		FileType:    fileType,
		FileName:    fileName,
		CurrentUser: *currentUser,
	})

	if ucErr != nil {
		var appErr *apperror.AppError
		if errors.As(ucErr, &appErr) {
			handler.WriteError(w, appErr)
		} else {
			slog.Error("Unexpected error in DeleteFile", "error", ucErr, "slug", slug)
			handler.WriteError(w, apperror.NewInternal())
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
