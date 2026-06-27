package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
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

// @Summary      Create user
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        body body createUserRequest true "User data"
// @Success      201 {object} createUserResponse
// @Failure      400 {object} apperror.AppError
// @Failure      409 {object} apperror.AppError
// @Router       /users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	out, err := h.createUser.Execute(r.Context(), appuser.CreateUserInput{
		Email:       req.Email,
		Password:    req.Password,
		Name:        req.Name,
		Nickname:    req.Nickname,
		Country:     req.Country,
		City:        req.City,
		Institution: req.Institution,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	var emailStr string
	if out.User.Email != nil {
		emailStr = *out.User.Email
	}
	handler.WriteJSON(r.Context(), w, http.StatusCreated, createUserResponse{
		Email:       emailStr,
		Name:        out.User.Name,
		Nickname:    out.User.Nickname,
		Country:     out.User.Country,
		City:        out.User.City,
		Institution: out.User.Institution,
		Role:        out.User.Role,
		CreatedAt:   out.User.CreatedAt.UTC().Format(time.RFC3339),
	})
}
