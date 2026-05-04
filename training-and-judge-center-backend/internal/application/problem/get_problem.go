package problem

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetProblemInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type ModifierDisplay struct {
	Nickname string
	Name     string
}

type FilesAvailability struct {
	TestCases bool
	Solutions []SolutionInfo
	Checker   bool
	Validator bool
}

type SolutionInfo struct {
	Filename string
	Language string
}

type GetProblemOutput struct {
	Problem   *problem.Problem
	Author    ModifierDisplay
	Modifiers []ModifierDisplay
	Files     *FilesAvailability
}

type GetProblemUseCase struct {
	repo         problem.Repository
	userProvider UserProvider
}

func NewGetProblemUseCase(repo problem.Repository, userProvider UserProvider) *GetProblemUseCase {
	return &GetProblemUseCase{repo: repo, userProvider: userProvider}
}

func (uc *GetProblemUseCase) Execute(ctx context.Context, in GetProblemInput) (*GetProblemOutput, error) {
	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.IsAdmin()
	isModifier := p.CanBeEditedBy(viewerID, isAdmin)

	if p.Status().String() == "DRAFT" && !isModifier {
		return nil, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can view this DRAFT problem")
	}

	authorDisplay, err := uc.userProvider.GetDisplay(ctx, p.AuthorID().Value())
	if err != nil {
		slog.ErrorContext(ctx, "failed to fetch author display", "error", err, "author_id", p.AuthorID().Value())
		return nil, apperror.NewInternal()
	}

	out := &GetProblemOutput{
		Problem: p,
		Author:  ModifierDisplay{Nickname: authorDisplay.Nickname, Name: authorDisplay.Name},
	}

	if isModifier {
		modifierIDs := make([]string, len(p.ModifierIDs()))
		for i, id := range p.ModifierIDs() {
			modifierIDs[i] = id.Value()
		}
		displays, err := uc.userProvider.GetDisplays(ctx, modifierIDs)
		if err != nil {
			slog.ErrorContext(ctx, "failed to fetch modifier displays", "error", err)
			return nil, apperror.NewInternal()
		}
		modifiers := make([]ModifierDisplay, 0, len(p.ModifierIDs()))
		for _, id := range modifierIDs {
			if d, ok := displays[id]; ok {
				modifiers = append(modifiers, ModifierDisplay{Nickname: d.Nickname, Name: d.Name})
			}
		}
		out.Modifiers = modifiers

		solutions := make([]SolutionInfo, 0, len(p.Solutions()))
		for _, sol := range p.Solutions() {
			solutions = append(solutions, SolutionInfo{Filename: sol.Filename(), Language: sol.Language()})
		}
		out.Files = &FilesAvailability{
			TestCases: p.TestCasesKey() != nil,
			Solutions: solutions,
			Checker:   p.Checker() != nil,
			Validator: p.Validator() != nil,
		}
	}

	return out, nil
}
