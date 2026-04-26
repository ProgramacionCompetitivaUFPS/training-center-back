package material

import (
	"encoding/json"
	"net/http"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())

	groupID := r.PathValue("groupId")
	if groupID == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeBadRequest,
			Message: "groupId is required",
		})
		return
	}

	var body createMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	out, err := h.createUC.Execute(r.Context(), appMaterial.CreateMaterialInput{
		CurrentUser: shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()},
		GroupID:     groupID,
		Title:       body.Title,
		Content:     body.Content,
		Tags:        body.Tags,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	handler.WriteJSON(w, http.StatusCreated, buildResponse(out.Material))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())

	groupID := r.PathValue("groupId")
	materialID := r.PathValue("materialId")
	if groupID == "" || materialID == "" {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeBadRequest,
			Message: "groupId and materialId are required",
		})
		return
	}

	var body updateMaterialRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request body",
		})
		return
	}

	out, err := h.updateUC.Execute(r.Context(), appMaterial.UpdateMaterialInput{
		CurrentUser: shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()},
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
