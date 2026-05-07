package problem

import (
	"io"
	"log/slog"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

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
// @Router       /problems/import [post]
func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{
			Code:    apperror.ErrCodeUnauthorized,
			Message: "Invalid or missing authentication token",
		})
		return
	}

	maxUploadBytes := int64(h.settings.MaxFileSizeTestCaseMB()) << 20
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		handler.WriteJSON(w, http.StatusRequestEntityTooLarge, apperror.AppError{
			Code:    apperror.ErrCodePayloadTooLarge,
			Message: "File exceeds maximum allowed size",
		})
		return
	}

	slug := r.FormValue("slug")
	if slug == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Missing required form field 'slug'",
		})
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Missing required form field 'file'",
		})
		return
	}
	defer file.Close()

	zipData, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		slog.ErrorContext(r.Context(), "import: failed to read uploaded file", "error", err)
		handler.WriteJSON(w, http.StatusInternalServerError, apperror.NewInternal())
		return
	}
	if int64(len(zipData)) > maxUploadBytes {
		handler.WriteJSON(w, http.StatusRequestEntityTooLarge, apperror.AppError{
			Code:    apperror.ErrCodePayloadTooLarge,
			Message: "File exceeds maximum allowed size",
		})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	result, ucErr := h.importUC.Execute(r.Context(), appProblem.ImportProblemInput{
		Slug:        slug,
		ZipData:     zipData,
		CurrentUser: currentUser,
	})
	if ucErr != nil {
		handler.WriteError(w, ucErr)
		return
	}

	p := result.Problem
	authorDisplay, err := h.userProvider.GetDisplay(r.Context(), p.AuthorID().Value())
	if err != nil {
		slog.DebugContext(r.Context(), "failed to fetch author display", "error", err, "user_id", p.AuthorID().Value())
	}
	handler.WriteJSON(w, http.StatusCreated, buildResponse(p, authorDisplay))
}
