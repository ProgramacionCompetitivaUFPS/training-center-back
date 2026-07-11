package group

import (
	"context"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GenerateInviteInput struct {
	GroupID      string
	UserNickname string
	UserEmail    string
	UserID       string
	CurrentUser  appshared.CurrentUser
}

type GenerateInviteOutput struct {
	Invitation GroupInvitationDTO
	Invitee    *UserDisplay // nil for a general (link-only) invitation
}

type GenerateInviteUseCase struct {
	groupRepo        domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	invitationRepo   domainGroup.InvitationRepository
	nicknameResolver NicknameResolver
	emailResolver    EmailResolver
	userProvider     UserProvider
	txManager        appshared.TransactionManager
}

func NewGenerateInviteUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	invitationRepo domainGroup.InvitationRepository,
	nicknameResolver NicknameResolver,
	emailResolver EmailResolver,
	userProvider UserProvider,
	txManager appshared.TransactionManager,
) *GenerateInviteUseCase {
	return &GenerateInviteUseCase{
		groupRepo:        groupRepo,
		memberRepo:       memberRepo,
		invitationRepo:   invitationRepo,
		nicknameResolver: nicknameResolver,
		emailResolver:    emailResolver,
		userProvider:     userProvider,
		txManager:        txManager,
	}
}

func (uc *GenerateInviteUseCase) Execute(ctx context.Context, input GenerateInviteInput) (*GenerateInviteOutput, error) {
	if input.GroupID == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "groupId", Message: "groupId is required"},
		})
	}

	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, input.GroupID, input.CurrentUser); err != nil {
		return nil, err
	}

	if g.JoinPolicy() != domainGroup.JoinPolicyInvite {
		return nil, apperror.NewBadRequest(ErrCodeInvalidJoinPolicy, "this group does not use invite mode")
	}

	provided := 0
	if input.UserNickname != "" {
		provided++
	}
	if input.UserEmail != "" {
		provided++
	}
	if input.UserID != "" {
		provided++
	}
	if provided > 1 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "userNickname/userEmail/userId", Message: "at most one of userNickname, userEmail, or userId may be provided"},
		})
	}

	var inviteeID *shared.UserID
	var inviteeDisplay *UserDisplay

	switch {
	case input.UserNickname != "":
		d, err := uc.nicknameResolver.ResolveByNickname(ctx, input.UserNickname)
		if err != nil {
			return nil, err
		}
		if d == nil {
			return nil, apperror.NewNotFound(ErrCodeNicknameNotFound, "no user found with that nickname")
		}
		inviteeDisplay = d
		id := shared.RestoreUserID(d.ID)
		inviteeID = &id
	case input.UserEmail != "":
		d, err := uc.emailResolver.ResolveByEmail(ctx, input.UserEmail)
		if err != nil {
			return nil, err
		}
		if d == nil {
			return nil, apperror.NewNotFound(ErrCodeEmailNotFound, "no user found with that email")
		}
		inviteeDisplay = d
		id := shared.RestoreUserID(d.ID)
		inviteeID = &id
	case input.UserID != "":
		displays, err := uc.userProvider.GetDisplays(ctx, []string{input.UserID})
		if err != nil {
			return nil, err
		}
		d, ok := displays[input.UserID]
		if !ok {
			return nil, apperror.NewNotFound(ErrCodeNicknameNotFound, "no user found with that id")
		}
		inviteeDisplay = d
		id := shared.RestoreUserID(input.UserID)
		inviteeID = &id
	}

	var newInv *domainGroup.GroupInvitation
	now := time.Now()

	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		// Re-check the join policy inside the transaction: the check above
		// used a group snapshot read before the transaction started, so a
		// concurrent UpdateGroup switching the policy away from INVITE in the
		// meantime would otherwise go unnoticed and leave a dangling invitation.
		currentGroup, err := uc.groupRepo.FindByID(txCtx, input.GroupID)
		if err != nil {
			return err
		}
		if currentGroup.JoinPolicy() != domainGroup.JoinPolicyInvite {
			return apperror.NewBadRequest(ErrCodeInvalidJoinPolicy, "this group does not use invite mode")
		}

		existing, err := uc.invitationRepo.FindPendingByGroupAndInvitee(txCtx, input.GroupID, inviteeID)
		if err != nil {
			return err
		}
		if existing != nil {
			if err := existing.Revoke(); err != nil {
				return err
			}
			if err := uc.invitationRepo.TransitionStatus(txCtx, existing.ID(), domainGroup.InvitationStatusPending, domainGroup.InvitationStatusRevoked); err != nil {
				return err
			}
		}

		inv, err := domainGroup.NewGroupInvitation(uuid.New().String(), input.GroupID, inviteeID, shared.RestoreUserID(input.CurrentUser.ID), now)
		if err != nil {
			return err
		}
		if err := uc.invitationRepo.Save(txCtx, inv); err != nil {
			return err
		}
		newInv = inv
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &GenerateInviteOutput{
		Invitation: groupInvitationToDTO(newInv),
		Invitee:    inviteeDisplay,
	}, nil
}
