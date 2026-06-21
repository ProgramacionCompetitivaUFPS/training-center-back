package submission

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

type ListMySubmissionsInput struct {
	CurrentUser appshared.CurrentUser
	Status      *string
	Language    *string
	ProblemSlug *string
	Page        int
	Limit       int
}

type ListMySubmissionsOutput struct {
	Submissions []SubmissionSummary
	Total       int
	Page        int
	Limit       int
}

type ListMySubmissionsUseCase struct {
	repo           domainsubmission.Repository
	problemDisplay ProblemDisplayProvider
	userDisplay    UserProvider
	contestDisplay ContestProvider
	problemBySlug  ProblemProvider
}

func NewListMySubmissionsUseCase(
	repo domainsubmission.Repository,
	problemDisplay ProblemDisplayProvider,
	userDisplay UserProvider,
	contestDisplay ContestProvider,
	problemBySlug ProblemProvider,
) *ListMySubmissionsUseCase {
	return &ListMySubmissionsUseCase{
		repo:           repo,
		problemDisplay: problemDisplay,
		userDisplay:    userDisplay,
		contestDisplay: contestDisplay,
		problemBySlug:  problemBySlug,
	}
}

func (uc *ListMySubmissionsUseCase) Execute(ctx context.Context, in ListMySubmissionsInput) (*ListMySubmissionsOutput, error) {
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

	uid := shared.RestoreUserID(in.CurrentUser.ID)

	f := domainsubmission.ListFilters{
		UserID:   &uid,
		Status:   in.Status,
		Language: in.Language,
		Page:     in.Page,
		Limit:    in.Limit,
	}

	if in.ProblemSlug != nil {
		info, err := uc.problemBySlug.GetProblemBySlug(ctx, *in.ProblemSlug)
		if err != nil {
			return nil, err
		}
		f.ProblemID = &info.ID
	}

	subs, total, err := uc.repo.List(ctx, f)
	if err != nil {
		return nil, err
	}

	if len(subs) == 0 {
		return &ListMySubmissionsOutput{
			Submissions: []SubmissionSummary{},
			Total:       0,
			Page:        in.Page,
			Limit:       in.Limit,
		}, nil
	}

	// User is the same for all submissions — fetch once.
	me, err := uc.userDisplay.GetUserByID(ctx, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}

	summaries := make([]SubmissionSummary, 0, len(subs))
	for _, s := range subs {
		problem, err := uc.problemDisplay.GetProblemByID(ctx, s.ProblemID())
		if err != nil {
			return nil, err
		}

		var contest *ContestDisplay
		if cid := s.ContestID(); cid != nil {
			cd, err := uc.contestDisplay.GetContestByID(ctx, *cid)
			if err != nil {
				return nil, err
			}
			contest = cd
		}

		summaries = append(summaries, SubmissionSummary{
			ID:            s.ID(),
			Status:        s.Status().String(),
			Visibility:    s.Visibility().String(),
			SubmittedAt:   s.SubmittedAt(),
			JudgedAt:      s.JudgedAt(),
			Problem:       *problem,
			Contest:       contest,
			SubmittedBy:   *me,
			Language:      s.Language().String(),
			ExecutionTime: s.TimeMs(),
			MemoryUsed:    s.MemoryKb(),
		})
	}

	return &ListMySubmissionsOutput{
		Submissions: summaries,
		Total:       total,
		Page:        in.Page,
		Limit:       in.Limit,
	}, nil
}
