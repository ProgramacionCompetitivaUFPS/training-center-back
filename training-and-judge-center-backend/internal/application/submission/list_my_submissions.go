package submission

import (
	"context"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListMySubmissionsInput struct {
	CurrentUser     appshared.CurrentUser
	Status          *string
	Language        *string
	ProblemSlug     *string
	SubmittedFrom   *time.Time
	SubmittedBefore *time.Time
	Page            int
	Limit           int
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
		UserID:          &uid,
		Status:          in.Status,
		Language:        in.Language,
		SubmittedFrom:   in.SubmittedFrom,
		SubmittedBefore: in.SubmittedBefore,
		Page:            in.Page,
		Limit:           in.Limit,
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

	// Batch problem/contest lookups instead of one query per submission (was up to
	// 2*maxPageLimit extra round trips per page).
	problemIDs := make([]string, 0, len(subs))
	seenProblem := make(map[string]bool, len(subs))
	contestIDs := make([]string, 0, len(subs))
	seenContest := make(map[string]bool, len(subs))
	for _, s := range subs {
		if pid := s.ProblemID(); pid != "" && !seenProblem[pid] {
			problemIDs = append(problemIDs, pid)
			seenProblem[pid] = true
		}
		if cid := s.ContestID(); cid != nil && !seenContest[*cid] {
			contestIDs = append(contestIDs, *cid)
			seenContest[*cid] = true
		}
	}

	problemDisplays, err := uc.problemDisplay.GetProblemsByIDs(ctx, problemIDs)
	if err != nil {
		return nil, err
	}
	contestDisplays, err := uc.contestDisplay.GetContestsByIDs(ctx, contestIDs)
	if err != nil {
		return nil, err
	}

	summaries := make([]SubmissionSummary, 0, len(subs))
	for _, s := range subs {
		problem := resolveProblemDisplayFromMap(problemDisplays, s)

		var contest *ContestDisplay
		if cid := s.ContestID(); cid != nil {
			cd, ok := contestDisplays[*cid]
			if !ok {
				// A contest_id on a submission is never stale the way problem_id can be
				// after a deletion — same error the single-row GetContestByID would give.
				return nil, apperror.NewNotFound(domainContest.ErrCodeContestNotFound, "contest not found")
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
			IsBlockly:     s.IsBlockly(),
			ExecutionTime: s.TimeMs(),
			MemoryKb:      s.MemoryKb(),
		})
	}

	return &ListMySubmissionsOutput{
		Submissions: summaries,
		Total:       total,
		Page:        in.Page,
		Limit:       in.Limit,
	}, nil
}
