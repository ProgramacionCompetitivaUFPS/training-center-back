package group

import (
	"context"
	"log/slog"

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
	Request *domainGroup.JoinRequest
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
		slog.ErrorContext(ctx, "failed to save rejected request", "error", err)
		return nil, apperror.NewInternal()
	}

	return &RejectRequestOutput{Request: req}, nil
}
