package contest

import (
	"context"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type CreateContestInput struct {
	CurrentUser       appshared.CurrentUser
	GroupID           string
	Name              string
	Description       *string
	StartTime         time.Time
	EndTime           time.Time
	Penalty           *int
	FreezeMinutes     *int
	EnablePostContest bool
	ParticipationMode string // "INDIVIDUAL" | "TEAM" | "MIXED"; defaults to "INDIVIDUAL"
	TeamSizeMin       *int   // defaults to 2 when mode allows teams
	TeamSizeMax       *int   // defaults to 5 when mode allows teams
	Problems          []string // slugs; may be empty
}

type CreateContestUseCase struct {
	repo            domainContest.Repository
	groupProvider   GroupProvider
	memberProvider  GroupMemberProvider
	problemProvider ProblemProvider
	ownerProvider   OwnerProvider
}

func NewCreateContestUseCase(
	repo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	problemProvider ProblemProvider,
	ownerProvider OwnerProvider,
) *CreateContestUseCase {
	return &CreateContestUseCase{
		repo:            repo,
		groupProvider:   groupProvider,
		memberProvider:  memberProvider,
		problemProvider: problemProvider,
		ownerProvider:   ownerProvider,
	}
}

func (uc *CreateContestUseCase) Execute(ctx context.Context, in CreateContestInput) (*ContestOutput, error) {
	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		return nil, apperror.NewNotFound(ErrCodeGroupNotFound, "group not found")
	}

	if !in.CurrentUser.IsAdmin() {
		role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if role == nil || *role != "LEAD" {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only group leads can create contests")
		}
	}

	var fieldErrs []apperror.FieldError

	name, nameErr := domainContest.NewContestName(in.Name)
	if err := apperror.AccumulateFieldErrors(nameErr, &fieldErrs); err != nil {
		return nil, err
	}

	if in.Description != nil && len([]rune(*in.Description)) > domainContest.MaxDescriptionLength {
		fieldErrs = append(fieldErrs, apperror.FieldError{
			Field:   "description",
			Message: "description cannot exceed 5000 characters",
		})
	}

	penaltyVal := 20
	if in.Penalty != nil {
		penaltyVal = *in.Penalty
	}
	penalty, penaltyErr := domainContest.NewPenalty(penaltyVal)
	if err := apperror.AccumulateFieldErrors(penaltyErr, &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	var freezeMinutes int
	if in.FreezeMinutes != nil {
		freezeMinutes = *in.FreezeMinutes
	} else if in.EndTime.Sub(in.StartTime) >= 4*time.Hour {
		freezeMinutes = 60
	}

	// Resolve and validate problems.
	type problemEntry struct {
		id    string
		slug  string
		title string
	}
	var problemEntries []problemEntry

	if len(in.Problems) > 0 {
		deduped := deduplicateSlugs(in.Problems)
		infos, err := uc.problemProvider.FindBySlugs(ctx, deduped, in.CurrentUser.ID, in.CurrentUser.IsAdmin())
		if err != nil {
			return nil, err
		}
		for _, slug := range deduped {
			info, ok := infos[slug]
			if !ok {
				return nil, apperror.NewNotFound(ErrCodeProblemNotFound, "problem '"+slug+"' not found")
			}
			if !info.IsPublished {
				return nil, apperror.NewBadRequest(ErrCodeProblemNotPublished, "problem '"+slug+"' is not published")
			}
			if !info.CanAdd {
				return nil, apperror.NewForbidden(ErrCodeProblemAccessDenied,
					"cannot add private problem '"+slug+"' — you are not a modifier")
			}
			problemEntries = append(problemEntries, problemEntry{id: info.ID, slug: slug, title: info.Title})
		}
	}

	// Participation mode and team size.
	modeRaw := in.ParticipationMode
	if modeRaw == "" {
		modeRaw = "INDIVIDUAL"
	}
	mode, modeErr := domainContest.NewParticipationMode(modeRaw)
	if err := apperror.AccumulateFieldErrors(modeErr, &fieldErrs); err != nil {
		return nil, err
	}

	teamSizeMinVal := 2
	if in.TeamSizeMin != nil {
		teamSizeMinVal = *in.TeamSizeMin
	}
	teamSizeMaxVal := 5
	if in.TeamSizeMax != nil {
		teamSizeMaxVal = *in.TeamSizeMax
	}
	teamSize, tsErr := domainContest.NewTeamSize(teamSizeMinVal, teamSizeMaxVal)
	if err := apperror.AccumulateFieldErrors(tsErr, &fieldErrs); err != nil {
		return nil, err
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	now := time.Now()
	newID := uuid.New().String()

	c, err := domainContest.NewContest(
		newID,
		name,
		in.Description,
		in.StartTime,
		in.EndTime,
		penalty,
		freezeMinutes,
		in.EnablePostContest,
		shared.RestoreGroupID(in.GroupID),
		shared.RestoreUserID(in.CurrentUser.ID),
		mode,
		teamSize,
		now,
	)
	if err != nil {
		return nil, err
	}

	for _, pe := range problemEntries {
		if addErr := c.AddProblem(uuid.New().String(), pe.id, now); addErr != nil {
			return nil, addErr
		}
	}

	if err := uc.repo.Create(ctx, c); err != nil {
		return nil, err
	}

	owner, err := uc.ownerProvider.GetDisplay(ctx, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}
	if owner == nil {
		owner = &UserDisplay{}
	}

	// Build problem display from already-resolved entries (order = insertion order).
	problemDisplays := make([]ProblemDisplay, 0, len(problemEntries))
	for i, pe := range problemEntries {
		problemDisplays = append(problemDisplays, ProblemDisplay{
			Slug:  pe.slug,
			Title: pe.title,
			Order: i + 1,
		})
	}

	return buildOutput(c, group, owner, problemDisplays, now), nil
}

func deduplicateSlugs(slugs []string) []string {
	seen := make(map[string]struct{}, len(slugs))
	out := make([]string, 0, len(slugs))
	for _, s := range slugs {
		if _, ok := seen[s]; !ok {
			seen[s] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
