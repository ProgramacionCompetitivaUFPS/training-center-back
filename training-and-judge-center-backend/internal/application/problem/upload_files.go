package problem

import (
	"context"
	"encoding/json"
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

	// Verifier sources are stored under the same name the judge writes them as
	// inside the sandbox, so the bucket can be navigated without the database.
	verifierBaseNameChecker   = "Checker"
	verifierBaseNameValidator = "Validator"
)

type UploadProblemFilesInput struct {
	Slug     string
	FileType string
	FileName string
	FileData []byte
	// ConfirmSlug is required to replace testCases or checker of a published problem in an active contest.
	ConfirmSlug string
	CurrentUser appshared.CurrentUser
}

type UploadProblemFilesOutput struct {
	Message  string
	FileType string
	FileName string
	Problem  ProblemDTO
	// ValidationID is set when the change is staged and not in effect yet.
	ValidationID     string
	AffectedContests []ContestRef
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
	contestChecker ActiveContestChecker
	contestLister  AffectedContestsProvider
	queue          ValidationQueue
}

func NewUploadProblemFilesUseCase(
	repo problem.Repository,
	validationRepo problem.ProblemValidationRepository,
	storage ProblemFileRepository,
	zipParser ZipParser,
	settings problem.PlatformSettings,
	contestChecker ActiveContestChecker,
	contestLister AffectedContestsProvider,
	queue ValidationQueue,
) *UploadProblemFilesUseCase {
	return &UploadProblemFilesUseCase{
		repo:           repo,
		validationRepo: validationRepo,
		storage:        storage,
		zipParser:      zipParser,
		settings:       settings,
		contestChecker: contestChecker,
		contestLister:  contestLister,
		queue:          queue,
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
		return uc.executeOnPublished(ctx, p, input)
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
	zipKey := TestCasesZipKey(basePath)

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
	return uc.handleVerifier(ctx, p, input, FileTypeChecker, verifierBaseNameChecker, "", p.Checker, func(f problem.JudgingFile) { p.SetChecker(f, now) }, now)
}

func (uc *UploadProblemFilesUseCase) handleValidator(ctx context.Context, p *problem.Problem, input UploadProblemFilesInput, now time.Time) (fileAction, error) {
	return uc.handleVerifier(ctx, p, input, FileTypeValidator, verifierBaseNameValidator, "", p.Validator, func(f problem.JudgingFile) { p.SetValidator(f, now) }, now)
}

func (uc *UploadProblemFilesUseCase) handleVerifier(
	ctx context.Context,
	p *problem.Problem,
	input UploadProblemFilesInput,
	fileType string,
	baseName string,
	// stageID keeps the checker apart from the live one, under the judging update that owns it.
	stageID string,
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

	fileKey := fmt.Sprintf("problems/%s/%s/%s%s", p.Slug().String(), fileType, baseName, filepath.Ext(cleanName))
	if stageID != "" {
		fileKey = stagedCheckerPrefix(p.Slug().String(), stageID) + baseName + filepath.Ext(cleanName)
	}
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

// executeOnPublished stages a new testCases or checker for a published problem in an active contest; a worker validates it before it replaces the live one.
func (uc *UploadProblemFilesUseCase) executeOnPublished(ctx context.Context, p *problem.Problem, input UploadProblemFilesInput) (*UploadProblemFilesOutput, error) {
	errPublished := apperror.NewBadRequest(ErrCodeProblemIsPublished, "Cannot upload files to a published problem. Unpublish first.")

	fileType := strings.ToLower(input.FileType)
	if fileType != FileTypeTestCases && fileType != FileTypeChecker {
		return nil, errPublished
	}

	inActive, err := uc.contestChecker.IsProblemInActiveContest(ctx, p.ID())
	if err != nil {
		return nil, err
	}
	if !inActive {
		return nil, errPublished
	}

	if p.AuthorID() != shared.RestoreUserID(input.CurrentUser.ID) && !input.CurrentUser.IsAdmin() {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author or Admin can change the judging files of a problem in an active contest")
	}
	if input.ConfirmSlug != p.Slug().String() {
		return nil, apperror.NewBadRequest(problem.ErrCodeSlugMismatch, "Confirmation slug does not match the problem slug")
	}
	if err := ensureNoActiveValidation(ctx, uc.validationRepo, p.ID()); err != nil {
		return nil, err
	}

	// Taken before staging mutates p in memory, so the response shows what is live.
	liveProblem := problemToDTO(p)

	validationID := uuid.New().String()
	now := time.Now()

	action, change, err := uc.stageJudgingChange(ctx, p, input, fileType, validationID, now)
	if err != nil {
		uc.cleanupFiles(ctx, action.rollbackFiles)
		return nil, err
	}

	v, err := problem.NewJudgingUpdateValidation(validationID, p.ID(), shared.RestoreUserID(input.CurrentUser.ID), change, now)
	if err != nil {
		uc.cleanupFiles(ctx, action.rollbackFiles)
		return nil, err
	}
	if err := uc.validationRepo.Save(ctx, v); err != nil {
		uc.cleanupFiles(ctx, action.rollbackFiles)
		return nil, err
	}

	if err := uc.queue.Publish(ctx, ValidationQueueMessage{
		ValidationID: v.ID(),
		ProblemID:    p.ID(),
		Slug:         p.Slug().String(),
		Priority:     QueuePriorityPublishValidation,
		EnqueuedAt:   now,
	}); err != nil {
		uc.abandonJudgingUpdate(ctx, v, action, now)
		return nil, err
	}

	// The update is already queued, and the adapter logged a failure here: the list is only a hint.
	affected, _ := uc.contestLister.ListOpenContestsOfProblem(ctx, p.ID())

	return &UploadProblemFilesOutput{
		Message:          "Judging update accepted; it takes effect once it passes validation",
		FileType:         input.FileType,
		FileName:         input.FileName,
		Problem:          liveProblem,
		ValidationID:     v.ID(),
		AffectedContests: affected,
	}, nil
}

// stageJudgingChange reuses the draft upload handlers on the in-memory problem, which is never saved.
func (uc *UploadProblemFilesUseCase) stageJudgingChange(
	ctx context.Context,
	p *problem.Problem,
	input UploadProblemFilesInput,
	fileType, validationID string,
	now time.Time,
) (fileAction, problem.PendingJudgingChange, error) {
	if fileType == FileTypeTestCases {
		action, err := uc.handleTestCases(ctx, p, input, now)
		if err != nil {
			return action, problem.PendingJudgingChange{}, err
		}
		change, err := problem.NewPendingJudgingChange(p.TestCasesKey(), nil)
		return action, change, err
	}

	action, err := uc.handleVerifier(ctx, p, input, FileTypeChecker, verifierBaseNameChecker, validationID, p.Checker,
		func(f problem.JudgingFile) { p.SetChecker(f, now) }, now)
	if err != nil {
		return action, problem.PendingJudgingChange{}, err
	}
	change, err := problem.NewPendingJudgingChange(nil, p.Checker())
	return action, change, err
}

// abandonJudgingUpdate closes a ticket that never reached the queue so it does not hold the problem's validation slot.
func (uc *UploadProblemFilesUseCase) abandonJudgingUpdate(ctx context.Context, v *problem.ProblemValidation, action fileAction, now time.Time) {
	uc.cleanupFiles(ctx, action.rollbackFiles)
	resultJSON, _ := json.Marshal(ValidationReport{ValidationLogs: []string{"✗ Could not queue the judging update"}})
	if err := v.Start(now); err != nil {
		return
	}
	_ = v.MarkSystemError(string(resultJSON), now)
	if err := uc.validationRepo.Save(ctx, v); err != nil {
		slog.ErrorContext(ctx, "upload_files: could not close an unqueued judging update", "error", err, "validation_id", v.ID())
	}
}
