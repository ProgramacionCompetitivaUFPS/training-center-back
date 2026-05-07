package group

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type RequestJoinInput struct {
	GroupID     string
	Message     *string
	CurrentUser appshared.CurrentUser
}

type RequestJoinOutput struct {
	Request *domainGroup.JoinRequest
}

type RequestJoinUseCase struct {
	groupRepo       domainGroup.Repository
	memberRepo      domainGroup.MemberRepository
	joinRequestRepo domainGroup.JoinRequestRepository
}

func NewRequestJoinUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	joinRequestRepo domainGroup.JoinRequestRepository,
) *RequestJoinUseCase {
	return &RequestJoinUseCase{groupRepo: groupRepo, memberRepo: memberRepo, joinRequestRepo: joinRequestRepo}
}

func (uc *RequestJoinUseCase) Execute(ctx context.Context, input RequestJoinInput) (*RequestJoinOutput, error) {
	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}
	if g == nil {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
	}

	if g.Visibility() == domainGroup.VisibilityNotVisible {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
	}

	if g.JoinPolicy() != domainGroup.JoinPolicyRequest {
		return nil, apperror.NewBadRequest(ErrCodeInvalidJoinPolicy, "this group does not accept join requests")
	}

	callerID := shared.RestoreUserID(input.CurrentUser.ID)

	member, err := uc.memberRepo.FindByGroupAndUser(ctx, g.ID(), callerID)
	if err != nil {
		return nil, err
	}
	if member != nil {
		return nil, apperror.NewConflict(ErrCodeAlreadyMember, "you are already a member of this group")
	}

	existing, err := uc.joinRequestRepo.FindByGroupAndUser(ctx, g.ID(), callerID)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.IsPending() {
		return nil, apperror.NewConflict(ErrCodeRequestAlreadyPending, "you already have a pending request for this group")
	}

	now := time.Now()
	newReqID := uuid.New().String()
	req, err := domainGroup.NewJoinRequest(newReqID, g.ID(), callerID, input.Message, now)
	if err != nil {
		return nil, err
	}

	if err := uc.joinRequestRepo.Save(ctx, req); err != nil {
		slog.ErrorContext(ctx, "failed to save join request", "error", err, "group_id", g.ID())
		return nil, apperror.NewInternal()
	}

	return &RequestJoinOutput{Request: req}, nil
}
