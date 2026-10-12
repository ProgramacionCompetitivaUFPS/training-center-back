package problem

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

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

	affected, err := uc.contestLister.ListOpenContestsOfProblem(ctx, p.ID())
	if err != nil {
		slog.WarnContext(ctx, "upload_files: could not list the contests a judging update affects", "error", err, "problem_id", p.ID())
	}

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
