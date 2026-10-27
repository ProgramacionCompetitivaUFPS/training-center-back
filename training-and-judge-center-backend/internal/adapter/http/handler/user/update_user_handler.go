package user

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type updateUserRequest struct {
	Name        *string `json:"name"`
	Nickname    *string `json:"nickname"`
	Institution *string `json:"institution"`
	City        *string `json:"city"`
	Country     *string `json:"country"`
}

type updatedUserResponse struct {
	Email       string `json:"email"`
	Name        string `json:"name"`
	Nickname    string `json:"nickname"`
	Country     string `json:"country"`
	City        string `json:"city"`
	Institution string `json:"institution"`
	Role        string `json:"role"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt,omitempty"`
}

// @Summary      Update my profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body updateUserRequest true "Fields to update"
// @Success      200 {object} updatedUserResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /users [put]
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	out, err := h.updateUser.Execute(r.Context(), appuser.UpdateUserInput{
		UserID:      cu.ID,
		Name:        req.Name,
		Nickname:    req.Nickname,
		Institution: req.Institution,
		City:        req.City,
		Country:     req.Country,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildUpdatedResponse(out.User))
}

func buildUpdatedResponse(u appuser.UserDTO) updatedUserResponse {
	resp := updatedUserResponse{
		Name:        u.Name,
		Nickname:    u.Nickname,
		Country:     u.Country,
		City:        u.City,
		Institution: u.Institution,
		Role:        u.Role,
		CreatedAt:   u.CreatedAt.UTC().Format(time.RFC3339),
	}
	if u.Email != nil {
		resp.Email = *u.Email
	}
	if u.UpdatedAt != nil {
		resp.UpdatedAt = u.UpdatedAt.UTC().Format(time.RFC3339)
	}
	return resp
}
