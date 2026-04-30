package material

import (
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
)

func (h *Handler) Publish(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.publishUC.Execute(r.Context(), appMaterial.PublishMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     r.PathValue("groupId"),
		MaterialID:  r.PathValue("materialId"),
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildResponse(out.Material))
}
