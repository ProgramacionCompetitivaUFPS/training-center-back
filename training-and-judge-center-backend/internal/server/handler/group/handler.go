package group

import (
	"context"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

type listGroupsUC interface {
	Execute(ctx context.Context, in appGroup.ListGroupsInput) (*appGroup.ListGroupsOutput, error)
}

type getGroupUC interface {
	Execute(ctx context.Context, in appGroup.GetGroupInput) (*appGroup.GetGroupOutput, error)
}

type listMyGroupsUC interface {
	Execute(ctx context.Context, in appGroup.ListMyGroupsInput) (*appGroup.ListMyGroupsOutput, error)
}

type Handler struct {
	listUC   listGroupsUC
	getUC    getGroupUC
	listMyUC listMyGroupsUC
}

func NewHandler(listUC listGroupsUC, getUC getGroupUC, listMyUC listMyGroupsUC) *Handler {
	return &Handler{listUC: listUC, getUC: getUC, listMyUC: listMyUC}
}
