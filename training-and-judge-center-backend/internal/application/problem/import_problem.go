package problem

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
	"golang.org/x/sync/errgroup"
)

type ImportProblemInput struct {
	Slug        string
	Zip         StagedFile
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

type importUploads struct {
	storage ProblemFileRepository
	slug    string
	keys    []string
}

func (u *importUploads) cleanup(ctx context.Context) {
	var left []string
	for _, key := range u.keys {
		if err := u.storage.DeleteFile(ctx, key); err != nil {
			left = append(left, key)
		}
	}
	if len(left) > 0 {
		slog.ErrorContext(ctx, "import_problem: import failed and the files it had uploaded were left in storage", "slug", u.slug, "keys", left)
	}
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
	slug, err := uc.validateRequest(ctx, input)
	if err != nil {
		return nil, err
	}

	pkg, err := uc.packageParser.ParsePackageZip(ctx, input.Zip)
	if err != nil {
		return nil, err
	}

	title, timeLimit, memoryLimit, err := uc.validatePackage(pkg)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	newProblem, err := problem.NewProblem(
		uuid.New().String(),
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

	uploads := &importUploads{storage: uc.storage, slug: slug.String()}
	if err := uc.uploadPackage(ctx, uploads, pkg, newProblem, now); err != nil {
		return nil, err
	}

	if err := uc.repo.Save(ctx, newProblem); err != nil {
		uploads.cleanup(ctx)
		return nil, err
	}

	return &ImportProblemOutput{Problem: problemToDTO(newProblem)}, nil
}

func (uc *ImportProblemUseCase) validateRequest(ctx context.Context, input ImportProblemInput) (problem.Slug, error) {
	if input.CurrentUser.Role != shared.RoleCoach && !input.CurrentUser.IsAdmin() {
		return problem.Slug{}, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only Coach and Admin users can import problems")
	}

	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return problem.Slug{}, err
	}

	exists, err := uc.repo.ExistsBySlug(ctx, slug)
	if err != nil {
		return problem.Slug{}, err
	}
	if exists {
		return problem.Slug{}, apperror.NewConflict(problem.ErrCodeSlugAlreadyExists, "A problem with slug '"+slug.String()+"' already exists")
	}

	return slug, nil
}

func (uc *ImportProblemUseCase) validatePackage(pkg *ParsedPackage) (problem.Title, *problem.TimeLimit, *problem.MemoryLimit, error) {
	var fieldErrs []apperror.FieldError

	title, titleErr := problem.NewTitle(pkg.Title)
	if err := apperror.AccumulateFieldErrors(titleErr, &fieldErrs); err != nil {
		return problem.Title{}, nil, nil, err
	}

	timeLimit, memoryLimit, err := validateLimits(uc.platformSettings, pkg.TimeLimitMs, pkg.MemoryLimit, &fieldErrs)
	if err != nil {
		return problem.Title{}, nil, nil, err
	}

	if len(fieldErrs) > 0 {
		return problem.Title{}, nil, nil, apperror.NewValidation(fieldErrs)
	}

	return title, timeLimit, memoryLimit, nil
}

func (uc *ImportProblemUseCase) uploadPackage(ctx context.Context, uploads *importUploads, pkg *ParsedPackage, p *problem.Problem, now time.Time) error {
	if pkg.Zip != nil {
		if err := uc.uploadTestCases(ctx, uploads, pkg, p, now); err != nil {
			return err
		}
	}

	if err := uc.uploadSolutions(ctx, uploads, pkg.Solutions, p, now); err != nil {
		return err
	}

	if pkg.Checker != nil {
		if err := uc.uploadVerifier(ctx, uploads, FileTypeChecker, verifierBaseNameChecker, pkg.Checker, p, now); err != nil {
			return err
		}
	}

	if pkg.Validator != nil {
		if err := uc.uploadVerifier(ctx, uploads, FileTypeValidator, verifierBaseNameValidator, pkg.Validator, p, now); err != nil {
			return err
		}
	}

	return nil
}

func (uc *ImportProblemUseCase) uploadTestCases(ctx context.Context, uploads *importUploads, pkg *ParsedPackage, p *problem.Problem, now time.Time) error {
	basePath := fmt.Sprintf("problems/%s/%s/%s", uploads.slug, FileTypeTestCases, uuid.New().String())
	zipKey := TestCasesZipKey(basePath)

	if err := uc.storage.UploadFileStream(ctx, zipKey, stagedReader(pkg.Zip)); err != nil {
		uploads.cleanup(ctx)
		return err
	}
	uploads.keys = append(uploads.keys, zipKey)

	if err := uc.uploadSamples(ctx, uploads, basePath, pkg.SampleFiles); err != nil {
		uploads.cleanup(ctx)
		return err
	}

	p.SetTestCases(basePath, now)
	return nil
}

func (uc *ImportProblemUseCase) uploadSamples(ctx context.Context, uploads *importUploads, basePath string, samples []ParsedFile) error {
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(uc.platformSettings.UploadMaxConcurrency())

	for _, file := range samples {
		destPath := fmt.Sprintf("%s/%s", basePath, file.Path)
		uploads.keys = append(uploads.keys, destPath)
		content := file.Content
		g.Go(func() error {
			return uc.storage.UploadFile(gCtx, destPath, content)
		})
	}

	return g.Wait()
}

func (uc *ImportProblemUseCase) uploadSolutions(ctx context.Context, uploads *importUploads, solutions []ParsedFile, p *problem.Problem, now time.Time) error {
	for _, sol := range solutions {
		cleanName := filepath.Base(sol.Path)
		lang, ok := uc.platformSettings.LanguageByExtension(strings.ToLower(filepath.Ext(cleanName)))
		if !ok {
			continue
		}

		fileKey := fmt.Sprintf("problems/%s/%s/%s", uploads.slug, FileTypeSolution, cleanName)
		solutionObj, err := problem.NewSolutionFile(cleanName, fileKey, lang)
		if err != nil {
			uploads.cleanup(ctx)
			return err
		}
		if err := uc.storage.UploadFile(ctx, fileKey, sol.Content); err != nil {
			uploads.cleanup(ctx)
			return err
		}
		uploads.keys = append(uploads.keys, fileKey)
		p.AddSolution(solutionObj, now)
	}

	return nil
}

func (uc *ImportProblemUseCase) uploadVerifier(
	ctx context.Context,
	uploads *importUploads,
	fileType, baseName string,
	f *ParsedFile,
	p *problem.Problem,
	now time.Time,
) error {
	cleanName := filepath.Base(f.Path)
	ext := strings.ToLower(filepath.Ext(cleanName))
	lang, ok := uc.platformSettings.LanguageByExtension(ext)
	if !ok {
		return apperror.NewBadRequest(ErrCodeProblemUnsupportedFileExt, fmt.Sprintf("Unsupported %s file extension: %s", fileType, ext))
	}
	fileKey := fmt.Sprintf("problems/%s/%s/%s%s", uploads.slug, fileType, baseName, ext)

	verifierObj, err := problem.NewVerifierFile(cleanName, fileKey, lang)
	if err != nil {
		uploads.cleanup(ctx)
		return err
	}

	if err := uc.storage.UploadFile(ctx, fileKey, f.Content); err != nil {
		uploads.cleanup(ctx)
		return err
	}
	uploads.keys = append(uploads.keys, fileKey)

	switch fileType {
	case FileTypeChecker:
		p.SetChecker(verifierObj, now)
	case FileTypeValidator:
		p.SetValidator(verifierObj, now)
	}
	return nil
}
