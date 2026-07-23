package group

import (
	"context"


	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RejectRequestInput struct {
	GroupID     string
	RequestID   string
	CurrentUser appshared.CurrentUser
}

type RejectRequestOutput struct {
	Request JoinRequestDTO
}

type RejectRequestUseCase struct {
	memberRepo      domainGroup.MemberRepository
	joinRequestRepo domainGroup.JoinRequestRepository
}

func NewRejectRequestUseCase(
	memberRepo domainGroup.MemberRepository,
	joinRequestRepo domainGroup.JoinRequestRepository,
) *RejectRequestUseCase {
	return &RejectRequestUseCase{memberRepo: memberRepo, joinRequestRepo: joinRequestRepo}
}

func (uc *RejectRequestUseCase) Execute(ctx context.Context, input RejectRequestInput) (*RejectRequestOutput, error) {
	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	req, err := uc.joinRequestRepo.FindByID(ctx, input.RequestID)
	if err != nil {
		return nil, err
	}
	if req == nil || req.GroupID() != input.GroupID {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeRequestNotFound, "join request not found")
	}

	if err := req.Reject(); err != nil {
		return nil, err
	}

	if err := uc.joinRequestRepo.Save(ctx, req); err != nil {
		return nil, err
	}

	return &RejectRequestOutput{Request: joinRequestToDTO(req)}, nil
}
