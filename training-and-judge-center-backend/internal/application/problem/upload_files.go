package problem

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	FileTypeTestCases = "testcases"
	FileTypeSolution  = "solution"
	FileTypeChecker   = "checker"
	FileTypeValidator = "validator"
)

type UploadProblemFilesInput struct {
	Slug        string
	FileType    string
	FileName    string
	FileData    []byte
	CurrentUser appshared.CurrentUser
}

type UploadProblemFilesOutput struct {
	Message  string
	FileType string
	FileName string
	Problem  ProblemDTO
}

type fileAction struct {
	rollbackFiles []string
	cleanupFiles  []string
	cleanupPrefix string
}

type UploadProblemFilesUseCase struct {
	repo           problem.Repository
	validationRepo problem.ProblemValidationRepository
	storage        ProblemFileRepository
	zipParser      ZipParser
	settings       problem.PlatformSettings
}

func NewUploadProblemFilesUseCase(
	repo problem.Repository,
	validationRepo problem.ProblemValidationRepository,
	storage ProblemFileRepository,
	zipParser ZipParser,
	settings problem.PlatformSettings,
) *UploadProblemFilesUseCase {
	return &UploadProblemFilesUseCase{
		repo:           repo,
		validationRepo: validationRepo,
		storage:        storage,
		zipParser:      zipParser,
		settings:       settings,
	}
}

func (uc *UploadProblemFilesUseCase) Execute(ctx context.Context, input UploadProblemFilesInput) (*UploadProblemFilesOutput, error) {
	slug, err := problem.NewSlug(input.Slug)
	if err != nil {
		return nil, err
	}

	p, err := uc.repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	if p.Status().IsPublished() {
		return nil, apperror.NewBadRequest(ErrCodeProblemIsPublished, "Cannot upload files to a published problem. Unpublish first.")
	}

	if !p.CanBeEditedBy(shared.RestoreUserID(input.CurrentUser.ID), input.CurrentUser.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can update this problem")
	}

	if err := ensureNoActiveValidation(ctx, uc.validationRepo, p.ID()); err != nil {
		return nil, err
	}

	now := time.Now()
	var action fileAction
	var handleErr error

	fileType := strings.ToLower(input.FileType)
	switch fileType {
	case FileTypeTestCases:
		action, handleErr = uc.handleTestCases(ctx, p, input, now)
	case FileTypeSolution:
		action, handleErr = uc.handleSolution(ctx, p, input, now)
	case FileTypeChecker:
		action, handleErr = uc.handleChecker(ctx, p, input, now)
	case FileTypeValidator:
		action, handleErr = uc.handleValidator(ctx, p, input, now)
	default:
		slog.WarnContext(ctx, "invalid file type provided", "file_type", input.FileType, "slug", p.Slug().String())
		return nil, apperror.NewBadRequest(ErrCodeProblemInvalidFileType, "Invalid file type. Allowed: testCases, solution, checker, validator")
	}

	if handleErr != nil {
		uc.cleanupFiles(ctx, action.rollbackFiles)
		return nil, handleErr
	}

	if err := uc.repo.Save(ctx, p); err != nil {
		uc.cleanupFiles(ctx, action.rollbackFiles)
		return nil, err
	}

	uc.cleanupFiles(ctx, action.cleanupFiles)
	uc.cleanupPrefix(ctx, action.cleanupPrefix)

	return &UploadProblemFilesOutput{
		Message:  "File uploaded successfully",
		FileType: input.FileType,
		FileName: input.FileName,
		Problem:  problemToDTO(p),
	}, nil
}

func (uc *UploadProblemFilesUseCase) cleanupFiles(ctx context.Context, keys []string) {
	for _, key := range keys {
		_ = uc.storage.DeleteFile(ctx, key)
	}
}

func (uc *UploadProblemFilesUseCase) cleanupPrefix(ctx context.Context, prefix string) {
	if prefix == "" {
		return
	}
	_ = uc.storage.DeleteFilesWithPrefix(ctx, prefix)
}

