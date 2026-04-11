package problem

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteProblemInput struct {
	Slug        string
	ConfirmSlug string
	CurrentUser shared.CurrentUser
}

type DeleteProblemUseCase struct {
	repo        problem.Repository
	fileStorage ProblemFileRepository
}

const maxRetries = 3

func NewDeleteProblemUseCase(repo problem.Repository, fileStorage ProblemFileRepository) *DeleteProblemUseCase {
	return &DeleteProblemUseCase{repo: repo, fileStorage: fileStorage}
}

func (uc *DeleteProblemUseCase) Execute(ctx context.Context, in DeleteProblemInput) (struct{}, error) {
	if in.ConfirmSlug == "" {
		return struct{}{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "confirmSlug", Message: "Must match the problem slug exactly"},
		})
	}

	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return struct{}{}, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return struct{}{}, err
	}

	if in.ConfirmSlug != p.Slug().String() {
		return struct{}{}, apperror.NewBadRequest(problem.ErrCodeSlugMismatch, "Confirmation slug does not match the problem slug")
	}

	viewerID := shared.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.IsAdmin()
	if p.AuthorID() != viewerID && !isAdmin {
		return struct{}{}, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author or Admin can delete this problem")
	}

	if err := uc.repo.Delete(ctx, p.ID()); err != nil {
		return struct{}{}, err
	}

	var cleanupFailed bool

	if p.TestCasesKey() != nil {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFilesWithPrefix(ctx, *p.TestCasesKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but test cases storage cleanup failed", "prefix", *p.TestCasesKey(), "error", err)
			cleanupFailed = true
		}
	}

	for _, sol := range p.Solutions() {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFile(ctx, sol.FileKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but solution storage cleanup failed", "key", sol.FileKey(), "error", err)
			cleanupFailed = true
		}
	}

	if p.Checker() != nil {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFile(ctx, p.Checker().FileKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but checker storage cleanup failed", "key", p.Checker().FileKey(), "error", err)
			cleanupFailed = true
		}
	}

	if p.Validator() != nil {
		var err error
		for i := 0; i < maxRetries; i++ {
			err = uc.fileStorage.DeleteFile(ctx, p.Validator().FileKey())
			if err == nil {
				break
			}
		}
		if err != nil {
			slog.ErrorContext(ctx, "problem deleted from DB but validator storage cleanup failed", "key", p.Validator().FileKey(), "error", err)
			cleanupFailed = true
		}
	}

	if cleanupFailed {
		return struct{}{}, apperror.NewInternal()
	}

	return struct{}{}, nil
}
