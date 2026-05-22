package problem

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
	"golang.org/x/sync/errgroup"
)

type ImportProblemInput struct {
	Slug        string
	ZipData     []byte
	CurrentUser appshared.CurrentUser
}

type ImportProblemOutput struct {
	Problem ProblemDTO
}

type ImportProblemUseCase struct {
	repo             problem.Repository
	storage          ProblemFileRepository
	packageParser    ICPCPackageParser
	platformSettings problem.PlatformSettings
}

func NewImportProblemUseCase(
	repo problem.Repository,
	storage ProblemFileRepository,
	packageParser ICPCPackageParser,
	platformSettings problem.PlatformSettings,
) *ImportProblemUseCase {
	return &ImportProblemUseCase{
		repo:             repo,
		storage:          storage,
		packageParser:    packageParser,
		platformSettings: platformSettings,
	}
}

func (uc *ImportProblemUseCase) Execute(ctx context.Context, input ImportProblemInput) (*ImportProblemOutput, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only Coach and Admin users can import problems")
	}

	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	exists, err := uc.repo.ExistsBySlug(ctx, slug)
	if err != nil {
		slog.ErrorContext(ctx, "failed to check slug existence before import", "error", err, "slug", slug.String())
		return nil, apperror.NewInternal()
	}
	if exists {
		return nil, apperror.NewConflict(problem.ErrCodeSlugAlreadyExists, "A problem with slug '"+slug.String()+"' already exists")
	}

	pkg, err := uc.packageParser.ParsePackageZip(ctx, input.ZipData)
	if err != nil {
		return nil, err
	}

	var fieldErrs []apperror.FieldError

	title, err := problem.NewTitle(pkg.Title)
	if err != nil {
		var valErr *apperror.AppError
		if errors.As(err, &valErr) {
			fieldErrs = append(fieldErrs, valErr.Details...)
		} else {
			return nil, apperror.NewInternal()
		}
	}

	globalMaxTime, globalMaxMemory := uc.platformSettings.GlobalLimits()

	var timeLimit *problem.TimeLimit
	if pkg.TimeLimitMs != nil {
		tl, err := problem.NewTimeLimit(*pkg.TimeLimitMs, globalMaxTime)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			}
		} else {
			timeLimit = &tl
		}
	}

	var memoryLimit *problem.MemoryLimit
	if pkg.MemoryLimit != nil {
		ml, err := problem.NewMemoryLimit(*pkg.MemoryLimit, globalMaxMemory)
		if err != nil {
			var valErr *apperror.AppError
			if errors.As(err, &valErr) {
				fieldErrs = append(fieldErrs, valErr.Details...)
			}
		} else {
			memoryLimit = &ml
		}
	}

	if len(fieldErrs) > 0 {
		return nil, apperror.NewValidation(fieldErrs)
	}



	var uploadedKeys []string
	cleanup := func() {
		for _, key := range uploadedKeys {
			if err := uc.storage.DeleteFile(ctx, key); err != nil {
				slog.ErrorContext(ctx, "rollback: failed to delete uploaded file", "key", key, "error", err)
			}
		}
	}

	now := time.Now()
	newID := uuid.New().String()
	newProblem, err := problem.NewProblem(
		newID,
		slug,
		title,
		problem.RestoreStatement(pkg.Statement),
		timeLimit,
		memoryLimit,
		nil,
		problem.Tags{},
		shared.RestoreUserID(input.CurrentUser.ID),
		now,
	)
	if err != nil {
		return nil, err
	}

	if pkg.ZipData != nil {
		uploadInstanceID := uuid.New().String()
		basePath := fmt.Sprintf("problems/%s/%s/%s", slug.String(), FileTypeTestCases, uploadInstanceID)
		zipKey := fmt.Sprintf("%s/testcases.zip", basePath)

		if err := uc.storage.UploadFile(ctx, zipKey, pkg.ZipData); err != nil {
			slog.ErrorContext(ctx, "import: failed to upload testcases zip", "error", err)
			cleanup()
			return nil, apperror.NewInternal()
		}
		uploadedKeys = append(uploadedKeys, zipKey)

		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(uc.platformSettings.UploadMaxConcurrency())

		for _, file := range pkg.SampleFiles {
			file := file
			destPath := fmt.Sprintf("%s/%s", basePath, file.Path)
			uploadedKeys = append(uploadedKeys, destPath)
			g.Go(func() error {
				if err := uc.storage.UploadFile(gCtx, destPath, file.Content); err != nil {
					slog.ErrorContext(gCtx, "import: failed to upload sample file", "path", destPath, "error", err)
					return apperror.NewInternal()
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			cleanup()
			return nil, err
		}

		newProblem.SetTestCases(basePath, now)
	}

	for _, sol := range pkg.Solutions {
		cleanName := filepath.Base(sol.Path)
		lang, ok := uc.platformSettings.LanguageByExtension(strings.ToLower(filepath.Ext(cleanName)))
		if !ok {
			continue
		}
		fileKey := fmt.Sprintf("problems/%s/%s/%s", slug.String(), FileTypeSolution, cleanName)
		solutionObj, err := problem.NewSolutionFile(cleanName, fileKey, lang)
		if err != nil {
			cleanup()
			return nil, err
		}
		if err := uc.storage.UploadFile(ctx, fileKey, sol.Content); err != nil {
			slog.ErrorContext(ctx, "import: failed to upload solution", "file", cleanName, "error", err)
			cleanup()
			return nil, apperror.NewInternal()
		}
		uploadedKeys = append(uploadedKeys, fileKey)
		newProblem.AddSolution(solutionObj, now)
	}

	if pkg.Checker != nil {
		if err := uc.uploadVerifier(ctx, slug.String(), FileTypeChecker, pkg.Checker, newProblem, &uploadedKeys, cleanup, now); err != nil {
			return nil, err
		}
	}

	if pkg.Validator != nil {
		if err := uc.uploadVerifier(ctx, slug.String(), FileTypeValidator, pkg.Validator, newProblem, &uploadedKeys, cleanup, now); err != nil {
			return nil, err
		}
	}

	if err := uc.repo.Save(ctx, newProblem); err != nil {
		slog.ErrorContext(ctx, "import: failed to save problem", "error", err, "slug", slug.String())
		cleanup()
		return nil, apperror.NewInternal()
	}

	return &ImportProblemOutput{Problem: problemToDTO(newProblem)}, nil
}

func (uc *ImportProblemUseCase) uploadVerifier(
	ctx context.Context,
	slugStr, fileType string,
	f *ParsedFile,
	p *problem.Problem,
	uploadedKeys *[]string,
	cleanup func(),
	now time.Time,
) error {
	cleanName := filepath.Base(f.Path)
	ext := strings.ToLower(filepath.Ext(cleanName))
	lang, ok := uc.platformSettings.LanguageByExtension(ext)
	if !ok {
		return apperror.NewBadRequest(ErrCodeProblemUnsupportedFileExt, fmt.Sprintf("Unsupported %s file extension: %s", fileType, ext))
	}
	fileKey := fmt.Sprintf("problems/%s/%s/%s", slugStr, fileType, cleanName)

	verifierObj, err := problem.NewVerifierFile(cleanName, fileKey, lang)
	if err != nil {
		cleanup()
		return err
	}

	if err := uc.storage.UploadFile(ctx, fileKey, f.Content); err != nil {
		slog.ErrorContext(ctx, "import: failed to upload "+fileType, "file", cleanName, "error", err)
		cleanup()
		return apperror.NewInternal()
	}
	*uploadedKeys = append(*uploadedKeys, fileKey)

	switch fileType {
	case FileTypeChecker:
		p.SetChecker(verifierObj, now)
	case FileTypeValidator:
		p.SetValidator(verifierObj, now)
	}
	return nil
}
