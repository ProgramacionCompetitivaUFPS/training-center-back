package group

import (
	"context"
	"log/slog"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CancelMyRequestInput struct {
	GroupID     string
	CurrentUser shared.CurrentUser
}

type CancelMyRequestUseCase struct {
	joinRequestRepo domainGroup.JoinRequestRepository
}

func NewCancelMyRequestUseCase(joinRequestRepo domainGroup.JoinRequestRepository) *CancelMyRequestUseCase {
	return &CancelMyRequestUseCase{joinRequestRepo: joinRequestRepo}
}

func (uc *CancelMyRequestUseCase) Execute(ctx context.Context, input CancelMyRequestInput) error {
	callerID := shared.RestoreUserID(input.CurrentUser.ID)

	req, err := uc.joinRequestRepo.FindByGroupAndUser(ctx, input.GroupID, callerID)
	if err != nil {
		return err
	}
	if req == nil {
		return apperror.NewNotFound(domainGroup.ErrCodeRequestNotFound, "you have no request for this group")
	}
	if !req.IsPending() {
		return apperror.NewBadRequest(domainGroup.ErrCodeRequestAlreadyProcessed, "cannot cancel a request that has already been processed")
	}

	if err := uc.joinRequestRepo.Delete(ctx, req.ID()); err != nil {
		slog.ErrorContext(ctx, "failed to delete join request", "error", err)
		return apperror.NewInternal()
	}

	return nil
}
