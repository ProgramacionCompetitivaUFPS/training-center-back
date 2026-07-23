package submission

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type SubmitSolutionInput struct {
	CurrentUser appshared.CurrentUser
	ProblemSlug string
	Language    string
	Compiler    string
	FileName    string
	FileData    []byte
	SubmittedAt time.Time
}

type SubmitSolutionOutput struct {
	ID           string
	Status       string
	SubmittedAt  time.Time
	ProblemSlug  string
	ProblemTitle string
	Language     string
	Compiler     string
	FileSize     int
	FileHash     string
}

type SubmitSolutionUseCase struct {
	problemProvider  ProblemProvider
	submissionRepo   domainSubmission.Repository
	sourceStorage    SourceStorage
	submissionQueue  SubmissionQueue
	maxFileSizeBytes int64
	rateLimitSeconds int
}

func NewSubmitSolutionUseCase(
	problemProvider ProblemProvider,
	submissionRepo domainSubmission.Repository,
	sourceStorage SourceStorage,
	submissionQueue SubmissionQueue,
	maxFileSizeBytes int64,
	rateLimitSeconds int,
) *SubmitSolutionUseCase {
	return &SubmitSolutionUseCase{
		problemProvider:  problemProvider,
		submissionRepo:   submissionRepo,
		sourceStorage:    sourceStorage,
		submissionQueue:  submissionQueue,
		maxFileSizeBytes: maxFileSizeBytes,
		rateLimitSeconds: rateLimitSeconds,
	}
}

func (uc *SubmitSolutionUseCase) Execute(ctx context.Context, in SubmitSolutionInput) (*SubmitSolutionOutput, error) {
	// 1. Validate file size before anything else
	if int64(len(in.FileData)) > uc.maxFileSizeBytes {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeFileTooLarge,
			fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", uc.maxFileSizeBytes))
	}

	// 2-3. Validate language, compiler, and file extension
	langVO, ext, err := validateLanguage(in.Language, in.Compiler, in.FileName)
	if err != nil {
		return nil, err
	}

	// 4. Get problem
	problem, err := uc.problemProvider.GetProblemBySlug(ctx, in.ProblemSlug)
	if err != nil {
		return nil, err
	}

	// 5. Problem must be PUBLISHED
	if !problem.IsPublished {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeProblemNotPublished,
			"only PUBLISHED problems can receive submissions")
	}

	// 6. Accessibility check
	if !problem.IsPublic {
		isModifier := false
		for _, mid := range problem.ModifierIDs {
			if mid == in.CurrentUser.ID {
				isModifier = true
				break
			}
		}
		if !isModifier {
			return nil, apperror.NewForbidden(domainSubmission.ErrCodeProblemNotAccessible,
				"only modifiers can submit to PRIVATE problems")
		}
	}

	userID := in.CurrentUser.ID

	// 7. Compute SHA256 hash
	sum := sha256.Sum256(in.FileData)
	fileHash := fmt.Sprintf("%x", sum)

	// 8. Duplicate check (same hash + user + problem)
	isDup, err := uc.submissionRepo.ExistsByHashAndUserAndProblem(ctx, fileHash, userID, problem.ID, nil)
	if err != nil {
		return nil, err
	}
	if isDup {
		return nil, apperror.NewConflict(domainSubmission.ErrCodeDuplicateSubmission,
			"this file has already been submitted to this problem")
	}

	// 9. Rate limit: last submission to same problem < rateLimitSeconds ago
	last, err := uc.submissionRepo.FindLastByUserAndProblem(ctx, userID, problem.ID, nil)
	if err != nil {
		var ae *apperror.AppError
		if !errors.As(err, &ae) || ae.Kind != apperror.KindNotFound {
			return nil, err
		}
	}
	if last != nil {
		elapsed := in.SubmittedAt.Sub(last.SubmittedAt())
		if elapsed.Seconds() < float64(uc.rateLimitSeconds) {
			return nil, apperror.NewTooManyRequests(ErrCodeRateLimitExceeded,
				fmt.Sprintf("please wait before submitting again (rate limit: %d second)", uc.rateLimitSeconds),
				uc.rateLimitSeconds)
		}
	}

	// 10. Build storage path and upload file
	submissionID := uuid.New().String()
	storagePath := fmt.Sprintf("%s/%s/general/%s%s", problem.ID, userID, submissionID, ext)

	if err := uc.sourceStorage.Upload(ctx, storagePath, in.FileData); err != nil {
		return nil, err
	}

	// 11. Create and persist submission
	sub, err := domainSubmission.NewSubmission(
		submissionID,
		problem.ID,
		shared.RestoreUserID(userID),
		nil,
		nil,
		langVO,
		in.Compiler,
		storagePath,
		fileHash,
		len(in.FileData),
		problem.Title,
		problem.Slug,
		in.SubmittedAt,
	)
	if err != nil {
		return nil, err
	}

	if err := uc.submissionRepo.Save(ctx, sub); err != nil {
		if delErr := uc.sourceStorage.Delete(ctx, storagePath); delErr != nil {
			slog.ErrorContext(ctx, "submission: failed to delete orphaned file after save error",
				"path", storagePath, "error", delErr)
		}
		return nil, err
	}

	// 12. Publish to judging queue (fire-and-forget; log error but don't fail)
	if err := uc.submissionQueue.Publish(ctx, SubmissionQueueMessage{
		SubmissionID: submissionID,
		Priority:     QueuePriorityPractice,
		EnqueuedAt:   in.SubmittedAt,
		Metadata: SubmissionQueueMetadata{
			ContestID: nil,
			ProblemID: problem.ID,
			UserID:    userID,
			Language:  langVO.String(),
		},
	}); err != nil {
		slog.ErrorContext(ctx, "failed to publish submission to queue", "submission_id", submissionID, "error", err)
	}

	return &SubmitSolutionOutput{
		ID:           submissionID,
		Status:       sub.Status().String(),
		SubmittedAt:  sub.SubmittedAt(),
		ProblemSlug:  problem.Slug,
		ProblemTitle: problem.Title,
		Language:     langVO.String(),
		Compiler:     in.Compiler,
		FileSize:     len(in.FileData),
		FileHash:     fileHash,
	}, nil
}
