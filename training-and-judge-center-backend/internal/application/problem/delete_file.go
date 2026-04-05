package problem

import (
	"context"
	"log/slog"
	"strings"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type DeleteProblemFileInput struct {
	Slug        string
	FileType    string
	FileName    string
	CurrentUser user.CurrentUser
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

func (usecase *DeleteProblemFileUseCase) Execute(ctx context.Context, input DeleteProblemFileInput) (struct{}, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return struct{}{}, err
	}

	foundProblem, err := usecase.repo.FindBySlug(ctx, slug)
	if err != nil {
		return struct{}{}, err
	}

	if foundProblem.Status.IsPublished() {
		return struct{}{}, apperror.NewBadRequest(ErrCodeProblemIsPublished, "Cannot delete files from a published problem. Unpublish first.")
	}

	if !foundProblem.CanBeEditedBy(problem.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.Role == user.RoleAdmin) {
		return struct{}{}, apperror.NewForbidden(apperror.ErrCodeForbidden, "Only the problem author, Admin, or assigned modifiers can update this problem")
	}

	var storageKeyToDelete string
	var deleteByPrefix bool

	fileType := strings.ToLower(input.FileType)
	switch fileType {
	case FileTypeTestCases:
		if input.FileName != "" {
			return struct{}{}, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "fileName is not applicable for testCases deletion")
		}
		if foundProblem.TestCasesKey == nil {
			return struct{}{}, apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no test cases to delete")
		}
		storageKeyToDelete = *foundProblem.TestCasesKey
		deleteByPrefix = true
		foundProblem.RemoveTestCases()

	case FileTypeSolution:
		if input.FileName == "" {
			return struct{}{}, apperror.NewBadRequest(ErrCodeProblemMissingFilename, "fileName is required to delete a solution")
		}
		found := false
		for _, solution := range foundProblem.Solutions {
			if solution.Filename() == input.FileName {
				storageKeyToDelete = solution.FileKey()
				foundProblem.RemoveSolution(solution.Filename())
				found = true
				break
			}
		}
		if !found {
			return struct{}{}, apperror.NewNotFound(ErrCodeProblemFileNotFound, "Solution not found")
		}

	case FileTypeChecker:
		if input.FileName != "" {
			return struct{}{}, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "fileName is not applicable for checker deletion")
		}
		if foundProblem.Checker == nil {
			return struct{}{}, apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no checker to delete")
		}
		storageKeyToDelete = foundProblem.Checker.FileKey()
		foundProblem.RemoveChecker()

	case FileTypeValidator:
		if input.FileName != "" {
			return struct{}{}, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "fileName is not applicable for validator deletion")
		}
		if foundProblem.Validator == nil {
			return struct{}{}, apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no validator to delete")
		}
		storageKeyToDelete = foundProblem.Validator.FileKey()
		foundProblem.RemoveValidator()

	default:
		slog.WarnContext(ctx, "invalid file type provided for deletion", "file_type", input.FileType, "slug", foundProblem.Slug.String())
		return struct{}{}, apperror.NewBadRequest(ErrCodeProblemInvalidFileType, "Invalid file type. Allowed: testCases, solution, checker, validator")
	}

	if deleteByPrefix {
		if err := usecase.fileStorage.DeleteFilesWithPrefix(ctx, storageKeyToDelete); err != nil {
			slog.ErrorContext(ctx, "failed to delete files from storage", "prefix", storageKeyToDelete, "error", err)
			return struct{}{}, apperror.NewInternal()
		}
	} else {
		if err := usecase.fileStorage.DeleteFile(ctx, storageKeyToDelete); err != nil {
			slog.ErrorContext(ctx, "failed to delete file from storage", "key", storageKeyToDelete, "error", err)
			return struct{}{}, apperror.NewInternal()
		}
	}

	if err := usecase.repo.Save(ctx, foundProblem); err != nil {
		slog.ErrorContext(ctx, "failed to save problem after file deletion", "error", err, "slug", foundProblem.Slug.String())
		return struct{}{}, apperror.NewInternal()
	}

	return struct{}{}, nil
}
