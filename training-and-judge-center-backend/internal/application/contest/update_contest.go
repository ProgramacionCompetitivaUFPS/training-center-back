package contest

import (
	"context"

	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// UpdateContestInput uses pointer fields so the caller can distinguish
// "not provided" (nil) from "explicitly set to zero/false/empty".
type UpdateContestInput struct {
	CurrentUser       appshared.CurrentUser
	GroupID           string
	ContestID         string
	Name              *string
	Description       **string // nil = no change; non-nil pointer-to-nil = clear
	StartTime         *time.Time
	EndTime           *time.Time
	Penalty           *int
	FreezeMinutes     *int
	EnablePostContest *bool
	// Problems, when non-nil, replaces the entire problem list.
	// Each entry is (slug, order).
	Problems          *[]ProblemOrderInput
	Locked            *bool
	ParticipationMode *string // "INDIVIDUAL" | "TEAM" | "MIXED"
	TeamSizeMin       *int
	TeamSizeMax       *int
	ShowTeamMembers   *bool
}

type ProblemOrderInput struct {
	Slug  string
	Order int
}

type problemEntry struct {
	id    string
	slug  string
	title string
	order int
}

type scalarFields struct {
	name     *domainContest.ContestName
	penalty  *domainContest.Penalty
	mode     *domainContest.ParticipationMode
	teamSize *domainContest.TeamSize
}

type UpdateContestUseCase struct {
	repo             domainContest.Repository
	groupProvider    GroupProvider
	memberProvider   GroupMemberProvider
	problemProvider  ProblemProvider
	ownerProvider    OwnerProvider
	teamParticipRepo domainContest.TeamParticipationRepository
	txManager        appshared.TransactionManager
}

func NewUpdateContestUseCase(
	repo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	problemProvider ProblemProvider,
	ownerProvider OwnerProvider,
	teamParticipRepo domainContest.TeamParticipationRepository,
	txManager appshared.TransactionManager,
) *UpdateContestUseCase {
	return &UpdateContestUseCase{
		repo:             repo,
		groupProvider:    groupProvider,
		memberProvider:   memberProvider,
		problemProvider:  problemProvider,
		ownerProvider:    ownerProvider,
		teamParticipRepo: teamParticipRepo,
		txManager:        txManager,
	}
}

func (uc *UpdateContestUseCase) Execute(ctx context.Context, in UpdateContestInput) (*ContestOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found in this group")
	}

	if !in.CurrentUser.IsAdmin() {
		role, err := uc.memberProvider.GetMemberRole(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			return nil, err
		}
		if role == nil || *role != "LEAD" {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only group leads can update contests")
		}
	}

	isOwner := c.OwnerID().Value() == in.CurrentUser.ID

	if in.Locked != nil && !isOwner && !in.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeOnlyOwnerOrAdmin, "only the contest owner or admin can lock or unlock the contest")
	}
	if c.Locked() && !isOwner && !in.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(domainContest.ErrCodeContestLocked, "contest is locked and cannot be modified")
	}

	if !hasAnyField(in) {
		return nil, apperror.NewBadRequest(ErrCodeNoFieldsToUpdate, "at least one field must be provided for update")
	}

	scalars, err := validateScalarFields(in, c)
	if err != nil {
		return nil, err
	}

	problemsChanged := in.Problems != nil
	var newProblemEntries []problemEntry
	if problemsChanged {
		newProblemEntries, err = uc.resolveProblems(ctx, *in.Problems, in.CurrentUser.ID, in.CurrentUser.IsAdmin())
		if err != nil {
			return nil, err
		}
	}

	now := time.Now()

	if err := c.Update(scalars.name, in.Description, in.StartTime, in.EndTime, scalars.penalty, in.FreezeMinutes, in.EnablePostContest, now); err != nil {
		return nil, err
	}

	if scalars.mode != nil || scalars.teamSize != nil {
		_, total, err := uc.teamParticipRepo.List(ctx, in.ContestID, 1, 1)
		if err != nil {
			return nil, err
		}
		if total > 0 {
			return nil, apperror.NewConflict(domainContest.ErrCodeContestHasTeamRegistrations,
				"cannot change participation settings while team registrations exist")
		}
	}
	if scalars.mode != nil {
		c.SetParticipationMode(*scalars.mode, now)
	}
	if scalars.teamSize != nil {
		c.SetTeamSize(*scalars.teamSize, now)
	}

	if problemsChanged {
		existingByProblemID := make(map[string]domainContest.ContestProblem, len(c.Problems()))
		for _, cp := range c.Problems() {
			existingByProblemID[cp.ProblemID()] = cp
		}
		newProblems := make([]domainContest.ContestProblem, 0, len(newProblemEntries))
		for _, pe := range newProblemEntries {
			if existing, ok := existingByProblemID[pe.id]; ok {
				newProblems = append(newProblems, domainContest.RestoreContestProblem(existing.ID(), pe.id, pe.order))
			} else {
				newProblems = append(newProblems, domainContest.RestoreContestProblem(uuid.New().String(), pe.id, pe.order))
			}
		}
		c.SetProblems(newProblems, now)
	}

	if in.Locked != nil {
		c.SetLocked(*in.Locked, now)
	}

	if in.ShowTeamMembers != nil {
		c.SetShowTeamMembers(*in.ShowTeamMembers, now)
	}

	if err := uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return uc.repo.Update(txCtx, c)
	}); err != nil {
		return nil, err
	}

	group, err := uc.groupProvider.FindByID(ctx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if group == nil {
		group = &GroupInfo{ID: in.GroupID}
	}

	owner, err := uc.ownerProvider.GetDisplay(ctx, c.OwnerID().Value())
	if err != nil {
		return nil, err
	}
	if owner == nil {
		owner = &UserDisplay{}
	}

	var problemDisplays []ProblemDisplay
	if problemsChanged {
		problemDisplays = make([]ProblemDisplay, 0, len(newProblemEntries))
		for _, pe := range newProblemEntries {
			problemDisplays = append(problemDisplays, ProblemDisplay{Slug: pe.slug, Title: pe.title, Order: pe.order})
		}
	} else {
		problemDisplays, err = uc.enrichProblems(ctx, c.Problems())
		if err != nil {
			return nil, err
		}
	}

	return buildOutput(c, group, owner, problemDisplays, now), nil
}

