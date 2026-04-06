package problem

import (
	"context"
	"log/slog"
	"math"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ListProblemsInput struct {
	CurrentUser    user.CurrentUser
	Status         *string
	Accessibility  *string
	Tags           []string
	AuthorNickname *string
	Page           int
	Limit          int
}

type ProblemSummary struct {
	Problem *problem.Problem
	Author  ModifierDisplay
}

type ListProblemsOutput struct {
	Problems   []ProblemSummary
	TotalCount int
	TotalPages int
	Page       int
	Limit      int
}

type ListProblemsUseCase struct {
	repo         problem.Repository
	userProvider UserProvider
}

func NewListProblemsUseCase(repo problem.Repository, userProvider UserProvider) *ListProblemsUseCase {
	return &ListProblemsUseCase{repo: repo, userProvider: userProvider}
}

func (uc *ListProblemsUseCase) Execute(ctx context.Context, in ListProblemsInput) (*ListProblemsOutput, error) {
	if in.Page < 1 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "page", Message: "Page must be a positive integer"},
		})
	}
	if in.Limit < 1 || in.Limit > 100 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "limit", Message: "Limit must be between 1 and 100"},
		})
	}

	filters := problem.ListFilters{
		Tags:          in.Tags,
		Accessibility: in.Accessibility,
		Page:          in.Page,
		Limit:         in.Limit,
	}

	if in.Status != nil {
		filters.Statuses = []string{*in.Status}
	}

	isAdmin := in.CurrentUser.Role == user.RoleAdmin
	if !isAdmin {
		filters.ViewerModifierID = &in.CurrentUser.ID
	}

	if in.AuthorNickname != nil {
		authorID, found, err := uc.userProvider.GetIDByNickname(ctx, *in.AuthorNickname)
		if err != nil {
			slog.ErrorContext(ctx, "failed to get author ID by nickname", "error", err, "nickname", *in.AuthorNickname)
			return nil, apperror.NewInternal()
		}
		if !found {
			return &ListProblemsOutput{
				Problems:   []ProblemSummary{},
				TotalCount: 0,
				TotalPages: 0,
				Page:       in.Page,
				Limit:      in.Limit,
			}, nil
		}
		filters.AuthorID = &authorID
	}

	problems, total, err := uc.repo.List(ctx, filters)
	if err != nil {
		return nil, err
	}

	authorIDs := make([]string, 0, len(problems))
	seen := make(map[string]bool)
	for _, p := range problems {
		id := p.AuthorID().Value()
		if !seen[id] {
			authorIDs = append(authorIDs, id)
			seen[id] = true
		}
	}

	displays, err := uc.userProvider.GetDisplays(ctx, authorIDs)
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch author displays", "error", err)
		return nil, apperror.NewInternal()
	}

	summaries := make([]ProblemSummary, 0, len(problems))
	for _, p := range problems {
		d := displays[p.AuthorID().Value()]
		var author ModifierDisplay
		if d != nil {
			author = ModifierDisplay{Nickname: d.Nickname, Name: d.Name}
		}
		summaries = append(summaries, ProblemSummary{Problem: p, Author: author})
	}

	totalPages := int(math.Ceil(float64(total) / float64(in.Limit)))

	return &ListProblemsOutput{
		Problems:   summaries,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}
