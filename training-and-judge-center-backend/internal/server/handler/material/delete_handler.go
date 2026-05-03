package material

import (
	"errors"
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type deleteNotAuthorResponse struct {
	Error    string `json:"error"`
	Message  string `json:"message"`
	AuthorID string `json:"authorId"`
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
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

	err := h.deleteUC.Execute(r.Context(), appMaterial.DeleteMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
		MaterialID:  materialID,
	})
	if err != nil {
		var notAuthorErr *appMaterial.NotMaterialAuthorError
		if errors.As(err, &notAuthorErr) {
			handler.WriteJSON(w, http.StatusForbidden, deleteNotAuthorResponse{
				Error:    appMaterial.ErrCodeNotMaterialAuthor,
				Message:  "Only the material author can delete this material",
				AuthorID: notAuthorErr.AuthorID,
			})
			return
		}
		handler.WriteError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
