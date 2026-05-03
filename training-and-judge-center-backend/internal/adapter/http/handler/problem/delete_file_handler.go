package problem

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Delete problem file
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        fileType path string true "File type"
// @Param        fileName query string false "File name (for solutions)"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /problems/p/{slug}/files/{fileType} [delete]
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
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

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	_, ucErr := h.deleteFileUC.Execute(ctx, appProblem.DeleteProblemFileInput{
		Slug:        slug,
		FileType:    fileType,
		FileName:    fileName,
		CurrentUser: currentUser,
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
