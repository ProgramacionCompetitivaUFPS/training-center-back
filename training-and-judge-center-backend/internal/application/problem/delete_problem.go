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

	if in.ConfirmSlug != p.Slug.String() {
		return struct{}{}, apperror.NewBadRequest(problem.ErrCodeSlugMismatch, "Confirmation slug does not match the problem slug")
	}

	viewerID := problem.RestoreUserID(in.CurrentUser.ID)
	isAdmin := in.CurrentUser.Role == user.RoleAdmin
	if p.AuthorID != viewerID && !isAdmin {
		return struct{}{}, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author or Admin can delete this problem")
	}

	if err := uc.repo.Delete(ctx, p.ID); err != nil {
		return struct{}{}, err
	}

	if p.TestCasesKey != nil {
		for i := 0; i < maxRetries; i++ {
			err := uc.fileStorage.DeleteFilesWithPrefix(ctx, *p.TestCasesKey)
			if err == nil {
				break
			}
			if i == maxRetries-1 {
				slog.ErrorContext(ctx, "failed to delete test cases from storage after retries", "prefix", *p.TestCasesKey, "error", err)
			}
		}
	}

	for _, sol := range p.Solutions {
		for i := 0; i < maxRetries; i++ {
			err := uc.fileStorage.DeleteFile(ctx, sol.FileKey())
			if err == nil {
				break
			}
			if i == maxRetries-1 {
				slog.ErrorContext(ctx, "failed to delete solution from storage after retries", "key", sol.FileKey(), "error", err)
			}
		}
	}

	if p.Checker != nil {
		for i := 0; i < maxRetries; i++ {
			err := uc.fileStorage.DeleteFile(ctx, p.Checker.FileKey())
			if err == nil {
				break
			}
			if i == maxRetries-1 {
				slog.ErrorContext(ctx, "failed to delete checker from storage after retries", "key", p.Checker.FileKey(), "error", err)
			}
		}
	}

	if p.Validator != nil {
		for i := 0; i < maxRetries; i++ {
			err := uc.fileStorage.DeleteFile(ctx, p.Validator.FileKey())
			if err == nil {
				break
			}
			if i == maxRetries-1 {
				slog.ErrorContext(ctx, "failed to delete validator from storage after retries", "key", p.Validator.FileKey(), "error", err)
			}
		}
	}

	return struct{}{}, nil
}
