package user

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
)

type listUserFilterOptionsResponse struct {
	Countries    []string `json:"countries"`
	Cities       []string `json:"cities"`
	Institutions []string `json:"institutions"`
}

// @Summary      List distinct country/city/institution values in use (admin)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} listUserFilterOptionsResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /admin/users/filters [get]
func (h *Handler) ListUserFilterOptions(w http.ResponseWriter, r *http.Request) {
	out, err := h.listUserFilterOptions.Execute(r.Context())
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listUserFilterOptionsResponse{
		Countries:    out.Countries,
		Cities:       out.Cities,
		Institutions: out.Institutions,
	})
}
