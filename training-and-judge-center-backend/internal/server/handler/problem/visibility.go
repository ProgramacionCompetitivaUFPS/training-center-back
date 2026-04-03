package problem

import (
	"encoding/json"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
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
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	out, err := h.unpublishUC.Execute(r.Context(), appProblem.UnpublishProblemInput{
		Slug:        slug,
		CurrentUser: *currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, unpublishResponse{
		Slug:    out.Slug,
		Status:  out.Status,
		Message: out.Message,
	})
}

func (h *Handler) ChangeAccessibility(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	var req changeAccessibilityRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body"})
		return
	}

	out, err := h.changeAccessibilityUC.Execute(r.Context(), appProblem.ChangeAccessibilityInput{
		Slug:          slug,
		Accessibility: req.Accessibility,
		CurrentUser:   *currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, changeAccessibilityResponse{
		Slug:          out.Slug,
		Accessibility: out.Accessibility,
		Status:        out.Status,
		Message:       out.Message,
	})
}
