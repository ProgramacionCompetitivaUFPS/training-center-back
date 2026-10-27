package handler

import (
	"encoding/json"
	"net/http"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type loginWithGoogleRequest struct {
	IDToken         string `json:"id_token"`
	RememberSession bool   `json:"rememberSession"`
}

// @Summary      Login with Google
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginWithGoogleRequest true "Google ID token"
// @Success      200 {object} loginResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /auth/google [post]
func (h *AuthHandler) LoginWithGoogle(w http.ResponseWriter, r *http.Request) {
	var req loginWithGoogleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	out, err := h.loginWithGoogleUseCase.Execute(r.Context(), appuser.LoginWithGoogleInput{
		IDToken:         req.IDToken,
		RememberSession: req.RememberSession,
		UserAgent:       userAgent(r),
		IPAddress:       clientIP(r),
	})
	if err != nil {
		WriteError(r.Context(), w, err)
		return
	}

	setRefreshCookie(w, out.RefreshToken, out.SessionExpiresAt)

	WriteJSON(r.Context(), w, http.StatusOK, loginResponse{
		Token:            out.Token,
		SessionExpiresAt: out.SessionExpiresAt.UTC().Format(time.RFC3339),
		User:             toUserResponse(out.User),
	})
}
