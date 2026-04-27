package group

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/server/handler"
)

func (h *Handler) CancelMyRequest(w http.ResponseWriter, r *http.Request) {
	caller, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := chi.URLParam(r, "groupId")

	if err := h.cancelMyRequest.Execute(r.Context(), appGroup.CancelMyRequestInput{
		GroupID:     groupID,
		CurrentUser: *caller,
	}); err != nil {
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
