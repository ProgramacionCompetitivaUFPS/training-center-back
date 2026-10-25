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
	IsBlockly   bool
	Status      string
	SubmittedAt time.Time
	JudgedAt    *time.Time // nil during ACTIVE for participants
	TimeMs      *int       // nil during ACTIVE for participants
	MemoryKb    *int       // nil during ACTIVE for participants
	Phase       *string    // nil during ACTIVE for participants; "competition" or "postcompetition"
	// CanViewDetail mirrors GetSubmission's access rule: own or team submission,
	// Lead/Admin, or a submission its author made public.
	CanViewDetail bool
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
	submissionsProvider ContestSubmissionProvider
	standingProvider    CallerStandingProvider
}

func NewListContestSubmissionsUseCase(
	contestRepo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	submissionsProvider ContestSubmissionProvider,
	standingProvider CallerStandingProvider,
) *ListContestSubmissionsUseCase {
	return &ListContestSubmissionsUseCase{
		contestRepo:         contestRepo,
		groupProvider:       groupProvider,
		memberProvider:      memberProvider,
		submissionsProvider: submissionsProvider,
		standingProvider:    standingProvider,
	}
}

func (uc *ListContestSubmissionsUseCase) Execute(ctx context.Context, in ListContestSubmissionsInput) (*ListContestSubmissionsOutput, error) {
	contest, err := uc.contestRepo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if contest.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}

	isAdmin := in.CurrentUser.IsAdmin()
	isLead, isMember := false, false
	if !isAdmin {
		role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		isMember = role != nil
		isLead = role != nil && *role == "LEAD"
	}

	if !group.IsVisible && !isMember && !isLead && !isAdmin {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
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

	callerStandingID := in.CurrentUser.ID
	if !isAdmin && !isLead {
		if sid, found, err := uc.standingProvider.GetCallerStandingID(ctx, in.ContestID, in.CurrentUser.ID); err == nil && found {
			callerStandingID = sid
		}
	}

	filters := ContestSubmissionFilters{
		ProblemSlug: in.ProblemSlug,
		Nickname:    in.Nickname,
		Limit:       in.Limit,
		Offset:      (in.Page - 1) * in.Limit,
	}
	switch in.Phase {
	case "competition":
		filters.SubmittedAtMost = &endTime
	case "postcompetition":
		filters.SubmittedAfter = &endTime
	}
	// Freeze: leads/admins see all; participants see own standing always + others pre-freeze only.
	if !isAdmin && !isLead && freezeTime != nil {
		filters.FreezeAt = freezeTime
		filters.ExceptStandingID = callerStandingID
	}

	subs, total, err := uc.submissionsProvider.ListByContest(ctx, in.ContestID, filters)
	if err != nil {
		return nil, err
	}

	showFullDetails := status == domainContest.StatusFinished || isLead || isAdmin
	showMembers := contest.ShowTeamMembers()

	result := make([]SubmissionDisplay, 0, len(subs))
	for _, sub := range subs {
		submitter := buildSubmitter(sub, showMembers)
		d := SubmissionDisplay{
			ID:          sub.ID,
			Problem:     SubmissionProblemDisplay{Slug: sub.ProblemSlug, Title: sub.ProblemTitle, Order: sub.ProblemOrder},
			SubmittedBy: submitter,
			Language:    sub.Language,
			IsBlockly:   sub.IsBlockly,
			Status:      sub.Status,
			SubmittedAt: sub.SubmittedAt,
			CanViewDetail: isAdmin || isLead || sub.UserID == in.CurrentUser.ID ||
				sub.StandingID == callerStandingID || sub.Visibility == "PUBLIC",
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
		Submissions: result,
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
