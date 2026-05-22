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
	createUC    *appMaterial.CreateMaterialUseCase
	updateUC    *appMaterial.UpdateMaterialUseCase
	getUC       *appMaterial.GetMaterialUseCase
	listUC      *appMaterial.ListMaterialsUseCase
	publishUC   *appMaterial.PublishMaterialUseCase
	unpublishUC *appMaterial.UnpublishMaterialUseCase
	pinUC       *appMaterial.PinMaterialUseCase
	unpinUC     *appMaterial.UnpinMaterialUseCase
	deleteUC    *appMaterial.DeleteMaterialUseCase
}

func NewHandler(
	createUC *appMaterial.CreateMaterialUseCase,
	updateUC *appMaterial.UpdateMaterialUseCase,
	getUC *appMaterial.GetMaterialUseCase,
	listUC *appMaterial.ListMaterialsUseCase,
	publishUC *appMaterial.PublishMaterialUseCase,
	unpublishUC *appMaterial.UnpublishMaterialUseCase,
	pinUC *appMaterial.PinMaterialUseCase,
	unpinUC *appMaterial.UnpinMaterialUseCase,
	deleteUC *appMaterial.DeleteMaterialUseCase,
) *Handler {
	return &Handler{
		createUC:    createUC,
		updateUC:    updateUC,
		getUC:       getUC,
		listUC:      listUC,
		publishUC:   publishUC,
		unpublishUC: unpublishUC,
		pinUC:       pinUC,
		unpinUC:     unpinUC,
		deleteUC:    deleteUC,
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
