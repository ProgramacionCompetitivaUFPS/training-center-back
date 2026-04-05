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

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	result, err := h.loginUseCase.Execute(r.Context(), appuser.LoginInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, loginResponse{
		Token: result.Token,
		User: userResponse{
			Email:       result.User.Email().String(),
			Name:        result.User.Name(),
			Nickname:    result.User.Nickname().String(),
			Country:     result.User.Country(),
			City:        result.User.City(),
			Institution: result.User.Institution(),
			Role:        result.User.Role().String(),
			CreatedAt:   result.User.CreatedAt().Format("2006-01-02T15:04:05Z"),
		},
	})
}
