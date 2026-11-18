package problem

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const zipTooLargeCode = "ZIP_TOO_LARGE"

// @Summary      Import problem from ZIP
// @Tags         problems
// @Accept       multipart/form-data
// @Produce      json
// @Security     BearerAuth
// @Param        slug formData string true "Problem slug"
// @Param        file formData file true "ZIP archive"
// @Success      201 {object} getProblemResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      413 {object} apperror.AppError
// @Router       /problems/import [post]
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{
			Code:    apperror.ErrCodeUnauthorized,
			Message: "Invalid or missing authentication token",
		})
		return
	}

	release, err := h.uploads.acquire(r.Context())
	if err != nil {
		handler.WriteError(r.Context(), w, uploadFailure(w, err))
		return
	}
	defer release()

	maxUploadBytes := int64(h.settings.MaxZipUploadSizeMB()) << 20
	upload, err := receiveUpload(w, r, maxUploadBytes)
	if err != nil {
		if errors.Is(err, errFileTooLarge) {
			handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Code: zipTooLargeCode, Message: "ZIP file exceeds the maximum upload size"},
			}))
			return
		}
		slog.WarnContext(r.Context(), "import: upload rejected", "error", err)
		handler.WriteError(r.Context(), w, uploadFailure(w, err))
		return
	}
	defer upload.Close()

	slug := upload.fields["slug"]
	if slug == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Missing required form field 'slug'",
		})
		return
	}
	if upload.file == nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Missing required form field 'file'",
		})
		return
	}

	currentUser := shared.CurrentUser{ID: cu.ID, Role: cu.Role}

	out, ucErr := h.importProblem.Execute(r.Context(), appProblem.ImportProblemInput{
		Slug:        slug,
		Zip:         upload.file,
		CurrentUser: currentUser,
	})
	if ucErr != nil {
		handler.WriteError(r.Context(), w, ucErr)
		return
	}

	p := out.Problem
	authorDisplay, err := h.userProvider.GetDisplay(r.Context(), p.AuthorID)
	if err != nil {
		slog.DebugContext(r.Context(), "failed to fetch author display", "error", err, "user_id", p.AuthorID)
	}
	handler.WriteJSON(r.Context(), w, http.StatusCreated, buildResponse(p, authorDisplay))
}
