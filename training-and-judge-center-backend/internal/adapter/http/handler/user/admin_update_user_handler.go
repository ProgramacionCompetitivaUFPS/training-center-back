package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type adminUpdateUserRequest struct {
	Name        *string `json:"name"`
	Nickname    *string `json:"nickname"`
	Institution *string `json:"institution"`
	City        *string `json:"city"`
	Country     *string `json:"country"`
	Email       *string `json:"email"`
	Role        *string `json:"role"`
}

// @Summary      Update user (admin)
// @Tags         admin
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "User ID"
// @Param        body body adminUpdateUserRequest true "Fields to update"
// @Success      200 {object} fullUserResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /admin/users/{id} [put]
func (h *UserHandler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")

	var req adminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	result, err := h.adminUpdateUser.Execute(r.Context(), appuser.AdminUpdateUserInput{
		TargetID:    targetID,
		Name:        req.Name,
		Nickname:    req.Nickname,
		Institution: req.Institution,
		City:        req.City,
		Country:     req.Country,
		Email:       req.Email,
		Role:        req.Role,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	resp := fullUserResponse{
		Name:        result.Name,
		Nickname:    result.Nickname,
		Institution: result.Institution,
		City:        result.City,
		Country:     result.Country,
		Role:        result.Role,
		CreatedAt:   result.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if result.Email != nil {
		resp.Email = *result.Email
	}
	if result.UpdatedAt != nil {
		resp.UpdatedAt = result.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}