func validateScalarFields(in UpdateContestInput, existing *domainContest.Contest) (scalarFields, error) {
	var fieldErrs []apperror.FieldError
	var result scalarFields

	if in.Name != nil {
		n, nameErr := domainContest.NewContestName(*in.Name)
		if err := apperror.AccumulateFieldErrors(nameErr, &fieldErrs); err != nil {
			return result, err
		}
		if nameErr == nil {
			result.name = &n
		}
	}

	if in.Description != nil && *in.Description != nil {
		if len([]rune(**in.Description)) > domainContest.MaxDescriptionLength {
			fieldErrs = append(fieldErrs, apperror.FieldError{
				Field:   "description",
				Message: "description cannot exceed 5000 characters",
			})
		}
	}

	if in.Penalty != nil {
		p, penaltyErr := domainContest.NewPenalty(*in.Penalty)
		if err := apperror.AccumulateFieldErrors(penaltyErr, &fieldErrs); err != nil {
			return result, err
		}
		if penaltyErr == nil {
			result.penalty = &p
		}
	}

	if in.ParticipationMode != nil {
		m, modeErr := domainContest.NewParticipationMode(*in.ParticipationMode)
		if err := apperror.AccumulateFieldErrors(modeErr, &fieldErrs); err != nil {
			return result, err
		}
		if modeErr == nil {
			result.mode = &m
		}
	}

	if in.TeamSizeMin != nil || in.TeamSizeMax != nil {
		minVal := existing.TeamSize().Min()
		if in.TeamSizeMin != nil {
			minVal = *in.TeamSizeMin
		}
		maxVal := existing.TeamSize().Max()
		if in.TeamSizeMax != nil {
			maxVal = *in.TeamSizeMax
		}
		ts, tsErr := domainContest.NewTeamSize(minVal, maxVal)
		if err := apperror.AccumulateFieldErrors(tsErr, &fieldErrs); err != nil {
			return result, err
		}
		if tsErr == nil {
			result.teamSize = &ts
		}
	}

	if len(fieldErrs) > 0 {
		return result, apperror.NewValidation(fieldErrs)
	}
	return result, nil
}

