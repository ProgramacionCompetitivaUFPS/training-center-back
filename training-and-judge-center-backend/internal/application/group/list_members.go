package group

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListMembersInput struct {
	GroupID     string
	Page        int
	Limit       int
	Role        *string
	CurrentUser appshared.CurrentUser
}

type MemberItemDTO struct {
	UserID   string
	Nickname string
	Name     string
	Role     string
	JoinedAt time.Time
}

type ListMembersOutput struct {
	Members    []MemberItemDTO
	TotalCount int
	TotalPages int
}

type ListMembersUseCase struct {
	groupRepo    domainGroup.Repository
	memberRepo   domainGroup.MemberRepository
	userProvider UserProvider
}

func NewListMembersUseCase(
	groupRepo domainGroup.Repository,
	memberRepo domainGroup.MemberRepository,
	userProvider UserProvider,
) *ListMembersUseCase {
	return &ListMembersUseCase{
		groupRepo:    groupRepo,
		memberRepo:   memberRepo,
		userProvider: userProvider,
	}
}

func (uc *ListMembersUseCase) Execute(ctx context.Context, input ListMembersInput) (*ListMembersOutput, error) {
	g, err := uc.groupRepo.FindByID(ctx, input.GroupID)
	if err != nil {
		return nil, err
	}

	if !g.IsDefault() && !input.CurrentUser.IsAdmin() {
		caller, err := uc.memberRepo.FindByGroupAndUser(ctx, input.GroupID, shared.RestoreUserID(input.CurrentUser.ID))
		if err != nil {
			return nil, err
		}
		if caller == nil {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only group members can list members")
		}
	}

	filters := domainGroup.MemberFilters{
		Page:  input.Page,
		Limit: input.Limit,
	}
	if input.Role != nil {
		r, err := domainGroup.NewMemberRole(*input.Role)
		if err != nil {
			return nil, err
		}
		filters.Role = &r
	}

	members, total, err := uc.memberRepo.FindByGroup(ctx, input.GroupID, filters)
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, len(members))
	for i, m := range members {
		userIDs[i] = m.UserID().Value()
	}

	displays, err := uc.userProvider.GetDisplays(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	items := make([]MemberItemDTO, 0, len(members))
	for _, m := range members {
		uid := m.UserID().Value()
		item := MemberItemDTO{
			UserID:   uid,
			Role:     m.Role().String(),
			JoinedAt: m.JoinedAt(),
		}
		if d := displays[uid]; d != nil {
			item.Nickname = d.Nickname
			item.Name = d.Name
		}
		items = append(items, item)
	}

	effectiveLimit := filters.Limit
	if effectiveLimit <= 0 {
		effectiveLimit = 20
	}
	totalPages := total / effectiveLimit
	if total%effectiveLimit != 0 {
		totalPages++
	}
	if totalPages == 0 {
		totalPages = 1
	}

	return &ListMembersOutput{Members: items, TotalCount: total, TotalPages: totalPages}, nil
}
