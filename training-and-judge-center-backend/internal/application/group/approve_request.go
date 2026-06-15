package group

import (
	"context"

	"time"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ApproveRequestInput struct {
	GroupID     string
	RequestID   string
	CurrentUser appshared.CurrentUser
}

type ApproveRequestOutput struct {
	Request JoinRequestDTO
}

type ApproveRequestUseCase struct {
	memberRepo      domainGroup.MemberRepository
	joinRequestRepo domainGroup.JoinRequestRepository
	txManager       appshared.TransactionManager
}

func NewApproveRequestUseCase(
	memberRepo domainGroup.MemberRepository,
	joinRequestRepo domainGroup.JoinRequestRepository,
	txManager appshared.TransactionManager,
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

	now := time.Now()
	newMemberID := uuid.New().String()
	newMember, err := domainGroup.NewGroupMember(newMemberID, input.GroupID, req.RequesterUserID(), domainGroup.MemberRoleMember, domainGroup.JoinMethodRequestApproved, nil, now)
	if err != nil {
		return nil, err
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		existing, err := uc.memberRepo.FindByGroupAndUser(txCtx, input.GroupID, req.RequesterUserID())
		if err != nil {
			return err
		}
		if existing != nil {
			return apperror.NewConflict(domainGroup.ErrCodeAlreadyMember, "user is already a member of this group")
		}

		if err := uc.joinRequestRepo.Save(txCtx, req); err != nil {
			return err
		}

		return uc.memberRepo.Save(txCtx, newMember)
	}); err != nil {
		return nil, err
	}

	return &ApproveRequestOutput{Request: joinRequestToDTO(req)}, nil
}

