package user

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// requestDeactivationRequest is optional: accounts without a password (Google-only) may send
// no body at all.
type requestDeactivationRequest struct {
	Password string `json:"password"`
}

// maxRequestDeactivationBody caps the body here instead of relying on middleware.LimitBody, which
// lets anything labelled multipart/* through untouched — a JSON body sent with that Content-Type
// would otherwise be decoded with no size limit. The body is just a password.
const maxRequestDeactivationBody = 4 << 10

// @Summary      Request account deactivation
// @Tags         users
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} map[string]string
// @Failure      401 {object} apperror.AppError
// @Router       /users/deactivation [post]
func (h *Handler) RequestDeactivation(w http.ResponseWriter, r *http.Request) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "invalid or missing authentication token"})
		return
	}
	userID := cu.ID
	ctx := r.Context()

	var req requestDeactivationRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestDeactivationBody)).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			handler.WriteJSON(r.Context(), w, http.StatusRequestEntityTooLarge, apperror.AppError{Code: "PAYLOAD_TOO_LARGE", Message: "Request body is too large"})
			return
		}
		handler.WriteJSON(r.Context(), w, http.StatusBadRequest, apperror.AppError{Code: "INVALID_JSON", Message: "request body must be valid JSON"})
		return
	}

	if err := h.requestDeactivation.Execute(ctx, appuser.RequestDeactivationInput{UserID: userID, Password: req.Password}); err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, map[string]string{
		"message": "A confirmation code has been sent to your email",
	})
}
