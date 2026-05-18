package contest

import (
	"context"
	"log/slog"
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
	Problems *[]ProblemOrderInput
	Locked   *bool
}

type ProblemOrderInput struct {
	Slug  string
	Order int
}

type UpdateContestUseCase struct {
	repo            domainContest.Repository
	groupProvider   GroupProvider
	memberProvider  GroupMemberProvider
	problemProvider ProblemProvider
	ownerProvider   OwnerProvider
}

func NewUpdateContestUseCase(
	repo domainContest.Repository,
	groupProvider GroupProvider,
	memberProvider GroupMemberProvider,
	problemProvider ProblemProvider,
	ownerProvider OwnerProvider,
) *UpdateContestUseCase {
	return &UpdateContestUseCase{
		repo:            repo,
		groupProvider:   groupProvider,
		memberProvider:  memberProvider,
		problemProvider: problemProvider,
		ownerProvider:   ownerProvider,
	}
}

func (uc *UpdateContestUseCase) Execute(ctx context.Context, in UpdateContestInput) (*ContestOutput, error) {
	c, err := uc.repo.FindByID(ctx, in.ContestID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
	}

	if c.GroupID().Value() != in.GroupID {
		return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found in this group")
	}

	if !in.CurrentUser.IsAdmin() {
		isLead, err := uc.memberProvider.IsLeadOfGroup(ctx, in.CurrentUser.ID, in.GroupID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to check group membership", "error", err,
				"user_id", in.CurrentUser.ID, "group_id", in.GroupID)
			return nil, apperror.NewInternal()
		}
		if !isLead {
			return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "only group leads can update contests")
		}
	}

	isOwner := c.OwnerID().Value() == in.CurrentUser.ID

	// Lock/unlock is restricted to owner or admin.
	if in.Locked != nil && !isOwner && !in.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeOnlyOwnerOrAdmin, "only the contest owner or admin can lock or unlock the contest")
	}

	// Locked contests block all changes except lock/unlock by owner or admin.
	if c.Locked() && !isOwner && !in.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(domainContest.ErrCodeContestLocked,
			"contest is locked and cannot be modified")
	}

	if !hasAnyField(in) {
		return nil, apperror.NewBadRequest(ErrCodeNoFieldsToUpdate, "at least one field must be provided for update")
	}

	// Validate scalar fields.
	var fieldErrs []apperror.FieldError

	var namePtr *domainContest.ContestName
	if in.Name != nil {
		n, nameErr := domainContest.NewContestName(*in.Name)
		if err := apperror.AccumulateFieldErrors(nameErr, &fieldErrs); err != nil {
			return nil, err
		}
		if nameErr == nil {
			namePtr = &n
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

	var penaltyPtr *domainContest.Penalty
	if in.Penalty != nil {
		p, penaltyErr := domainContest.NewPenalty(*in.Penalty)
		if err := apperror.AccumulateFieldErrors(penaltyErr, &fieldErrs); err != nil {
			return nil, err
		}
		if penaltyErr == nil {
			penaltyPtr = &p
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}

	// Resolve problems if the list was provided.
	type problemEntry struct {
		id    string
		slug  string
		title string
		order int
	}
	var newProblemEntries []problemEntry
	problemsChanged := in.Problems != nil

	if problemsChanged {
		deduped := deduplicateBySlug(*in.Problems)
		slugs := make([]string, len(deduped))
		for i, p := range deduped {
			slugs[i] = p.Slug
		}

		if len(slugs) > 0 {
			infos, err := uc.problemProvider.FindBySlugs(ctx, slugs, in.CurrentUser.ID, in.CurrentUser.IsAdmin())
			if err != nil {
				slog.ErrorContext(ctx, "failed to fetch problems by slug", "error", err)
				return nil, apperror.NewInternal()
			}
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
				newProblemEntries = append(newProblemEntries, problemEntry{
					id:    info.ID,
					slug:  entry.Slug,
					title: info.Title,
					order: entry.Order,
				})
			}
		}
	}

	now := time.Now()

	// Apply scalar updates (also validates times).
	if err := c.Update(namePtr, in.Description, in.StartTime, in.EndTime, penaltyPtr, in.FreezeMinutes, in.EnablePostContest, now); err != nil {
		return nil, err
	}

	// Replace problem list if provided.
	if problemsChanged {
		existingByProblemID := make(map[string]domainContest.ContestProblem, len(c.Problems()))
		for _, cp := range c.Problems() {
			existingByProblemID[cp.ProblemID()] = cp
		}

		newProblems := make([]domainContest.ContestProblem, 0, len(newProblemEntries))
		for _, pe := range newProblemEntries {
			if existing, ok := existingByProblemID[pe.id]; ok {
				// Keep existing row ID, update order.
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

	if err := uc.repo.Update(ctx, c); err != nil {
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
		slog.ErrorContext(ctx, "failed to resolve owner display", "error", err, "owner_id", c.OwnerID().Value())
		return nil, apperror.NewInternal()
	}
	if owner == nil {
		owner = &UserDisplay{}
	}

	// Build problem displays.
	var problemDisplays []ProblemDisplay
	if problemsChanged {
		problemDisplays = make([]ProblemDisplay, 0, len(newProblemEntries))
		for _, pe := range newProblemEntries {
			problemDisplays = append(problemDisplays, ProblemDisplay{
				Slug:  pe.slug,
				Title: pe.title,
				Order: pe.order,
			})
		}
	} else {
		problemDisplays, err = uc.enrichProblems(ctx, c.Problems())
		if err != nil {
			return nil, err
		}
	}

	return buildOutput(c, group, owner, problemDisplays, now), nil
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
		slog.ErrorContext(ctx, "failed to enrich contest problems", "error", err)
		return nil, apperror.NewInternal()
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
		in.EnablePostContest != nil || in.Problems != nil || in.Locked != nil
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
