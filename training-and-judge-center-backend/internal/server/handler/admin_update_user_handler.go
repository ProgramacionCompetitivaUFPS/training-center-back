package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
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

func (h *UserHandler) AdminUpdateUser(w http.ResponseWriter, r *http.Request) {
	targetID := chi.URLParam(r, "id")

	var req adminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
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
		respondError(w, err)
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

	respondJSON(w, http.StatusOK, resp)
}
