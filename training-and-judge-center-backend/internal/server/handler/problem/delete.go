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

type deleteProblemRequest struct {
	ConfirmSlug string `json:"confirmSlug"`
}

func (h *Handler) DeleteProblem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")

	var req deleteProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{Code: apperror.ErrCodeBadRequest, Message: "Invalid request body"})
		return
	}

	currentUser := user.CurrentUser{ID: claims.UserID, Role: claims.Role}

	_, err := h.deleteProblemUC.Execute(r.Context(), appProblem.DeleteProblemInput{
		Slug:        slug,
		ConfirmSlug: req.ConfirmSlug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
