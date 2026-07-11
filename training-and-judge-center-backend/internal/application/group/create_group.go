package group

import (
	"context"
	"fmt"

	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateGroupInput struct {
	Name            string
	Description     *string
	JoinMode        string
	Visibility      string
	MemberNicknames []string
	LeadNicknames   []string
	CurrentUser     appshared.CurrentUser
}

// InitialMember describes a member/lead actually added at group creation time
// (excludes the creator, who is always the initial lead). It lives here, not
// in dto.go, since only CreateGroupOutput uses it (A4).
type InitialMember struct {
	UserID   string
	Nickname string
	Name     string
	Role     string
	JoinedAt time.Time
}

type CreateGroupOutput struct {
	ID          string
	Name        string
	Description *string
	JoinPolicy  string
	Visibility  string
	IsDefault   bool
	CreatedBy   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Members     []InitialMember
}

type CreateGroupUseCase struct {
	repo             domainGroup.Repository
	memberRepo       domainGroup.MemberRepository
	nicknameResolver NicknameResolver
	txManager        appshared.TransactionManager
}

func NewCreateGroupUseCase(
	repo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	nicknameResolver NicknameResolver,
	txManager appshared.TransactionManager,
) *CreateGroupUseCase {
	return &CreateGroupUseCase{repo: repo, memberRepo: memberRepo, nicknameResolver: nicknameResolver, txManager: txManager}
}

func (uc *CreateGroupUseCase) Execute(ctx context.Context, input CreateGroupInput) (*CreateGroupOutput, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(
			ErrCodeInsufficientPermissions,
			"Only Admin and Coach users can create groups",
		)
	}

	var fieldErrs []apperror.FieldError

	groupName, err := domainGroup.NewGroupName(input.Name)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	joinPolicy, err := domainGroup.NewJoinPolicy(input.JoinMode)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	visibility, err := domainGroup.NewVisibility(input.Visibility)
	if err := apperror.AccumulateFieldErrors(err, &fieldErrs); err != nil {
		return nil, err
	}

	for i, n := range input.MemberNicknames {
		if n == "" {
			fieldErrs = append(fieldErrs, apperror.FieldError{Field: fmt.Sprintf("memberNicknames[%d]", i), Message: "nickname must not be empty"})
		}
	}
	for i, n := range input.LeadNicknames {
		if n == "" {
			fieldErrs = append(fieldErrs, apperror.FieldError{Field: fmt.Sprintf("leadNicknames[%d]", i), Message: "nickname must not be empty"})
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	exists, err := uc.repo.ExistsByName(ctx, groupName)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.NewConflict(domainGroup.ErrCodeNameAlreadyExists, "A group with this name already exists")
	}

	now := time.Now()
	newID := uuid.New().String()
	creatorID := shared.RestoreUserID(input.CurrentUser.ID)

	leads, err := resolveMemberNicknames(ctx, uc.nicknameResolver, input.LeadNicknames, domainGroup.MemberRoleLead)
	if err != nil {
		return nil, err
	}
	members, err := resolveMemberNicknames(ctx, uc.nicknameResolver, input.MemberNicknames, domainGroup.MemberRoleMember)
	if err != nil {
		return nil, err
	}
	leads = excludeUserID(leads, creatorID)
	members = excludeUserID(members, creatorID)
	members = excludeByUserID(members, leads)

	g, err := domainGroup.NewGroup(newID, groupName, input.Description, visibility, joinPolicy, creatorID, now)
	if err != nil {
		return nil, err
	}

	newLeadMemberID := uuid.New().String()
	creatorLead, err := domainGroup.NewGroupMember(newLeadMemberID, newID, creatorID, domainGroup.MemberRoleLead, domainGroup.JoinMethodDirectAdd, nil, now)
	if err != nil {
		return nil, err
	}

	extraMembers := make([]*domainGroup.GroupMember, 0, len(leads)+len(members))
	added := make([]InitialMember, 0, len(leads)+len(members))
	for _, role := range []struct {
		entries []resolvedNickname
		role    domainGroup.MemberRole
	}{
		{leads, domainGroup.MemberRoleLead},
		{members, domainGroup.MemberRoleMember},
	} {
		for _, entry := range role.entries {
			m, err := domainGroup.NewGroupMember(uuid.New().String(), newID, entry.userID, role.role, domainGroup.JoinMethodDirectAdd, &creatorID, now)
			if err != nil {
				return nil, err
			}
			extraMembers = append(extraMembers, m)
			added = append(added, InitialMember{
				UserID:   entry.userID.Value(),
				Nickname: entry.display.Nickname,
				Name:     entry.display.Name,
				Role:     role.role.String(),
				JoinedAt: now,
			})
		}
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.repo.Save(txCtx, g); err != nil {
			return err
		}
		if err := uc.memberRepo.Save(txCtx, creatorLead); err != nil {
			return err
		}
		if len(extraMembers) > 0 {
			return uc.memberRepo.SaveAll(txCtx, extraMembers)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	return &CreateGroupOutput{
		ID:          g.ID(),
		Name:        g.Name().Value(),
		Description: g.Description(),
		JoinPolicy:  g.JoinPolicy().String(),
		Visibility:  g.Visibility().String(),
		IsDefault:   g.IsDefault(),
		CreatedBy:   g.CreatedBy().Value(),
		CreatedAt:   g.CreatedAt(),
		UpdatedAt:   g.UpdatedAt(),
		Members:     added,
	}, nil
}
