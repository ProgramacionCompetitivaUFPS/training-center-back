package submission

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

type ListProblemSubmissionsInput struct {
	CurrentUser appshared.CurrentUser
	ProblemSlug string
	Status      *string
	Language    *string
	OnlyMine    bool
	Page        int
	Limit       int
}

type ListProblemSubmissionsOutput struct {
	Submissions []SubmissionSummary
	Total       int
	Page        int
	Limit       int
}

type ListProblemSubmissionsUseCase struct {
	repo          domainsubmission.Repository
	problemBySlug ProblemProvider
	userDisplay   UserProvider
}

func NewListProblemSubmissionsUseCase(
	repo domainsubmission.Repository,
	problemBySlug ProblemProvider,
	userDisplay UserProvider,
) *ListProblemSubmissionsUseCase {
	return &ListProblemSubmissionsUseCase{
		repo:          repo,
		problemBySlug: problemBySlug,
		userDisplay:   userDisplay,
	}
}

func (uc *ListProblemSubmissionsUseCase) Execute(ctx context.Context, in ListProblemSubmissionsInput) (*ListProblemSubmissionsOutput, error) {
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxPageLimit); err != nil {
		return nil, err
	}

	if in.Status != nil {
		if _, err := domainsubmission.NewStatus(*in.Status); err != nil {
			return nil, err
		}
	}

	if in.Language != nil {
		if _, err := domainsubmission.NewLanguage(*in.Language); err != nil {
			return nil, err
		}
	}

	problem, err := uc.problemBySlug.GetProblemBySlug(ctx, in.ProblemSlug)
	if err != nil {
		return nil, err
	}

	isAdmin := in.CurrentUser.Role == shared.RoleAdmin

	pub := domainsubmission.VisibilityPublic.String()
	f := domainsubmission.ListFilters{
		ProblemID:      &problem.ID,
		ExcludeContest: true,
		Status:         in.Status,
		Language:       in.Language,
		Page:           in.Page,
		Limit:          in.Limit,
	}

	switch {
	case in.OnlyMine:
		uid := shared.RestoreUserID(in.CurrentUser.ID)
		f.UserID = &uid
	case !isAdmin:
		f.Visibility = &pub
	}

	subs, total, err := uc.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}

	problemDisplay := ProblemDisplay{Slug: problem.Slug, Title: problem.Title}

	summaries := make([]SubmissionSummary, 0, len(subs))
	for _, s := range subs {
		user, err := uc.userDisplay.GetUserByID(ctx, s.UserID().String())
		if err != nil {
			return nil, err
		}

		summaries = append(summaries, SubmissionSummary{
			ID:            s.ID(),
			Status:        s.Status().String(),
			Visibility:    s.Visibility().String(),
			SubmittedAt:   s.SubmittedAt(),
			JudgedAt:      s.JudgedAt(),
			Problem:       problemDisplay,
			Contest:       nil,
			SubmittedBy:   *user,
			Language:      s.Language().String(),
			ExecutionTime: s.TimeMs(),
			MemoryKb:      s.MemoryKb(),
		})
	}

	return &ListProblemSubmissionsOutput{
		Submissions: summaries,
		Total:       total,
		Page:        in.Page,
		Limit:       in.Limit,
	}, nil
}
