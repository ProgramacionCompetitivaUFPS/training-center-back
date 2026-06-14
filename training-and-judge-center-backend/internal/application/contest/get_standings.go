package contest

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const maxStandingsParticipants = 10_000

// RankedProblem is the per-problem view returned in a RankedEntry.
type RankedProblem struct {
	Attempts   int
	AcceptedAt *time.Time // nil if not solved (or solved after freeze, which is hidden)
	Penalty    int        // 0 if not solved
}

// RankedEntry is one row in the standings table.
type RankedEntry struct {
	Rank           int
	ContestantID   string
	ProblemsSolved int
	TotalPenalty   int
	LastAcceptedAt *time.Time
	Problems       map[string]RankedProblem // key: problemID
}

type GetStandingsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	Realtime    bool
	Page        int
	Limit       int
}

type StandingsMeta struct {
	LastUpdated   time.Time
	IsFrozen      bool
	FrozenAt      *time.Time
	ContestStatus string
}

type GetStandingsOutput struct {
	Entries []RankedEntry
	Total   int
	Page    int
	Limit   int
	Meta    StandingsMeta
}

type GetStandingsUseCase struct {
	contestRepo        domainContest.Repository
	registrationRepo   domainContest.RegistrationRepository
	submissionProvider StandingsSubmissionProvider
	groupProvider      GroupProvider
	memberProvider     GroupMemberProvider
	standingsCache     StandingsCache
	staleness          time.Duration
	lockTTL            time.Duration
}

func NewGetStandingsUseCase(
	contestRepo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	submissionProvider StandingsSubmissionProvider,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	standingsCache StandingsCache,
	staleness time.Duration,
) *GetStandingsUseCase {
	return &GetStandingsUseCase{
		contestRepo:        contestRepo,
		registrationRepo:   registrationRepo,
		submissionProvider: submissionProvider,
		groupProvider:      groupProvider,
		memberProvider:     memberProvider,
		standingsCache:     standingsCache,
		staleness:          staleness,
		lockTTL:            staleness * 2,
	}
}

func (uc *GetStandingsUseCase) Execute(ctx context.Context, in GetStandingsInput) (*GetStandingsOutput, error) {
	contest, err := uc.contestRepo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if contest == nil {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}
	if contest.GroupID().Value() != in.GroupID {
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
	}

	if !group.IsVisible && !isMember && !isLead && !isAdmin {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	now := time.Now()
	status := contest.Status(now)

	var freezeTime *time.Time
	if contest.FreezeMinutes() > 0 && status == domainContest.StatusActive {
		ft := contest.EndTime().Add(-time.Duration(contest.FreezeMinutes()) * time.Minute)
		if now.After(ft) {
			freezeTime = &ft
		}
	}

	applyFreeze := freezeTime != nil
	if in.Realtime && applyFreeze {
		if !isAdmin && !isLead {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only leads and admins can view real-time standings during freeze")
		}
		applyFreeze = false
	}

	cached, err := uc.standingsCache.Get(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}

	if cached != nil {
		if time.Since(cached.LastUpdated) > uc.staleness {
			go uc.backgroundRefresh(in.ContestID)
		}
		return uc.buildOutput(cached, contest, applyFreeze, status, freezeTime, in.Page, in.Limit), nil
	}

	rebuilt, err := uc.rebuild(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	return uc.buildOutput(rebuilt, contest, applyFreeze, status, freezeTime, in.Page, in.Limit), nil
}

func (uc *GetStandingsUseCase) rebuild(ctx context.Context, contestID string) (*CachedStandings, error) {
	regs, _, err := uc.registrationRepo.ListByContest(ctx, contestID, 1, maxStandingsParticipants)
	if err != nil {
		return nil, err
	}

	subs, err := uc.submissionProvider.ListByContest(ctx, contestID)
	if err != nil {
		return nil, err
	}

	standings := make(map[string]*domainContest.ParticipantStanding, len(regs))
	for _, reg := range regs {
		standings[reg.UserID()] = &domainContest.ParticipantStanding{
			ContestantID: reg.UserID(),
			Problems:     make(map[string]domainContest.ProblemAttempt),
		}
	}

	for _, sub := range subs {
		p, ok := standings[sub.UserID]
		if !ok {
			continue
		}
		attempt := p.Problems[sub.ProblemID]
		if attempt.AcceptedAt != nil {
			continue // already solved; ignore later submissions
		}
		switch sub.Status {
		case "ACCEPTED":
			t := sub.SubmittedAt
			attempt.AcceptedAt = &t
			p.Problems[sub.ProblemID] = attempt
		case "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", "MEMORY_LIMIT_EXCEEDED", "RUNTIME_ERROR":
			attempt.WrongAttemptTimes = append(attempt.WrongAttemptTimes, sub.SubmittedAt)
			p.Problems[sub.ProblemID] = attempt
		}
		// COMPILATION_ERROR and SYSTEM_ERROR don't count as wrong attempts (ICPC rules)
	}

	participants := make([]domainContest.ParticipantStanding, 0, len(standings))
	for _, p := range standings {
		participants = append(participants, *p)
	}

	cached := &CachedStandings{Participants: participants, LastUpdated: time.Now()}
	if err := uc.standingsCache.Set(ctx, contestID, cached); err != nil {
		slog.ErrorContext(ctx, "standings cache set failed after rebuild", "contest_id", contestID, "error", err)
	}
	return cached, nil
}

func (uc *GetStandingsUseCase) backgroundRefresh(contestID string) {
	ctx := context.Background()
	acquired, err := uc.standingsCache.AcquireRefreshLock(ctx, contestID, uc.lockTTL)
	if err != nil || !acquired {
		return
	}
	defer uc.standingsCache.ReleaseRefreshLock(ctx, contestID)

	if _, err := uc.rebuild(ctx, contestID); err != nil {
		slog.ErrorContext(ctx, "standings background refresh failed", "contest_id", contestID, "error", err)
	}
}

func (uc *GetStandingsUseCase) buildOutput(
	cached *CachedStandings,
	contest *domainContest.Contest,
	applyFreeze bool,
	status domainContest.Status,
	freezeTime *time.Time,
	page, limit int,
) *GetStandingsOutput {
	var effectiveFreezeTime *time.Time
	if applyFreeze {
		effectiveFreezeTime = freezeTime
	}

	entries := RankStandings(cached.Participants, contest.StartTime(), contest.Penalty().Value(), effectiveFreezeTime)

	total := len(entries)
	start := (page - 1) * limit
	if start > total {
		start = total
	}
	end := start + limit
	if end > total {
		end = total
	}

	meta := StandingsMeta{
		LastUpdated:   cached.LastUpdated,
		ContestStatus: status.String(),
	}
	if freezeTime != nil {
		meta.IsFrozen = true
		meta.FrozenAt = freezeTime
	}

	return &GetStandingsOutput{
		Entries: entries[start:end],
		Total:   total,
		Page:    page,
		Limit:   limit,
		Meta:    meta,
	}
}
