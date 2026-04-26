package material

import (
	"context"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
)

type createMaterialUC interface {
	Execute(ctx context.Context, in appMaterial.CreateMaterialInput) (*appMaterial.CreateMaterialOutput, error)
}

type updateMaterialUC interface {
	Execute(ctx context.Context, in appMaterial.UpdateMaterialInput) (*appMaterial.UpdateMaterialOutput, error)
}

type Handler struct {
	createUC createMaterialUC
	updateUC updateMaterialUC
}

func NewHandler(createUC createMaterialUC, updateUC updateMaterialUC) *Handler {
	return &Handler{createUC: createUC, updateUC: updateUC}
}
