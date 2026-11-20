package group

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

type GetMyRequestInput struct {
	GroupID     string
	CurrentUser appshared.CurrentUser
}

type GetMyRequestOutput struct {
	Request *JoinRequestDTO
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
		return &GetMyRequestOutput{Request: nil}, nil
	}

	dto := joinRequestToDTO(req)
	return &GetMyRequestOutput{Request: &dto}, nil
}
