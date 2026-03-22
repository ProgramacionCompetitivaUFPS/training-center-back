package handler

import (
	"encoding/json"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
)

func (h *UserHandler) RequestDeactivation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}
	userID := claims.UserID
	ctx := r.Context()

	if err := h.requestDeactivation.Execute(ctx, userID); err != nil {
		respondError(w, err)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{
		"message": "A confirmation code has been sent to your email",
	})
}

type confirmDeactivationBody struct {
	Code string `json:"code"`
}

func (h *UserHandler) ConfirmDeactivation(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{
			"error":   "UNAUTHORIZED",
			"message": "Invalid or missing authentication token",
		})
		return
	}
	userID := claims.UserID
	ctx := r.Context()

	var body confirmDeactivationBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Code == "" || len(body.Code) != 6 {
		respondJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "code", "message": "Code must be 6 characters long"},
			},
		})
		return
	}

	ip := r.RemoteAddr
	userAgent := r.UserAgent()

	input := appuser.ConfirmDeactivationInput{
		UserID:    userID,
		Code:      body.Code,
		IP:        &ip,
		UserAgent: &userAgent,
	}

	if err := h.confirmDeactivation.Execute(ctx, input); err != nil {
		respondError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
