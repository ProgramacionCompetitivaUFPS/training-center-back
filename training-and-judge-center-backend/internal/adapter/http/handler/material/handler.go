package material

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	createMaterial    *appMaterial.CreateMaterialUseCase
	updateMaterial    *appMaterial.UpdateMaterialUseCase
	getMaterial       *appMaterial.GetMaterialUseCase
	listMaterials     *appMaterial.ListMaterialsUseCase
	publishMaterial   *appMaterial.PublishMaterialUseCase
	unpublishMaterial *appMaterial.UnpublishMaterialUseCase
	pinMaterial       *appMaterial.PinMaterialUseCase
	unpinMaterial     *appMaterial.UnpinMaterialUseCase
	deleteMaterial    *appMaterial.DeleteMaterialUseCase
}

func NewHandler(
	createMaterial *appMaterial.CreateMaterialUseCase,
	updateMaterial *appMaterial.UpdateMaterialUseCase,
	getMaterial *appMaterial.GetMaterialUseCase,
	listMaterials *appMaterial.ListMaterialsUseCase,
	publishMaterial *appMaterial.PublishMaterialUseCase,
	unpublishMaterial *appMaterial.UnpublishMaterialUseCase,
	pinMaterial *appMaterial.PinMaterialUseCase,
	unpinMaterial *appMaterial.UnpinMaterialUseCase,
	deleteMaterial *appMaterial.DeleteMaterialUseCase,
) *Handler {
	return &Handler{
		createMaterial:    createMaterial,
		updateMaterial:    updateMaterial,
		getMaterial:       getMaterial,
		listMaterials:     listMaterials,
		publishMaterial:   publishMaterial,
		unpublishMaterial: unpublishMaterial,
		pinMaterial:       pinMaterial,
		unpinMaterial:     unpinMaterial,
		deleteMaterial:    deleteMaterial,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}
	return &u, true
}