func (uc *UploadProblemFilesUseCase) handleTestCases(ctx context.Context, p *problem.Problem, input UploadProblemFilesInput, now time.Time) (fileAction, error) {
	sampleFiles, err := uc.zipParser.ParseTestCasesZip(ctx, input.FileData)
	if err != nil {
		return fileAction{}, err
	}

	uploadInstanceID := uuid.New().String()
	basePath := fmt.Sprintf("problems/%s/%s/%s", p.Slug().String(), FileTypeTestCases, uploadInstanceID)
	zipKey := fmt.Sprintf("%s/testcases.zip", basePath)

	allNewKeys := make([]string, 0, len(sampleFiles)+1)
	allNewKeys = append(allNewKeys, zipKey)
	for _, file := range sampleFiles {
		allNewKeys = append(allNewKeys, fmt.Sprintf("%s/%s", basePath, file.Path))
	}

	if err := uc.storage.UploadFile(ctx, zipKey, input.FileData); err != nil {
		return fileAction{rollbackFiles: allNewKeys}, err
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(uc.settings.UploadMaxConcurrency())

	for _, file := range sampleFiles {
		file := file
		g.Go(func() error {
			destinationPath := fmt.Sprintf("%s/%s", basePath, file.Path)
			if err := uc.storage.UploadFile(gCtx, destinationPath, file.Content); err != nil {
				return err
			}
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return fileAction{rollbackFiles: allNewKeys}, err
	}

	action := fileAction{rollbackFiles: allNewKeys}
	if p.TestCasesKey() != nil {
		action.cleanupPrefix = *p.TestCasesKey()
	}

	p.SetTestCases(basePath, now)
	return action, nil
}

func (uc *UploadProblemFilesUseCase) handleSolution(ctx context.Context, p *problem.Problem, input UploadProblemFilesInput, now time.Time) (fileAction, error) {
	if input.FileName == "" {
		return fileAction{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "fileName", Message: "Filename is required"},
		})
	}

	cleanName := filepath.Base(input.FileName)
	lang, err := uc.getLanguageForFile(cleanName, FileTypeSolution)
	if err != nil {
		return fileAction{}, err
	}

	fileKey := fmt.Sprintf("problems/%s/%s/%s", p.Slug().String(), FileTypeSolution, cleanName)
	solutionObj, err := problem.NewSolutionFile(cleanName, fileKey, lang)
	if err != nil {
		return fileAction{}, err
	}

	if err := uc.storage.UploadFile(ctx, fileKey, input.FileData); err != nil {
		return fileAction{}, err
	}

	action := fileAction{rollbackFiles: []string{fileKey}}

	if old := p.AddSolution(solutionObj, now); old != nil && old.FileKey() != fileKey {
		action.cleanupFiles = []string{old.FileKey()}
	}

	return action, nil
}

func (uc *UploadProblemFilesUseCase) handleChecker(ctx context.Context, p *problem.Problem, input UploadProblemFilesInput, now time.Time) (fileAction, error) {
	return uc.handleVerifier(ctx, p, input, FileTypeChecker, p.Checker, func(f problem.JudgingFile) { p.SetChecker(f, now) }, now)
}

func (uc *UploadProblemFilesUseCase) handleValidator(ctx context.Context, p *problem.Problem, input UploadProblemFilesInput, now time.Time) (fileAction, error) {
	return uc.handleVerifier(ctx, p, input, FileTypeValidator, p.Validator, func(f problem.JudgingFile) { p.SetValidator(f, now) }, now)
}

func (uc *UploadProblemFilesUseCase) handleVerifier(
	ctx context.Context,
	p *problem.Problem,
	input UploadProblemFilesInput,
	fileType string,
	getVerifier func() *problem.JudgingFile,
	setVerifier func(problem.JudgingFile),
	now time.Time,
) (fileAction, error) {
	if input.FileName == "" {
		return fileAction{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "fileName", Message: "Filename is required"},
		})
	}

	cleanName := filepath.Base(input.FileName)
	lang, err := uc.getLanguageForFile(cleanName, fileType)
	if err != nil {
		return fileAction{}, err
	}

	fileKey := fmt.Sprintf("problems/%s/%s/%s", p.Slug().String(), fileType, cleanName)
	verifierObj, err := problem.NewVerifierFile(cleanName, fileKey, lang)
	if err != nil {
		return fileAction{}, err
	}

	current := getVerifier()
	action := fileAction{}
	if current == nil || current.FileKey() != fileKey {
		action.rollbackFiles = []string{fileKey}
	}
	if current != nil && current.FileKey() != fileKey {
		action.cleanupFiles = []string{current.FileKey()}
	}

	if err := uc.storage.UploadFile(ctx, fileKey, input.FileData); err != nil {
		return fileAction{rollbackFiles: action.rollbackFiles}, err
	}

	setVerifier(verifierObj)
	return action, nil
}

func (uc *UploadProblemFilesUseCase) getLanguageForFile(cleanName, fileType string) (string, error) {
	ext := strings.ToLower(filepath.Ext(cleanName))
	lang, ok := uc.settings.LanguageByExtension(ext)
	if !ok {
		msg := fmt.Sprintf("Unsupported %s file type", fileType)
		return "", apperror.NewValidation([]apperror.FieldError{
			{Field: "file", Message: msg},
		})
	}
	return lang, nil
}
