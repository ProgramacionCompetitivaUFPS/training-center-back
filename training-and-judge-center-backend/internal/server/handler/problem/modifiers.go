package problem

import (
	"encoding/json"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) AddModifier(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	var body struct {
		UserID string `json:"userId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.UserID == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body or missing userId"})
		return
	}

	currentUser := user.CurrentUser{ID: claims.UserID, Role: claims.Role}

	_, err := h.addModifierUC.Execute(r.Context(), appProblem.AddModifierInput{
		Slug:        slug,
		UserID:      body.UserID,
		CurrentUser: currentUser,
	})

	if err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) RemoveModifier(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	userID := r.PathValue("userId")
	if slug == "" || userID == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Slug and userId are required"})
		return
	}

	currentUser := user.CurrentUser{ID: claims.UserID, Role: claims.Role}

	_, err := h.removeModifierUC.Execute(r.Context(), appProblem.RemoveModifierInput{
		Slug:        slug,
		UserID:      userID,
		CurrentUser: currentUser,
	})

	if err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListModifiers(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid token"})
		return
	}

	slug := r.PathValue("slug")
	if slug == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Slug is required"})
		return
	}

	currentUser := user.CurrentUser{ID: claims.UserID, Role: claims.Role}

	modifiers, err := h.listModifiersUC.Execute(r.Context(), appProblem.ListModifiersInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{"modifiers": modifiers})
}
