package user

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
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
// @Success      200 {object} updatedUserResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /admin/users/{id} [put]
func (h *Handler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")

	var req adminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	out, err := h.adminUpdateUser.Execute(r.Context(), appuser.AdminUpdateUserInput{
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
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, buildUpdatedResponse(out.User))
}
