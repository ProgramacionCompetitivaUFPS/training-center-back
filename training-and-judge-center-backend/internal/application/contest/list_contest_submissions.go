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
	Type     string   // "INDIVIDUAL" or "TEAM"
	Nickname string   // for INDIVIDUAL
	Name     string   // for INDIVIDUAL
	TeamID   string   // for TEAM
	TeamName string   // for TEAM
	Members  []string // for TEAM, populated when contest.ShowTeamMembers=true
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
	submissionsProvider ContestSubmissionProvider
	standingProvider    CallerStandingProvider
}

func NewListContestSubmissionsUseCase(
	contestRepo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	participantProvider ContestParticipantProvider,
	submissionsProvider ContestSubmissionProvider,
	standingProvider CallerStandingProvider,
) *ListContestSubmissionsUseCase {
	return &ListContestSubmissionsUseCase{
		contestRepo:         contestRepo,
		groupProvider:       groupProvider,
		memberProvider:      memberProvider,
		participantProvider: participantProvider,
		submissionsProvider: submissionsProvider,
		standingProvider:    standingProvider,
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

	callerStandingID := in.CurrentUser.ID
	if !isAdmin && !isLead && freezeTime != nil {
		if sid, found, err := uc.standingProvider.GetCallerStandingID(ctx, in.ContestID, in.CurrentUser.ID); err == nil && found {
			callerStandingID = sid
		}
	}

	showFullDetails := status == domainContest.StatusFinished || isLead || isAdmin
	showMembers := contest.ShowTeamMembers()

	var result []SubmissionDisplay
	for _, sub := range subs {
		// Freeze: leads/admins see all; participants see own standing always + others pre-freeze only.
		if !isAdmin && !isLead && freezeTime != nil && sub.StandingID != callerStandingID {
			if !sub.SubmittedAt.Before(*freezeTime) {
				continue
			}
		}

		if in.Phase == "competition" && sub.SubmittedAt.After(endTime) {
			continue
		}
		if in.Phase == "postcompetition" && !sub.SubmittedAt.After(endTime) {
			continue
		}

		submitter := buildSubmitter(sub, showMembers)
		d := SubmissionDisplay{
			ID:          sub.ID,
			Problem:     SubmissionProblemDisplay{Slug: sub.ProblemSlug, Title: sub.ProblemTitle, Order: sub.ProblemOrder},
			SubmittedBy: submitter,
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

func buildSubmitter(sub RichSubmissionData, showMembers bool) SubmissionSubmitterDisplay {
	if sub.TeamID != nil {
		s := SubmissionSubmitterDisplay{
			Type:     "TEAM",
			TeamID:   *sub.TeamID,
			TeamName: *sub.TeamName,
		}
		if showMembers {
			s.Members = sub.TeamMemberNicknames
		}
		return s
	}
	return SubmissionSubmitterDisplay{
		Type:     "INDIVIDUAL",
		Nickname: sub.Nickname,
		Name:     sub.SubmitterName,
	}
}
