package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type updateUserRequest struct {
	Name        *string `json:"name"`
	Nickname    *string `json:"nickname"`
	Institution *string `json:"institution"`
	City        *string `json:"city"`
	Country     *string `json:"country"`
}

// @Summary      Update my profile
// @Tags         users
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body updateUserRequest true "Fields to update"
// @Success      200 {object} fullUserResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Router       /users [put]
func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	result, err := h.updateUser.Execute(r.Context(), appuser.UpdateUserInput{
		UserID:      claims.UserID,
		Name:        req.Name,
		Nickname:    req.Nickname,
		Institution: req.Institution,
		City:        req.City,
		Country:     req.Country,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	resp := fullUserResponse{
		Name:        result.User.Name,
		Nickname:    result.User.Nickname,
		Country:     result.User.Country,
		City:        result.User.City,
		Institution: result.User.Institution,
		Role:        result.User.Role,
		CreatedAt:   result.User.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if result.User.Email != nil {
		resp.Email = *result.User.Email
	}
	if result.User.UpdatedAt != nil {
		resp.UpdatedAt = result.User.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}
