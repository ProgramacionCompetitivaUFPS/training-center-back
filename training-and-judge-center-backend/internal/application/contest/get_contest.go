package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetContestInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
}

type ProblemDetailDisplay struct {
	Position    int // maps to the domain's Order field
	Slug        string
	Title       string
	TimeLimit   int
	MemoryLimit int
}

type GetContestOutput struct {
	ID                string
	Name              string
	Description       *string
	StartTime         time.Time
	EndTime           time.Time
	Duration          int // in seconds (spec FR-010)
	Penalty           int
	FreezeMinutes     int
	EnablePostContest bool
	Locked            *bool // nil when caller is not Lead or Admin (omitted from JSON)
	ParticipantCount  int
	IsRegistered      bool
	Group             GroupDisplay
	Owner             UserDisplay
	Problems          []ProblemDetailDisplay
	Status            string
	CreatedAt         time.Time
	UpdatedAt         *time.Time
}

type GetContestUseCase struct {
	repo                domainContest.Repository
	groupProvider       GroupProvider
	memberProvider      GroupMemberProvider
	problemProvider     ProblemProvider
	ownerProvider       OwnerProvider
	participantProvider ContestParticipantProvider
}

func NewGetContestUseCase(
	repo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	problemProvider ProblemProvider,
	ownerProvider OwnerProvider,
	participantProvider ContestParticipantProvider,
) *GetContestUseCase {
	return &GetContestUseCase{
		repo:                repo,
		groupProvider:       groupProvider,
		memberProvider:      memberProvider,
		problemProvider:     problemProvider,
		ownerProvider:       ownerProvider,
		participantProvider: participantProvider,
	}
}

func (uc *GetContestUseCase) Execute(ctx context.Context, in GetContestInput) (*GetContestOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	isAdmin := in.CurrentUser.IsAdmin()

	isMember, isLead := false, false
	if !isAdmin {
		isMember, err = uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		isLead, err = uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if isLead {
			isMember = true
		}
	}

	if !group.IsVisible && !isMember && !isAdmin {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	now := time.Now()
	status := c.Status(now)

	participantCount, err := uc.participantProvider.CountParticipants(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	isRegistered, err := uc.participantProvider.IsRegistered(ctx, in.ContestID, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}

	showProblems := isAdmin || (isMember && status != domainContest.StatusScheduled)

	problemDisplays := []ProblemDetailDisplay{}
	if showProblems && len(c.Problems()) > 0 {
		ids := make([]string, len(c.Problems()))
		for i, cp := range c.Problems() {
			ids[i] = cp.ProblemID()
		}
		infos, err := uc.problemProvider.FindByIDsWithLimits(ctx, ids)
		if err != nil {
			return nil, err
		}
		problemDisplays = make([]ProblemDetailDisplay, 0, len(c.Problems()))
		for _, cp := range c.Problems() {
			info, ok := infos[cp.ProblemID()]
			if !ok {
				continue
			}
			problemDisplays = append(problemDisplays, ProblemDetailDisplay{
				Position:    cp.Order(),
				Slug:        info.Slug,
				Title:       info.Title,
				TimeLimit:   info.TimeLimit,
				MemoryLimit: info.MemoryLimit,
			})
		}
	}

	owner, err := uc.ownerProvider.GetDisplay(ctx, c.OwnerID().Value())
	if err != nil {
		return nil, err
	}
	if owner == nil {
		owner = &UserDisplay{}
	}

	var locked *bool
	if isAdmin || isLead {
		v := c.Locked()
		locked = &v
	}

	return &GetContestOutput{
		ID:                c.ID(),
		Name:              c.Name().Value(),
		Description:       c.Description(),
		StartTime:         c.StartTime(),
		EndTime:           c.EndTime(),
		Duration:          c.Duration() * 60,
		Penalty:           c.Penalty().Value(),
		FreezeMinutes:     c.FreezeMinutes(),
		EnablePostContest: c.EnablePostContest(),
		Locked:            locked,
		ParticipantCount:  participantCount,
		IsRegistered:      isRegistered,
		Group:             GroupDisplay{ID: group.ID, Name: group.Name},
		Owner:             *owner,
		Problems:          problemDisplays,
		Status:            status.String(),
		CreatedAt:         c.CreatedAt(),
		UpdatedAt:         c.UpdatedAt(),
	}, nil
}
