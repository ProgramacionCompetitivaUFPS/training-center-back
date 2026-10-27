package handler

import (
	"net/http"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type refreshResponse struct {
	Token            string `json:"token"`
	SessionExpiresAt string `json:"sessionExpiresAt"`
}

// @Summary      Refresh access token
// @Tags         auth
// @Produce      json
// @Success      200 {object} refreshResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      429 {object} apperror.AppError
// @Router       /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	secret, ok := readRefreshCookie(r)
	if !ok {
		WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized,
			"Invalid or expired session. Please log in again"))
		return
	}

	out, err := h.refreshUseCase.Execute(r.Context(), appuser.RefreshInput{
		RefreshToken: secret,
		UserAgent:    userAgent(r),
		IPAddress:    clientIP(r),
	})
	if err != nil {
		WriteError(r.Context(), w, err)
		return
	}

	if out.RefreshToken != "" {
		setRefreshCookie(w, out.RefreshToken, out.SessionExpiresAt)
	}

	WriteJSON(r.Context(), w, http.StatusOK, refreshResponse{
		Token:            out.Token,
		SessionExpiresAt: out.SessionExpiresAt.UTC().Format(time.RFC3339),
	})
}
