package material

import (
	"context"

	"github.com/training-judge-center/backend/internal/application/material/usecase"
)

type createMaterialUC interface {
	Execute(ctx context.Context, in usecase.CreateMaterialInput) (*usecase.CreateMaterialOutput, error)
}

type updateMaterialUC interface {
	Execute(ctx context.Context, in usecase.UpdateMaterialInput) (*usecase.UpdateMaterialOutput, error)
}

type Handler struct {
	createUC createMaterialUC
	updateUC updateMaterialUC
}

func NewHandler(createUC createMaterialUC, updateUC updateMaterialUC) *Handler {
	return &Handler{createUC: createUC, updateUC: updateUC}
}
