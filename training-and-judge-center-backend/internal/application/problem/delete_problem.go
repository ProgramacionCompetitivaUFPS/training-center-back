package problem

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteProblemInput struct {
	Slug        string
	ConfirmSlug string
	CurrentUser user.CurrentUser
}

type DeleteProblemUseCase struct {
	repo        problem.Repository
	fileStorage ProblemFileRepository
}

func NewDeleteProblemUseCase(repo problem.Repository, fileStorage ProblemFileRepository) *DeleteProblemUseCase {
	return &DeleteProblemUseCase{repo: repo, fileStorage: fileStorage}
}

func (uc *DeleteProblemUseCase) Execute(ctx context.Context, in DeleteProblemInput) error {
	if in.ConfirmSlug == "" {
		return apperror.NewValidation([]apperror.FieldError{
			{Field: "confirmSlug", Message: "Must match the problem slug exactly"},
		})
	}

	slug, err := problem.NewSlug(in.Slug)
	if err != nil {
		return err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if in.ConfirmSlug != p.Slug.String() {
		return apperror.NewBadRequest(problem.ErrCodeSlugMismatch, "Confirmation slug does not match the problem slug")
	}

	viewerID := problem.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.Role == user.RoleAdmin
	if p.AuthorID != viewerID && !isAdmin {
		return apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author or Admin can delete this problem")
	}

	if p.TestCasesKey != nil {
		if err := uc.fileStorage.DeleteFilesWithPrefix(ctx, *p.TestCasesKey); err != nil {
			slog.ErrorContext(ctx, "failed to delete test cases from storage", "prefix", *p.TestCasesKey, "error", err)
		}
	}

	for _, sol := range p.Solutions {
		if err := uc.fileStorage.DeleteFile(ctx, sol.FileKey()); err != nil {
			slog.ErrorContext(ctx, "failed to delete solution from storage", "key", sol.FileKey(), "error", err)
		}
	}

	if p.Checker != nil {
		if err := uc.fileStorage.DeleteFile(ctx, p.Checker.FileKey()); err != nil {
			slog.ErrorContext(ctx, "failed to delete checker from storage", "key", p.Checker.FileKey(), "error", err)
		}
	}

	if p.Validator != nil {
		if err := uc.fileStorage.DeleteFile(ctx, p.Validator.FileKey()); err != nil {
			slog.ErrorContext(ctx, "failed to delete validator from storage", "key", p.Validator.FileKey(), "error", err)
		}
	}

	return uc.repo.Delete(ctx, p.ID)
}
