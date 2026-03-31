package problem

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{
			Code:    apperror.ErrCodeUnauthorized,
			Message: "Invalid or missing authentication token",
		})
		return
	}

	var body createProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	langOverrides := make([]appProblem.LanguageOverrideInput, 0, len(body.LangOverrides))
	for _, lo := range body.LangOverrides {
		langOverrides = append(langOverrides, appProblem.LanguageOverrideInput{
			Language:    lo.Language,
			TimeLimit:   lo.TimeLimit,
			MemoryLimit: lo.MemoryLimit,
		})
	}

	result, ucErr := h.createUC.Execute(r.Context(), appProblem.CreateProblemInput{
		Slug:          body.Slug,
		Title:         body.Title,
		Statement:     body.Statement,
		TimeLimit:     body.TimeLimit,
		MemoryLimit:   body.MemoryLimit,
		LangOverrides: langOverrides,
		Tags:          body.Tags,
		CurrentUser:   *currentUser,
	})

	if ucErr != nil {
		handler.WriteError(w, ucErr)
		return
	}

	p := result.Problem
	authorDisplay, _ := h.userProvider.GetDisplay(r.Context(), p.AuthorID)

	handler.WriteJSON(w, http.StatusCreated, buildResponse(p, authorDisplay))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	var body updateProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeValidationError, Message: "Invalid request body"})
		return
	}

	var langOverrides []appProblem.LanguageOverrideInput
	if body.LangOverrides != nil {
		langOverrides = make([]appProblem.LanguageOverrideInput, 0, len(body.LangOverrides))
		for _, lo := range body.LangOverrides {
			langOverrides = append(langOverrides, appProblem.LanguageOverrideInput{
				Language:    lo.Language,
				TimeLimit:   lo.TimeLimit,
				MemoryLimit: lo.MemoryLimit,
			})
		}
	}

	result, ucErr := h.updateUC.Execute(r.Context(), appProblem.UpdateProblemInput{
		Slug:          slug,
		Title:         body.Title,
		Statement:     body.Statement,
		TimeLimit:     body.TimeLimit,
		MemoryLimit:   body.MemoryLimit,
		LangOverrides: langOverrides,
		Tags:          body.Tags,
		Accessibility: body.Accessibility,
		CurrentUser:   *currentUser,
	})

	if ucErr != nil {
		var appErr *apperror.AppError
		if errors.As(ucErr, &appErr) {
			handler.WriteError(w, appErr)
		} else {
			slog.Error("Unexpected error in UpdateProblem", "error", ucErr, "slug", slug)
			handler.WriteError(w, apperror.NewInternal())
		}
		return
	}

	p := result.Problem
	authorDisplay, _ := h.userProvider.GetDisplay(r.Context(), p.AuthorID)
	handler.WriteJSON(w, http.StatusOK, buildResponse(p, authorDisplay))
}

func (h *Handler) Import(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{
			Code:    apperror.ErrCodeUnauthorized,
			Message: "Invalid or missing authentication token",
		})
		return
	}

	maxUploadBytes := int64(h.settings.GetMaxFileSizeTestCaseMB()) << 20
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

	result, ucErr := h.importUC.Execute(r.Context(), appProblem.ImportProblemInput{
		Slug:        slug,
		ZipData:     zipData,
		CurrentUser: *currentUser,
	})
	if ucErr != nil {
		handler.WriteError(w, ucErr)
		return
	}

	p := result.Problem
	authorDisplay, _ := h.userProvider.GetDisplay(r.Context(), p.AuthorID)
	handler.WriteJSON(w, http.StatusCreated, buildResponse(p, authorDisplay))
}
