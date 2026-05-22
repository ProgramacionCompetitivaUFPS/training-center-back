package user

import (
	"encoding/json"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
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
// @Failure      400 {object} apperror.AppError
// @Failure      500 {object} apperror.AppError
// @Router       /password/forgot [post]
func (h *UserHandler) RequestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var body requestPasswordRecoveryBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]string{
			"error":   "INVALID_JSON",
			"message": "Request body must be valid JSON",
		})
		return
	}

	if body.Email == "" {
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, map[string]interface{}{
			"error":   "VALIDATION_ERROR",
			"message": "Invalid request data",
			"details": []map[string]string{
				{"field": "email", "message": "Email is required"},
			},
		})
		return
	}

	if err := h.requestPasswordRecovery.Execute(ctx, appuser.RequestPasswordRecoveryInput{Email: body.Email}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]string{
		"message": "If the email is registered, a recovery code has been sent",
	})
}
