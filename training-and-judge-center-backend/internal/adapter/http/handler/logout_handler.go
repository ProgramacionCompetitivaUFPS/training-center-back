package handler

import (
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

// @Summary      Logout
// @Tags         auth
// @Success      204
// @Failure      500 {object} apperror.AppError
// @Router       /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	wrapped, _ := readRefreshCookie(r)

	if err := h.logoutUseCase.Execute(r.Context(), appuser.LogoutInput{RefreshToken: wrapped}); err != nil {
		WriteError(r.Context(), w, err)
		return
	}

	clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}
