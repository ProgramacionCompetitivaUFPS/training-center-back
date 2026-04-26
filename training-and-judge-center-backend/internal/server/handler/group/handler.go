package group

import (
	appGroup "github.com/training-judge-center/backend/internal/application/group"
)

type GroupHandler struct {
	createGroup *appGroup.CreateGroupUseCase
}

func NewGroupHandler(createGroup *appGroup.CreateGroupUseCase) *GroupHandler {
	return &GroupHandler{
		createGroup: createGroup,
	}
}
