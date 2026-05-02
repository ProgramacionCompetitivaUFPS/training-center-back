package material

import (
	"errors"
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
)

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	err := h.deleteUC.Execute(r.Context(), appMaterial.DeleteMaterialInput{
		CurrentUser: *currentUser,
		GroupID:     r.PathValue("groupId"),
		MaterialID:  r.PathValue("materialId"),
	})
	if err != nil {
		var notAuthorErr *appMaterial.NotMaterialAuthorError
		if errors.As(err, &notAuthorErr) {
			handler.WriteJSON(w, http.StatusForbidden, struct {
				Error    string `json:"error"`
				Message  string `json:"message"`
				AuthorID string `json:"authorId"`
			}{
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
