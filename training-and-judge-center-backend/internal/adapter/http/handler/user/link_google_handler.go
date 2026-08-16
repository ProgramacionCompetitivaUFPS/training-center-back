package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type linkGoogleRequest struct {
	IDToken string `json:"id_token"`
}

// @Summary      Link Google account
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body linkGoogleRequest true "Google ID token"
// @Success      204
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /users/google [post]
func (h *Handler) LinkGoogle(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	var req linkGoogleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	if err := h.linkGoogle.Execute(r.Context(), appuser.LinkGoogleIdentityInput{
		UserID:  cu.ID,
		IDToken: req.IDToken,
	}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
