package material

import (
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
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