func (uc *UpdateContestUseCase) resolveProblems(ctx context.Context, inputs []ProblemOrderInput, userID string, isAdmin bool) ([]problemEntry, error) {
	return resolveContestProblems(ctx, uc.problemProvider, inputs, userID, isAdmin)
}

// resolveContestProblems deduplicates slugs, fetches problem metadata, and
// validates that each problem is published and accessible to the caller.
// Shared by CreateContestUseCase and UpdateContestUseCase.
func resolveContestProblems(ctx context.Context, provider ProblemProvider, inputs []ProblemOrderInput, userID string, isAdmin bool) ([]problemEntry, error) {
	deduped := deduplicateBySlug(inputs)
	if len(deduped) == 0 {
		return nil, nil
	}

	slugs := make([]string, len(deduped))
	for i, p := range deduped {
		slugs[i] = p.Slug
	}

	infos, err := provider.FindBySlugs(ctx, slugs, userID, isAdmin)
	if err != nil {
		return nil, err
	}

	entries := make([]problemEntry, 0, len(deduped))
	for _, entry := range deduped {
		info, ok := infos[entry.Slug]
		if !ok {
			return nil, apperror.NewNotFound(ErrCodeProblemNotFound, "problem '"+entry.Slug+"' not found")
		}
		if !info.IsPublished {
			return nil, apperror.NewBadRequest(ErrCodeProblemNotPublished, "problem '"+entry.Slug+"' is not published")
		}
		if !info.CanAdd {
			return nil, apperror.NewForbidden(ErrCodeProblemAccessDenied,
				"cannot add private problem '"+entry.Slug+"' — you are not a modifier")
		}
		entries = append(entries, problemEntry{
			id:    info.ID,
			slug:  entry.Slug,
			title: info.Title,
			order: entry.Order,
		})
	}
	return entries, nil
}

func (uc *UpdateContestUseCase) enrichProblems(ctx context.Context, cps []domainContest.ContestProblem) ([]ProblemDisplay, error) {
	if len(cps) == 0 {
		return []ProblemDisplay{}, nil
	}
	ids := make([]string, len(cps))
	for i, cp := range cps {
		ids[i] = cp.ProblemID()
	}
	basics, err := uc.problemProvider.FindByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	displays := make([]ProblemDisplay, 0, len(cps))
	for _, cp := range cps {
		if b, ok := basics[cp.ProblemID()]; ok {
			displays = append(displays, ProblemDisplay{Slug: b.Slug, Title: b.Title, Order: cp.Order()})
		}
	}
	return displays, nil
}

func hasAnyField(in UpdateContestInput) bool {
	return in.Name != nil || in.Description != nil || in.StartTime != nil ||
		in.EndTime != nil || in.Penalty != nil || in.FreezeMinutes != nil ||
		in.EnablePostContest != nil || in.Problems != nil || in.Locked != nil ||
		in.ParticipationMode != nil || in.TeamSizeMin != nil || in.TeamSizeMax != nil ||
		in.ShowTeamMembers != nil
}

func deduplicateBySlug(entries []ProblemOrderInput) []ProblemOrderInput {
	seen := make(map[string]struct{}, len(entries))
	out := make([]ProblemOrderInput, 0, len(entries))
	for _, e := range entries {
		if _, ok := seen[e.Slug]; !ok {
			seen[e.Slug] = struct{}{}
			out = append(out, e)
		}
	}
	return out
}
