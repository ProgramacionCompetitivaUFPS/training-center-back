package problem

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type changeAccessibilityRequest struct {
	Accessibility string `json:"accessibility"`
}

type changeAccessibilityResponse struct {
	Slug          string `json:"slug"`
	Accessibility string `json:"accessibility"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// @Summary      Change problem accessibility
// @Tags         problems
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        body body changeAccessibilityRequest true "Accessibility value"
// @Success      200 {object} changeAccessibilityResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /problems/p/{slug}/accessibility [patch]
func (h *Handler) ChangeAccessibility(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	var req changeAccessibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body"})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	out, err := h.changeAccessibilityUC.Execute(r.Context(), appProblem.ChangeAccessibilityInput{
		Slug:          slug,
		Accessibility: req.Accessibility,
		CurrentUser:   currentUser,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, changeAccessibilityResponse{
		Slug:          out.Problem.Slug,
		Accessibility: out.Problem.Accessibility,
		Status:        out.Problem.Status,
		Message:       fmt.Sprintf("Problem accessibility changed to %s", out.Problem.Accessibility),
	})
}
