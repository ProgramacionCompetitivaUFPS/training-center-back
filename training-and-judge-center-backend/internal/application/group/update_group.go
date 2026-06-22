package group

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type UpdateGroupInput struct {
	GroupID     string
	Name        *string
	Description *string
	Visibility  *string
	JoinPolicy  *string
	CurrentUser appshared.CurrentUser
}

type UpdateGroupOutput struct {
	ID                   string
	Name                 string
	Description          *string
	Visibility           string
	JoinPolicy           string
	IsDefault            bool
	CreatedBy            string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	MembersCount         int
	RequestsAutoApproved int
	RequestsAutoRejected int
}

type UpdateGroupUseCase struct {
	groupRepo  domainGroup.Repository
	memberRepo domainGroup.MemberRepository
	reqRepo    domainGroup.JoinRequestRepository
	txManager  appshared.TransactionManager
}

func NewUpdateGroupUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	reqRepo domainGroup.JoinRequestRepository,
	txManager appshared.TransactionManager,
) *UpdateGroupUseCase {
	return &UpdateGroupUseCase{
		groupRepo:  groupRepo,
		memberRepo: memberRepo,
		reqRepo:    reqRepo,
		txManager:  txManager,
	}
}

func (uc *UpdateGroupUseCase) Execute(ctx context.Context, in UpdateGroupInput) (*UpdateGroupOutput, error) {
	g, err := uc.groupRepo.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	if g.IsDefault() {
		return nil, apperror.NewBadRequest(domainGroup.ErrCodeCannotModifyGlobalGroup, "the global group cannot be modified")
	}

	if err := requireLeadOrAdmin(ctx, uc.memberRepo, in.GroupID, in.CurrentUser); err != nil {
		return nil, err
	}

	if in.Name == nil && in.Description == nil && in.Visibility == nil && in.JoinPolicy == nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "body", Message: "at least one field must be provided"},
		})
	}

	now := time.Now()

	var newName *domainGroup.GroupName
	if in.Name != nil {
		parsed, err := domainGroup.NewGroupName(*in.Name)
		if err != nil {
			return nil, err
		}
		if !strings.EqualFold(*in.Name, g.Name().Value()) {
			exists, err := uc.groupRepo.ExistsByName(ctx, parsed)
			if err != nil {
				return nil, err
			}
			if exists {
				return nil, apperror.NewConflict(domainGroup.ErrCodeNameAlreadyExists, "a group with this name already exists")
			}
		}
		newName = &parsed
	}

	var newVisibility *domainGroup.Visibility
	var newJoinPolicy *domainGroup.JoinPolicy

	if in.Visibility != nil {
		v, err := domainGroup.NewVisibility(*in.Visibility)
		if err != nil {
			return nil, err
		}
		newVisibility = &v
	}
	if in.JoinPolicy != nil {
		jp, err := domainGroup.NewJoinPolicy(*in.JoinPolicy)
		if err != nil {
			return nil, err
		}
		newJoinPolicy = &jp
	}

	// Auto-force INVITE when visibility becomes NOT_VISIBLE
	if newVisibility != nil && *newVisibility == domainGroup.VisibilityNotVisible {
		invite := domainGroup.JoinPolicyInvite
		newJoinPolicy = &invite
	}

	// Validate resulting combination
	if newJoinPolicy != nil {
		effectiveVisibility := g.Visibility()
		if newVisibility != nil {
			effectiveVisibility = *newVisibility
		}
		if effectiveVisibility == domainGroup.VisibilityNotVisible && *newJoinPolicy != domainGroup.JoinPolicyInvite {
			return nil, apperror.NewBadRequest(domainGroup.ErrCodeInvalidPolicyCombination, "non-visible groups can only use the INVITE join policy")
		}
	}

	oldPolicy := g.JoinPolicy()
	autoApproved := 0
	autoRejected := 0

	var pendingRequests []*domainGroup.JoinRequest
	needPending := (newVisibility != nil && *newVisibility == domainGroup.VisibilityNotVisible) ||
		(newJoinPolicy != nil && oldPolicy == domainGroup.JoinPolicyRequest)

	if needPending {
		pending := domainGroup.JoinRequestStatusPending
		reqs, _, err := uc.reqRepo.FindByGroup(ctx, in.GroupID, domainGroup.JoinRequestFilters{
			Status: &pending,
			Page:   1,
			Limit:  10000,
		})
		if err != nil {
			return nil, err
		}
		pendingRequests = reqs
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if newVisibility != nil && *newVisibility == domainGroup.VisibilityNotVisible {
			for _, req := range pendingRequests {
				if err := req.Reject(); err != nil {
					return err
				}
				if err := uc.reqRepo.Save(txCtx, req); err != nil {
					return err
				}
				autoRejected++
			}
		} else if newJoinPolicy != nil && oldPolicy == domainGroup.JoinPolicyRequest {
			if *newJoinPolicy == domainGroup.JoinPolicyOpen {
				for _, req := range pendingRequests {
					existing, err := uc.memberRepo.FindByGroupAndUser(txCtx, in.GroupID, req.RequesterUserID())
					if err != nil {
						return err
					}
					if existing != nil {
						_ = req.Reject()
						if err := uc.reqRepo.Save(txCtx, req); err != nil {
							return err
						}
						continue
					}
					if err := req.Approve(); err != nil {
						return err
					}
					if err := uc.reqRepo.Save(txCtx, req); err != nil {
						return err
					}
					newMember, err := domainGroup.NewGroupMember(
						uuid.New().String(), in.GroupID, req.RequesterUserID(),
						domainGroup.MemberRoleMember, domainGroup.JoinMethodRequestApproved, nil, now,
					)
					if err != nil {
						return err
					}
					if err := uc.memberRepo.Save(txCtx, newMember); err != nil {
						return err
					}
					autoApproved++
				}
			} else {
				for _, req := range pendingRequests {
					if err := req.Reject(); err != nil {
						return err
					}
					if err := uc.reqRepo.Save(txCtx, req); err != nil {
						return err
					}
					autoRejected++
				}
			}
		}

		var descUpdate **string
		if in.Description != nil {
			descUpdate = &in.Description
		}
		g.UpdateMetadata(newName, descUpdate, now)

		if newVisibility != nil || newJoinPolicy != nil {
			if err := g.UpdatePolicies(newVisibility, newJoinPolicy, now); err != nil {
				return err
			}
		}

		return uc.groupRepo.Update(txCtx, g)
	}); err != nil {
		return nil, err
	}

	membersCount, err := uc.memberRepo.CountMembers(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	return &UpdateGroupOutput{
		ID:                   g.ID(),
		Name:                 g.Name().Value(),
		Description:          g.Description(),
		Visibility:           g.Visibility().String(),
		JoinPolicy:           g.JoinPolicy().String(),
		IsDefault:            g.IsDefault(),
		CreatedBy:            g.CreatedBy().Value(),
		CreatedAt:            g.CreatedAt(),
		UpdatedAt:            g.UpdatedAt(),
		MembersCount:         membersCount,
		RequestsAutoApproved: autoApproved,
		RequestsAutoRejected: autoRejected,
	}, nil
}
