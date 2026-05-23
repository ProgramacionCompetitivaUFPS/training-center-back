package problem

import (
	"context"
	"log/slog"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const maxPageLimit = 100

type ListProblemsInput struct {
	CurrentUser    appshared.CurrentUser
	Status         *string
	Accessibility  *string
	Tags           []string
	AuthorNickname *string
	Page           int
	Limit          int
}

type ProblemSummary struct {
	Problem ProblemDTO
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
	if err := appshared.ValidatePagination(in.Page, in.Limit, maxPageLimit); err != nil {
		return nil, err
	}

	filters := problem.ListFilters{
		Tags:  in.Tags,
		Page:  in.Page,
		Limit: in.Limit,
	}

	if in.Status != nil {
		s, err := problem.NewStatus(*in.Status)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "status", Message: "Invalid status. Must be DRAFT or PUBLISHED"},
			})
		}
		filters.Statuses = []problem.Status{s}
	}

	if in.Accessibility != nil {
		acc, err := problem.NewAccessibility(*in.Accessibility)
		if err != nil {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "accessibility", Message: "Invalid accessibility. Must be PUBLIC or PRIVATE"},
			})
		}
		filters.Accessibility = &acc
	}

	isAdmin := in.CurrentUser.IsAdmin()
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
		summaries = append(summaries, ProblemSummary{Problem: problemToDTO(p), Author: author})
	}

	totalPages := appshared.CalcTotalPages(total, in.Limit)

	return &ListProblemsOutput{
		Problems:   summaries,
		TotalCount: total,
		TotalPages: totalPages,
		Page:       in.Page,
		Limit:      in.Limit,
	}, nil
}
