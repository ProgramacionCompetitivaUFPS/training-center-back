package group

import (
	"context"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetMyRequestInput struct {
	GroupID     string
	CurrentUser shared.CurrentUser
}

type GetMyRequestOutput struct {
	Request *domainGroup.JoinRequest
}

type GetMyRequestUseCase struct {
	joinRequestRepo domainGroup.JoinRequestRepository
}

func NewGetMyRequestUseCase(joinRequestRepo domainGroup.JoinRequestRepository) *GetMyRequestUseCase {
	return &GetMyRequestUseCase{joinRequestRepo: joinRequestRepo}
}

func (uc *GetMyRequestUseCase) Execute(ctx context.Context, input GetMyRequestInput) (*GetMyRequestOutput, error) {
	callerID := shared.RestoreUserID(input.CurrentUser.ID)

	req, err := uc.joinRequestRepo.FindByGroupAndUser(ctx, input.GroupID, callerID)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeRequestNotFound, "you have not requested to join this group")
	}

	return &GetMyRequestOutput{Request: req}, nil
}
