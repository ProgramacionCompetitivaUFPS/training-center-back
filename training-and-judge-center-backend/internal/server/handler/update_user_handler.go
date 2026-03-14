package handler

import (
	"encoding/json"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

type updateUserRequest struct {
	Name        *string `json:"name"`
	Nickname    *string `json:"nickname"`
	Institution *string `json:"institution"`
}

func (h *UserHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}

	var req updateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	result, err := h.updateUser.Execute(r.Context(), claims.UserID, appuser.UpdateUserInput{
		Name:        req.Name,
		Nickname:    req.Nickname,
		Institution: req.Institution,
	})
	if err != nil {
		respondError(w, err)
		return
	}

	resp := fullUserResponse{
		Email:       result.Email.String(),
		Name:        result.Name,
		Nickname:    result.Nickname.String(),
		Country:     result.Country,
		City:        result.City,
		Institution: result.Institution,
		Role:        result.Role.String(),
		CreatedAt:   result.CreatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if result.UpdatedAt != nil {
		resp.UpdatedAt = result.UpdatedAt.Format("2006-01-02T15:04:05Z")
	}

	respondJSON(w, http.StatusOK, resp)
}
