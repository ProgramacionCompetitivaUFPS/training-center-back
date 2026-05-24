package problem

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteProblemFileInput struct {
	Slug        string
	FileType    string
	FileName    string
	CurrentUser appshared.CurrentUser
}

type DeleteProblemFileUseCase struct {
	repo        problem.Repository
	fileStorage ProblemFileRepository
}

func NewDeleteProblemFileUseCase(
	repo problem.Repository,
	fileStorage ProblemFileRepository,
) *DeleteProblemFileUseCase {
	return &DeleteProblemFileUseCase{
		repo:        repo,
		fileStorage: fileStorage,
	}
}

func (usecase *DeleteProblemFileUseCase) Execute(ctx context.Context, input DeleteProblemFileInput) error {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return err
	}

	foundProblem, err := usecase.repo.FindBySlug(ctx, slug)
	if err != nil {
		return err
	}

	if foundProblem.Status().IsPublished() {
		return apperror.NewBadRequest(ErrCodeProblemIsPublished, "Cannot delete files from a published problem. Unpublish first.")
	}

	if !foundProblem.CanBeEditedBy(shared.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.IsAdmin()) {
		return apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can update this problem")
	}

	var storageKeyToDelete string
	var deleteByPrefix bool

	now := time.Now()
	fileType := strings.ToLower(input.FileType)
	switch fileType {
	case FileTypeTestCases:
		if input.FileName != "" {
			return apperror.NewBadRequest(ErrCodeFileNameNotApplicable, "fileName is not applicable for testCases deletion")
		}
		if foundProblem.TestCasesKey() == nil {
			return apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no test cases to delete")
		}
		storageKeyToDelete = *foundProblem.TestCasesKey()
		deleteByPrefix = true
		foundProblem.RemoveTestCases(now)

	case FileTypeSolution:
		if input.FileName == "" {
			return apperror.NewBadRequest(ErrCodeProblemMissingFilename, "fileName is required to delete a solution")
		}
		found := false
		for _, solution := range foundProblem.Solutions() {
			if solution.Filename() == input.FileName {
				storageKeyToDelete = solution.FileKey()
				if err := foundProblem.RemoveSolution(solution.Filename(), now); err != nil {
					return err
				}
				found = true
				break
			}
		}
		if !found {
			return apperror.NewNotFound(ErrCodeProblemFileNotFound, "Solution not found")
		}

	case FileTypeChecker:
		if input.FileName != "" {
			return apperror.NewBadRequest(ErrCodeFileNameNotApplicable, "fileName is not applicable for checker deletion")
		}
		if foundProblem.Checker() == nil {
			return apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no checker to delete")
		}
		storageKeyToDelete = foundProblem.Checker().FileKey()
		foundProblem.RemoveChecker(now)

	case FileTypeValidator:
		if input.FileName != "" {
			return apperror.NewBadRequest(ErrCodeFileNameNotApplicable, "fileName is not applicable for validator deletion")
		}
		if foundProblem.Validator() == nil {
			return apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no validator to delete")
		}
		storageKeyToDelete = foundProblem.Validator().FileKey()
		foundProblem.RemoveValidator(now)

	default:
		slog.WarnContext(ctx, "invalid file type provided for deletion", "file_type", input.FileType, "slug", foundProblem.Slug().String())
		return apperror.NewBadRequest(ErrCodeProblemInvalidFileType, "Invalid file type. Allowed: testCases, solution, checker, validator")
	}

	if deleteByPrefix {
		if err := usecase.fileStorage.DeleteFilesWithPrefix(ctx, storageKeyToDelete); err != nil {
			return err
		}
	} else {
		if err := usecase.fileStorage.DeleteFile(ctx, storageKeyToDelete); err != nil {
			return err
		}
	}

	if err := usecase.repo.Save(ctx, foundProblem); err != nil {
		return err
	}

	return nil
}
