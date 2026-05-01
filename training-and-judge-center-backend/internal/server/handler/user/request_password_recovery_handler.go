package user

import (
	"encoding/json"
	"net/http"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
)

type requestPasswordRecoveryBody struct {
	Email string `json:"email"`
}

// @Summary      Request password recovery
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body requestPasswordRecoveryBody true "Email"
// @Success      200 {object} map[string]string
// @Router       /password/forgot [post]
func (h *UserHandler) RequestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body requestPasswordRecoveryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Email == "" {
		handler.WriteJSON(w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "email", "message": "Email is required"},
			},
		})
		return
	}

	if err := h.requestPasswordRecovery.Execute(ctx, appuser.RequestPasswordRecoveryInput{Email: body.Email}); err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, map[string]string{
		"message": "If the email is registered, a recovery code has been sent",
	})
}
