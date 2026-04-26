package material

import (
	"encoding/json"
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	materialID := r.PathValue("materialId")
	if groupID == "" || materialID == "" {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "groupId and materialId are required"))
		return
	}

	var body updateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeValidationError, "Invalid request body"))
		return
	}

	out, err := h.updateUC.Execute(r.Context(), appMaterial.UpdateMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
		MaterialID:  materialID,
		Title:       body.Title,
		Content:     body.Content,
		Tags:        body.Tags,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusOK, buildResponse(out.Material))
}
