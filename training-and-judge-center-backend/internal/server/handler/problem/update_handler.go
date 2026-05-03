package problem

import (
	"encoding/json"
	"log/slog"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type updateProblemRequest struct {
	Title         *string               `json:"title"`
	Statement     *string               `json:"statement"`
	TimeLimit     *int                  `json:"timeLimit"`
	MemoryLimit   *int                  `json:"memoryLimit"`
	LangOverrides []langOverrideRequest `json:"languageOverrides"`
	Tags          []string              `json:"tags"`
	Accessibility *string               `json:"accessibility"`
}

// @Summary      Update problem
// @Tags         problems
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        body body updateProblemRequest true "Fields to update"
// @Success      200 {object} getProblemResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
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

	var body updateProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeValidationError, Message: "Invalid request body"})
		return
	}

	var langOverrides []appProblem.LanguageOverrideInput
	if body.LangOverrides != nil {
		langOverrides = convertLangOverrides(body.LangOverrides)
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	result, ucErr := h.updateUC.Execute(r.Context(), appProblem.UpdateProblemInput{
		Slug:          slug,
		Title:         body.Title,
		Statement:     body.Statement,
		TimeLimit:     body.TimeLimit,
		MemoryLimit:   body.MemoryLimit,
		LangOverrides: langOverrides,
		Tags:          body.Tags,
		Accessibility: body.Accessibility,
		CurrentUser:   currentUser,
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
	handler.WriteJSON(w, http.StatusOK, buildResponse(p, authorDisplay))
}
