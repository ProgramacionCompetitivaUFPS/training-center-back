package group

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ApproveRequestInput struct {
	GroupID     string
	RequestID   string
	CurrentUser shared.CurrentUser
}

type ApproveRequestOutput struct {
	Request *domainGroup.JoinRequest
}

type ApproveRequestUseCase struct {
	memberRepo      domainGroup.MemberRepository
	joinRequestRepo domainGroup.JoinRequestRepository
	txManager       TransactionManager
}

func NewApproveRequestUseCase(
	memberRepo domainGroup.MemberRepository,
	joinRequestRepo domainGroup.JoinRequestRepository,
	txManager TransactionManager,
) *ApproveRequestUseCase {
	return &ApproveRequestUseCase{memberRepo: memberRepo, joinRequestRepo: joinRequestRepo, txManager: txManager}
}

func (uc *ApproveRequestUseCase) Execute(ctx context.Context, input ApproveRequestInput) (*ApproveRequestOutput, error) {
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

	if err := req.Approve(); err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(ctx context.Context) error {
		existing, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, req.RequesterUserID())
		if err != nil {
			return err
		}
		if existing != nil {
			return apperror.NewConflict(domainGroup.ErrCodeAlreadyMember, "user is already a member of this group")
		}

		if err := uc.joinRequestRepo.Save(ctx, req); err != nil {
			slog.ErrorContext(ctx, "failed to save approved request", "error", err)
			return apperror.NewInternal()
		}

		now := time.Now()
		newMemberID := uuid.New().String()
		newMember, err := domainGroup.NewGroupMember(newMemberID, input.GroupID, req.RequesterUserID(), domainGroup.MemberRoleMember, now)
		if err != nil {
			return err
		}
		return uc.memberRepo.Save(ctx, newMember)
	}); err != nil {
		return nil, err
	}

	return &ApproveRequestOutput{Request: req}, nil
}

