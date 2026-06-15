package contest

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListContestSubmissionsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	ProblemSlug *string
	Nickname    *string
	Phase       string // "competition" | "postcompetition" | "" (all)
	Page        int
	Limit       int
}

type SubmissionProblemDisplay struct {
	Slug  string
	Title string
	Order int
}

type SubmissionSubmitterDisplay struct {
	Nickname string
}

type SubmissionDisplay struct {
	ID          string
	Problem     SubmissionProblemDisplay
	SubmittedBy SubmissionSubmitterDisplay
	Language    string
	Status      string
	SubmittedAt time.Time
	JudgedAt    *time.Time // nil during ACTIVE for participants
	TimeMs      *int       // nil during ACTIVE for participants
	MemoryKb    *int       // nil during ACTIVE for participants
	Phase       *string    // nil during ACTIVE for participants; "competition" or "postcompetition"
}

type SubmissionsContestMeta struct {
	ID         string
	Name       string
	Status     string
	InFreeze   bool
	FreezeTime *time.Time
}

type ListContestSubmissionsOutput struct {
	Submissions []SubmissionDisplay
	Total       int
	Page        int
	Limit       int
	Meta        SubmissionsContestMeta
}

type ListContestSubmissionsUseCase struct {
	contestRepo         domainContest.Repository
	groupProvider       GroupProvider
	memberProvider      GroupMemberProvider
	participantProvider ContestParticipantProvider
	submissionsProvider ContestSubmissionsProvider
}

func NewListContestSubmissionsUseCase(
	contestRepo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	participantProvider ContestParticipantProvider,
	submissionsProvider ContestSubmissionsProvider,
) *ListContestSubmissionsUseCase {
	return &ListContestSubmissionsUseCase{
		contestRepo:         contestRepo,
		groupProvider:       groupProvider,
		memberProvider:      memberProvider,
		participantProvider: participantProvider,
		submissionsProvider: submissionsProvider,
	}
}

func (uc *ListContestSubmissionsUseCase) Execute(ctx context.Context, in ListContestSubmissionsInput) (*ListContestSubmissionsOutput, error) {
	contest, err := uc.contestRepo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if contest == nil || contest.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	isAdmin := in.CurrentUser.IsAdmin()
	isLead, isMember := false, false
	if !isAdmin {
		isMember, err = uc.memberProvider.IsMemberOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		isLead, err = uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
	}

	if !group.IsVisible && !isMember && !isLead && !isAdmin {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	// Leads and admins bypass the registration requirement.
	if !isAdmin && !isLead {
		registered, err := uc.participantProvider.IsRegistered(ctx, in.ContestID, in.CurrentUser.ID)
		if err != nil {
			return nil, err
		}
		if !registered {
			return nil, apperror.NewForbidden(ErrCodeNotRegistered, "you must be registered to view contest submissions")
		}
	}

	if in.Page < 1 {
		in.Page = 1
	}

	now := time.Now()
	status := contest.Status(now)
	endTime := contest.EndTime()

	var freezeTime *time.Time
	if contest.FreezeMinutes() > 0 && status == domainContest.StatusActive {
		ft := endTime.Add(-time.Duration(contest.FreezeMinutes()) * time.Minute)
		if now.After(ft) {
			freezeTime = &ft
		}
	}

	subs, err := uc.submissionsProvider.ListByContest(ctx, in.ContestID, ContestSubmissionFilters{
		ProblemSlug: in.ProblemSlug,
		Nickname:    in.Nickname,
	})
	if err != nil {
		return nil, err
	}

	callerID := in.CurrentUser.ID
	showFullDetails := status == domainContest.StatusFinished || isLead || isAdmin
	postContestEnabled := contest.EnablePostContest()

	var result []SubmissionDisplay
	for _, sub := range subs {
		// Freeze: leads/admins see all; participants see own always + others pre-freeze only.
		if !isAdmin && !isLead && freezeTime != nil && sub.UserID != callerID {
			if !sub.SubmittedAt.Before(*freezeTime) {
				continue
			}
		}

		// Post-contest submissions hidden when the feature is disabled.
		if !isAdmin && !isLead && !postContestEnabled && sub.SubmittedAt.After(endTime) {
			continue
		}

		// Phase filter.
		if in.Phase == "competition" && sub.SubmittedAt.After(endTime) {
			continue
		}
		if in.Phase == "postcompetition" && !sub.SubmittedAt.After(endTime) {
			continue
		}

		d := SubmissionDisplay{
			ID:          sub.ID,
			Problem:     SubmissionProblemDisplay{Slug: sub.ProblemSlug, Title: sub.ProblemTitle, Order: sub.ProblemOrder},
			SubmittedBy: SubmissionSubmitterDisplay{Nickname: sub.Nickname},
			Language:    sub.Language,
			Status:      sub.Status,
			SubmittedAt: sub.SubmittedAt,
		}

		if showFullDetails {
			d.JudgedAt = sub.JudgedAt
			d.TimeMs = sub.TimeMs
			d.MemoryKb = sub.MemoryKb
			phase := "competition"
			if sub.SubmittedAt.After(endTime) {
				phase = "postcompetition"
			}
			d.Phase = &phase
		}

		result = append(result, d)
	}

	total := len(result)
	start := (in.Page - 1) * in.Limit
	if start > total {
		start = total
	}
	end := start + in.Limit
	if end > total {
		end = total
	}

	meta := SubmissionsContestMeta{
		ID:     contest.ID(),
		Name:   contest.Name().Value(),
		Status: status.String(),
	}
	if freezeTime != nil {
		meta.InFreeze = true
		meta.FreezeTime = freezeTime
	}

	return &ListContestSubmissionsOutput{
		Submissions: result[start:end],
		Total:       total,
		Page:        in.Page,
		Limit:       in.Limit,
		Meta:        meta,
	}, nil
}
