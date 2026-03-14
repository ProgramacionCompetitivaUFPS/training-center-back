package handler

import (
	"encoding/json"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type createUserRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Institution string `json:"institution"`
}

type createUserResponse struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Institution string `json:"institution"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
}

func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	result, err := h.createUser.Execute(r.Context(), appuser.CreateUserInput{
		Email:       req.Email,
		Password:    req.Password,
		Name:        req.Name,
		Nickname:    req.Nickname,
		Country:     req.Country,
		City:        req.City,
		Institution: req.Institution,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusCreated, createUserResponse{
		Email:       result.Email.String(),
		Name:        result.Name,
		Nickname:    result.Nickname.String(),
		Country:     result.Country,
		City:        result.City,
		Institution: result.Institution,
		Role:        result.Role.String(),
		CreatedAt:   result.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
