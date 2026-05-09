package handler

import (
	"encoding/json"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
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
	loginUseCase *appuser.LoginUseCase
}

func NewAuthHandler(loginUseCase *appuser.LoginUseCase) *AuthHandler {
	return &AuthHandler{loginUseCase: loginUseCase}
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
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	out, err := h.loginUseCase.Execute(r.Context(), appuser.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	var email string
	if out.User.Email != nil {
		email = *out.User.Email
	}
	respondJSON(w, http.StatusOK, loginResponse{
		Token: out.Token,
		User: userResponse{
			Email:       email,
			Name:        out.User.Name,
			Nickname:    out.User.Nickname,
			Country:     out.User.Country,
			City:        out.User.City,
			Institution: out.User.Institution,
			Role:        out.User.Role,
			CreatedAt:   out.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}
