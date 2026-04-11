package problem

import (
	"encoding/json"
	"fmt"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type unpublishResponse struct {
	Slug    string `json:"slug"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type changeAccessibilityRequest struct {
	Accessibility string `json:"accessibility"`
}

type changeAccessibilityResponse struct {
	Slug          string `json:"slug"`
	Accessibility string `json:"accessibility"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

func (h *Handler) Unpublish(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	p, err := h.unpublishUC.Execute(r.Context(), appProblem.UnpublishProblemInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, unpublishResponse{
		Slug:    p.Slug().String(),
		Status:  p.Status().String(),
		Message: "Problem unpublished successfully. You can now make changes.",
	})
}

func (h *Handler) ChangeAccessibility(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	var req changeAccessibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body"})
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	p, err := h.changeAccessibilityUC.Execute(r.Context(), appProblem.ChangeAccessibilityInput{
		Slug:          slug,
		Accessibility: req.Accessibility,
		CurrentUser:   currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, changeAccessibilityResponse{
		Slug:          p.Slug().String(),
		Accessibility: p.Accessibility().String(),
		Status:        p.Status().String(),
		Message:       fmt.Sprintf("Problem accessibility changed to %s", p.Accessibility().String()),
	})
}
