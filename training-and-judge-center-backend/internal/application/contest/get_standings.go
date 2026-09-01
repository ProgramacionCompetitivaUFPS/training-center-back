package contest

import (
	"context"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

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
	Rank            int
	ContestantID    string
	ParticipantType string // "INDIVIDUAL" or "TEAM"
	ProblemsSolved  int
	TotalPenalty    int
	LastAcceptedAt  *time.Time
	Problems        map[string]RankedProblem // key: problemID
}

type GetStandingsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	Realtime    bool
	Country     string
	City        string
	Institution string
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
	contestRepo          domainContest.Repository
	registrationRepo     domainContest.RegistrationRepository
	submissionProvider   StandingsSubmissionProvider
	teamParticipProvider TeamParticipantProvider
	profileProvider      ParticipantProfileProvider
	groupProvider        GroupProvider
	memberProvider       GroupMemberProvider
	standingsCache       StandingsCache
	staleness            time.Duration
	lockTTL              time.Duration
}

func NewGetStandingsUseCase(
	contestRepo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	submissionProvider StandingsSubmissionProvider,
	teamParticipProvider TeamParticipantProvider,
	profileProvider ParticipantProfileProvider,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	standingsCache StandingsCache,
	staleness time.Duration,
) *GetStandingsUseCase {
	return &GetStandingsUseCase{
		contestRepo:          contestRepo,
		registrationRepo:     registrationRepo,
		submissionProvider:   submissionProvider,
		teamParticipProvider: teamParticipProvider,
		profileProvider:      profileProvider,
		groupProvider:        groupProvider,
		memberProvider:       memberProvider,
		standingsCache:       standingsCache,
		staleness:            staleness,
		lockTTL:              staleness * 2,
	}
}

func (uc *GetStandingsUseCase) Execute(ctx context.Context, in GetStandingsInput) (*GetStandingsOutput, error) {
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
	isMember, isLead := false, false
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
		return uc.buildOutput(cached, contest, applyFreeze, status, freezeTime, in), nil
	}

	rebuilt, err := uc.rebuild(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	return uc.buildOutput(rebuilt, contest, applyFreeze, status, freezeTime, in), nil
}

func (uc *GetStandingsUseCase) rebuild(ctx context.Context, contestID string) (*CachedStandings, error) {
	g, gctx := errgroup.WithContext(ctx)

	var regs []*domainContest.ContestRegistration
	var teamMembers map[string][]string
	var subs []ContestSubmissionData

	g.Go(func() error {
		var err error
		regs, _, err = uc.registrationRepo.ListByContest(gctx, contestID, 1, maxStandingsParticipants)
		return err
	})
	g.Go(func() error {
		var err error
		teamMembers, err = uc.teamParticipProvider.ListSelectedMembersByContest(gctx, contestID)
		return err
	})
	g.Go(func() error {
		var err error
		subs, err = uc.submissionProvider.ListByContest(gctx, contestID)
		return err
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// userToContestant maps submitter userID → contestantID.
	// For individual participants: userID → userID.
	// For team members: userID → teamID (all members submit under the team).
	// Teams are processed first so that in MIXED contests, team membership takes
	// precedence: a user who is both individually registered and a selected team
	// member has their submissions attributed to the team only.
	userToContestant := make(map[string]string, len(regs)+len(teamMembers)*3)
	standings := make(map[string]*domainContest.ParticipantStanding, len(regs)+len(teamMembers))

	for teamID, members := range teamMembers {
		standings[teamID] = &domainContest.ParticipantStanding{
			ContestantID:    teamID,
			ParticipantType: "TEAM",
			Problems:        make(map[string]domainContest.ProblemAttempt),
		}
		for _, memberID := range members {
			userToContestant[memberID] = teamID
		}
	}

	for _, reg := range regs {
		if _, onTeam := userToContestant[reg.UserID()]; onTeam {
			continue // MIXED: user is a selected team member; submissions count for the team
		}
		standings[reg.UserID()] = &domainContest.ParticipantStanding{
			ContestantID:    reg.UserID(),
			ParticipantType: "INDIVIDUAL",
			Problems:        make(map[string]domainContest.ProblemAttempt),
		}
		userToContestant[reg.UserID()] = reg.UserID()
	}

	for _, sub := range subs {
		contestantID, ok := userToContestant[sub.UserID]
		if !ok {
			continue
		}
		p := standings[contestantID]
		attempt := p.Problems[sub.ProblemID]
		if attempt.AcceptedAt != nil {
			continue // already solved; ignore later submissions
		}
		switch sub.Status {
		case "ACCEPTED":
			t := sub.SubmittedAt
			attempt.AcceptedAt = &t
			p.Problems[sub.ProblemID] = attempt
		// Literal strings rather than the domain type, so a verdict added there
		// does not reach this list on its own — see the test that pins them.
		case "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", "MEMORY_LIMIT_EXCEEDED",
			"OUTPUT_LIMIT_EXCEEDED", "RUNTIME_ERROR":
			attempt.WrongAttemptTimes = append(attempt.WrongAttemptTimes, sub.SubmittedAt)
			p.Problems[sub.ProblemID] = attempt
		}
		// COMPILATION_ERROR and SYSTEM_ERROR don't count as wrong attempts (ICPC rules)
	}

	participants := make([]domainContest.ParticipantStanding, 0, len(standings))
	for _, p := range standings {
		participants = append(participants, *p)
	}

	profileIDs := make([]string, 0, len(userToContestant))
	for userID := range userToContestant {
		profileIDs = append(profileIDs, userID)
	}
	profiles, err := uc.profileProvider.GetProfiles(ctx, profileIDs)
	if err != nil {
		slog.WarnContext(ctx, "rebuild: profile enrichment degraded", "error", err)
		profiles = map[string]*ParticipantProfile{}
	}

	cached := &CachedStandings{
		Participants: participants,
		TeamMembers:  teamMembers,
		Profiles:     profiles,
		LastUpdated:  time.Now(),
	}
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
	in GetStandingsInput,
) *GetStandingsOutput {
	var effectiveFreezeTime *time.Time
	if applyFreeze {
		effectiveFreezeTime = freezeTime
	}

	participants := FilterStandingsByProfile(cached, in.Country, in.City, in.Institution)
	entries := RankStandings(participants, contest.StartTime(), contest.Penalty().Value(), effectiveFreezeTime)

	total := len(entries)
	start := (in.Page - 1) * in.Limit
	if start > total {
		start = total
	}
	end := start + in.Limit
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
		Page:    in.Page,
		Limit:   in.Limit,
		Meta:    meta,
	}
}
