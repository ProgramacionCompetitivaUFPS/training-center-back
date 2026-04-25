package group

import (
	"context"
	"log/slog"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
	"github.com/training-judge-center/backend/pkg/timeutil"
)

type GetGroupInput struct {
	GroupID     string
	CurrentUser shared.CurrentUser
}

type LeadDisplay struct {
	UserID   string
	Nickname string
	Name     string
}

type GroupStatistics struct {
	MemberCount int
	LeadCount   int
}

type UserMembership struct {
	IsMember bool
	Role     *domainGroup.MemberRole
	JoinedAt *string
}

type GetGroupOutput struct {
	Group      *domainGroup.Group
	Statistics GroupStatistics
	Leads      []LeadDisplay
	Membership UserMembership
}

type GetGroupUseCase struct {
	repo         domainGroup.Repository
	memberRepo   domainGroup.MemberRepository
	userProvider UserProvider
}

func NewGetGroupUseCase(repo domainGroup.Repository, memberRepo domainGroup.MemberRepository, userProvider UserProvider) *GetGroupUseCase {
	return &GetGroupUseCase{repo: repo, memberRepo: memberRepo, userProvider: userProvider}
}

func (uc *GetGroupUseCase) Execute(ctx context.Context, in GetGroupInput) (*GetGroupOutput, error) {
	if in.GroupID == "" {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "groupId", Message: "groupId is required"},
		})
	}

	g, err := uc.repo.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.IsAdmin()

	membership, err := uc.memberRepo.FindByGroupAndUser(ctx, g.ID(), viewerID)
	if err != nil {
		return nil, err
	}

	if g.Visibility() == domainGroup.VisibilityNotVisible && membership == nil && !isAdmin {
		return nil, apperror.NewNotFound(domainGroup.ErrCodeGroupNotFound, "group not found")
	}

	memberCount, err := uc.memberRepo.CountMembers(ctx, g.ID())
	if err != nil {
		return nil, err
	}
	leadCount, err := uc.memberRepo.CountLeads(ctx, g.ID())
	if err != nil {
		return nil, err
	}

	leadMembers, err := uc.memberRepo.ListLeads(ctx, g.ID())
	if err != nil {
		return nil, err
	}

	leadIDs := make([]string, 0, len(leadMembers))
	for _, lm := range leadMembers {
		leadIDs = append(leadIDs, lm.UserID().Value())
	}
	displays, err := uc.userProvider.GetDisplays(ctx, leadIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch lead displays", "error", err, "group_id", g.ID())
		return nil, apperror.NewInternal()
	}

	leads := make([]LeadDisplay, 0, len(leadMembers))
	for _, lm := range leadMembers {
		id := lm.UserID().Value()
		d := displays[id]
		if d == nil {
			d = &UserDisplay{Nickname: "unknown", Name: ""}
		}
		leads = append(leads, LeadDisplay{UserID: id, Nickname: d.Nickname, Name: d.Name})
	}

	um := UserMembership{IsMember: membership != nil}
	if membership != nil {
		r := membership.Role()
		um.Role = &r
		ja := membership.JoinedAt().Format(timeutil.RFC3339UTC)
		um.JoinedAt = &ja
	}

	return &GetGroupOutput{
		Group:      g,
		Statistics: GroupStatistics{MemberCount: memberCount, LeadCount: leadCount},
		Leads:      leads,
		Membership: um,
	}, nil
}
