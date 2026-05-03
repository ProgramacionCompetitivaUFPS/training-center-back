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

type createProblemRequest struct {
	Slug          string                `json:"slug"`
	Title         string                `json:"title"`
	Statement     *string               `json:"statement"`
	TimeLimit     *int                  `json:"timeLimit"`
	MemoryLimit   *int                  `json:"memoryLimit"`
	LangOverrides []langOverrideRequest `json:"languageOverrides"`
	Tags          []string              `json:"tags"`
}

// @Summary      Create problem
// @Tags         problems
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createProblemRequest true "Problem data"
// @Success      201 {object} getProblemResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /problems [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
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

	langOverrides := convertLangOverrides(body.LangOverrides)
	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	result, ucErr := h.createUC.Execute(r.Context(), appProblem.CreateProblemInput{
		Slug:          body.Slug,
		Title:         body.Title,
		Statement:     body.Statement,
		TimeLimit:     body.TimeLimit,
		MemoryLimit:   body.MemoryLimit,
		LangOverrides: langOverrides,
		Tags:          body.Tags,
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

	handler.WriteJSON(w, http.StatusCreated, buildResponse(p, authorDisplay))
}
