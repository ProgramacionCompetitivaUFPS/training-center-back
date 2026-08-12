package handler

import (
	"encoding/json"
	"net/http"
	"time"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type loginRequest struct {
	Email           string `json:"email"`
	Password        string `json:"password"`
	RememberSession bool   `json:"rememberSession"`
}

type loginResponse struct {
	Token            string       `json:"token"`
	SessionExpiresAt string       `json:"sessionExpiresAt"`
	User             userResponse `json:"user"`
}

type userResponse struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Institution string `json:"institution"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
}

type AuthHandler struct {
	loginUseCase   *appuser.LoginUseCase
	refreshUseCase *appuser.RefreshUseCase
	logoutUseCase  *appuser.LogoutUseCase
}

func NewAuthHandler(loginUseCase *appuser.LoginUseCase, refreshUseCase *appuser.RefreshUseCase, logoutUseCase *appuser.LogoutUseCase) *AuthHandler {
	return &AuthHandler{loginUseCase: loginUseCase, refreshUseCase: refreshUseCase, logoutUseCase: logoutUseCase}
}

// @Summary      Login
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "Credentials"
// @Success      200 {object} loginResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	out, err := h.loginUseCase.Execute(r.Context(), appuser.LoginInput{
		Email:           req.Email,
		Password:        req.Password,
		RememberSession: req.RememberSession,
		UserAgent:       userAgent(r),
		IPAddress:       clientIP(r),
	})
	if err != nil {
		WriteError(r.Context(), w, err)
		return
	}

	setRefreshCookie(w, out.RefreshToken, out.SessionExpiresAt)

	var email string
	if out.User.Email != nil {
		email = *out.User.Email
	}
	WriteJSON(r.Context(), w, http.StatusOK, loginResponse{
		Token:            out.Token,
		SessionExpiresAt: out.SessionExpiresAt.UTC().Format(time.RFC3339),
		User: userResponse{
			Email:       email,
			Name:        out.User.Name,
			Nickname:    out.User.Nickname,
			Country:     out.User.Country,
			City:        out.User.City,
			Institution: out.User.Institution,
			Role:        out.User.Role,
			CreatedAt:   out.User.CreatedAt.UTC().Format(time.RFC3339),
		},
	})
}
