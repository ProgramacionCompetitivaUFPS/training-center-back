package handler

import (
	"encoding/json"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
)

type requestPasswordRecoveryBody struct {
	Email string `json:"email"`
}

func (h *UserHandler) RequestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body requestPasswordRecoveryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Email == "" {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "email", "message": "Email is required"},
			},
		})
		return
	}

	if err := h.requestPasswordRecovery.Execute(ctx, appuser.RequestPasswordRecoveryInput{Email: body.Email}); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "If the email is registered, a recovery code has been sent",
	})
}
