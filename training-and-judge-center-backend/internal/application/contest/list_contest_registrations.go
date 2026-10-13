package contest

import (
	"context"
	"sort"
	"strings"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListContestRegistrationsInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	Page        int
	Limit       int
	// Search filters by a case-insensitive substring of the nickname (INDIVIDUAL)
	// or of the team name or any member nickname (TEAM). Empty means no filter.
	Search string
}

// RegistrationEntry is one registered participant: an individual or a team.
// A team is always a single entry, regardless of its roster size — it
// counts as one participant, matching how standings rank it.
type RegistrationEntry struct {
	Type         string // "INDIVIDUAL" or "TEAM"
	Nickname     string // INDIVIDUAL only
	TeamName     string // TEAM only
	Members      []string // TEAM only; member nicknames
	RegisteredAt time.Time
}

type ListContestRegistrationsOutput struct {
	Registrations []RegistrationEntry
	Total         int
	Page          int
	Limit         int
}

type ListContestRegistrationsUseCase struct {
	repo                domainContest.Repository
	registrationRepo    domainContest.RegistrationRepository
	teamParticipRepo    domainContest.TeamParticipationRepository
	groupProvider       GroupProvider
	memberProvider      GroupMemberProvider
	nicknameProvider    ParticipantNicknameProvider
	teamDisplayProvider TeamDisplayProvider
}

func NewListContestRegistrationsUseCase(
	repo domainContest.Repository,
	registrationRepo domainContest.RegistrationRepository,
	teamParticipRepo domainContest.TeamParticipationRepository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	nicknameProvider ParticipantNicknameProvider,
	teamDisplayProvider TeamDisplayProvider,
) *ListContestRegistrationsUseCase {
	return &ListContestRegistrationsUseCase{
		repo:                repo,
		registrationRepo:    registrationRepo,
		teamParticipRepo:    teamParticipRepo,
		groupProvider:       groupProvider,
		memberProvider:      memberProvider,
		nicknameProvider:    nicknameProvider,
		teamDisplayProvider: teamDisplayProvider,
	}
}

func (uc *ListContestRegistrationsUseCase) Execute(ctx context.Context, in ListContestRegistrationsInput) (*ListContestRegistrationsOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	// Whoever can see the contest can see who is in it: anyone for a visible
	// group, only members (or Admin) for a hidden one.
	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if !group.IsVisible && !in.CurrentUser.IsAdmin() {
		role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if role == nil {
			return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
		}
	}

	regs, _, err := uc.registrationRepo.ListByContest(ctx, in.ContestID, 1, maxStandingsParticipants)
	if err != nil {
		return nil, err
	}
	teamParticipations, _, err := uc.teamParticipRepo.List(ctx, in.ContestID, 1, maxStandingsParticipants)
	if err != nil {
		return nil, err
	}

	userIDs := make([]string, 0, len(regs))
	for _, r := range regs {
		userIDs = append(userIDs, r.UserID())
	}
	for _, p := range teamParticipations {
		userIDs = append(userIDs, p.SelectedMembers()...)
	}
	nicknames, err := uc.nicknameProvider.GetNicknamesByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	teamIDs := make([]string, len(teamParticipations))
	for i, p := range teamParticipations {
		teamIDs[i] = p.TeamID()
	}
	teamDisplays, err := uc.teamDisplayProvider.GetDisplays(ctx, teamIDs)
	if err != nil {
		return nil, err
	}

	entries := make([]RegistrationEntry, 0, len(regs)+len(teamParticipations))
	for _, r := range regs {
		entries = append(entries, RegistrationEntry{
			Type:         "INDIVIDUAL",
			Nickname:     nicknames[r.UserID()],
			RegisteredAt: r.RegisteredAt(),
		})
	}
	for _, p := range teamParticipations {
		members := make([]string, 0, len(p.SelectedMembers()))
		for _, uid := range p.SelectedMembers() {
			members = append(members, nicknames[uid])
		}
		teamName := ""
		if d := teamDisplays[p.TeamID()]; d != nil {
			teamName = d.Name
		}
		entries = append(entries, RegistrationEntry{
			Type:         "TEAM",
			TeamName:     teamName,
			Members:      members,
			RegisteredAt: p.RegisteredAt(),
		})
	}

	if q := strings.ToLower(strings.TrimSpace(in.Search)); q != "" {
		filtered := entries[:0]
		for _, e := range entries {
			if e.matches(q) {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].RegisteredAt.Before(entries[j].RegisteredAt)
	})

	total := len(entries)
	start := (in.Page - 1) * in.Limit
	if start > total {
		start = total
	}
	end := start + in.Limit
	if end > total {
		end = total
	}

	return &ListContestRegistrationsOutput{
		Registrations: entries[start:end],
		Total:         total,
		Page:          in.Page,
		Limit:         in.Limit,
	}, nil
}

// matches reports whether q (already lowercased) is a substring of the entry's
// nickname, team name or any of its member nicknames.
func (e RegistrationEntry) matches(q string) bool {
	if strings.Contains(strings.ToLower(e.Nickname), q) || strings.Contains(strings.ToLower(e.TeamName), q) {
		return true
	}
	for _, m := range e.Members {
		if strings.Contains(strings.ToLower(m), q) {
			return true
		}
	}
	return false
}
