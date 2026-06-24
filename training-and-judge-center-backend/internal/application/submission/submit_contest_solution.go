package submission

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/shared"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type SubmitContestSolutionInput struct {
	CurrentUser appshared.CurrentUser
	GroupID     string
	ContestID   string
	ProblemSlug string
	Language    string
	Compiler    string
	FileName    string
	FileData    []byte
	SubmittedAt time.Time
}

type SubmitContestSolutionOutput struct {
	ID           string
	Status       string
	SubmittedAt  time.Time
	ProblemSlug  string
	ProblemTitle string
	ContestID    string
	ContestName  string
	Language     string
	Compiler     string
	FileSize     int
	FileHash     string
}

type SubmitContestSolutionUseCase struct {
	contestProvider  ContestSubmissionProvider
	standingResolver StandingIDResolver
	problemProvider  ProblemProvider
	submissionRepo   domainSubmission.Repository
	sourceStorage    SourceStorage
	submissionQueue  SubmissionQueue
	maxFileSizeBytes int64
	rateLimitSeconds int
}

func NewSubmitContestSolutionUseCase(
	contestProvider ContestSubmissionProvider,
	standingResolver StandingIDResolver,
	problemProvider ProblemProvider,
	submissionRepo domainSubmission.Repository,
	sourceStorage SourceStorage,
	submissionQueue SubmissionQueue,
	maxFileSizeBytes int64,
	rateLimitSeconds int,
) *SubmitContestSolutionUseCase {
	return &SubmitContestSolutionUseCase{
		contestProvider:  contestProvider,
		standingResolver: standingResolver,
		problemProvider:  problemProvider,
		submissionRepo:   submissionRepo,
		sourceStorage:    sourceStorage,
		submissionQueue:  submissionQueue,
		maxFileSizeBytes: maxFileSizeBytes,
		rateLimitSeconds: rateLimitSeconds,
	}
}

func (uc *SubmitContestSolutionUseCase) Execute(ctx context.Context, in SubmitContestSolutionInput) (*SubmitContestSolutionOutput, error) {
	// 1. Validate file size before anything else
	if int64(len(in.FileData)) > uc.maxFileSizeBytes {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeFileTooLarge,
			fmt.Sprintf("file size exceeds maximum allowed size of %d bytes", uc.maxFileSizeBytes))
	}

	// 2. Validate language
	langVO, err := domainSubmission.NewLanguage(in.Language)
	if err != nil {
		return nil, err
	}

	// 3. Validate compiler and file extension match language
	cfg, ok := languageConfig[in.Language]
	if !ok {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeCompilerMismatch, "unsupported language")
	}
	if in.Compiler != cfg.compiler {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeCompilerMismatch, "compiler does not match the selected language")
	}
	fileExt := strings.ToLower(filepath.Ext(in.FileName))
	validExt := false
	for _, ext := range cfg.extensions {
		if fileExt == ext {
			validExt = true
			break
		}
	}
	if !validExt {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeCompilerMismatch, "file extension does not match the selected language")
	}

	// 4. Get contest info (validates contest exists and belongs to the group)
	contest, err := uc.contestProvider.GetContestForSubmission(ctx, in.GroupID, in.ContestID)
	if err != nil {
		return nil, err
	}

	// 5. Resolve standingID (validates registration)
	standingID, registered, err := uc.standingResolver.ResolveStandingID(ctx, in.ContestID, in.CurrentUser.ID)
	if err != nil {
		return nil, err
	}
	if !registered {
		return nil, apperror.NewForbidden(domainSubmission.ErrCodeNotRegistered,
			"you must be registered to the contest to submit solutions")
	}

	// 6. Validate contest status using the captured submittedAt timestamp
	var queuePriority int
	if in.SubmittedAt.Before(contest.StartTime) {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeContestNotStarted,
			"submissions are only allowed during ACTIVE contests or postcompetition")
	}
	if in.SubmittedAt.After(contest.EndTime) {
		if !contest.EnablePostContest {
			return nil, apperror.NewBadRequest(domainSubmission.ErrCodeContestFinished,
				"the contest has ended and postcompetition is not enabled")
		}
		queuePriority = QueuePriorityPostContest
	} else {
		queuePriority = QueuePriorityContest
	}

	// 7. Get problem by slug
	problem, err := uc.problemProvider.GetProblemBySlug(ctx, in.ProblemSlug)
	if err != nil {
		return nil, err
	}

	// 8. Problem must be PUBLISHED
	if !problem.IsPublished {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeProblemNotPublished,
			"only PUBLISHED problems can receive submissions")
	}

	// 9. Problem must be part of the contest
	inContest := false
	for _, pid := range contest.ProblemIDs {
		if pid == problem.ID {
			inContest = true
			break
		}
	}
	if !inContest {
		return nil, apperror.NewBadRequest(domainSubmission.ErrCodeProblemNotInContest,
			"this problem is not part of this contest")
	}

	userID := in.CurrentUser.ID

	// 10. Compute SHA256 hash
	sum := sha256.Sum256(in.FileData)
	fileHash := fmt.Sprintf("%x", sum)

	// 11. Duplicate check scoped to this contest
	isDup, err := uc.submissionRepo.ExistsByHashAndUserAndProblem(ctx, fileHash, userID, problem.ID, &in.ContestID)
	if err != nil {
		return nil, err
	}
	if isDup {
		return nil, apperror.NewConflict(domainSubmission.ErrCodeDuplicateSubmission,
			"this file has already been submitted to this contest problem")
	}

	// 12. Rate limit scoped to this contest
	last, err := uc.submissionRepo.FindLastByUserAndProblem(ctx, userID, problem.ID, &in.ContestID)
	if err != nil {
		var ae *apperror.AppError
		if !errors.As(err, &ae) || ae.Kind != apperror.KindNotFound {
			return nil, err
		}
	}
	if last != nil {
		elapsed := in.SubmittedAt.Sub(last.SubmittedAt())
		if elapsed.Seconds() < float64(uc.rateLimitSeconds) {
			return nil, apperror.NewTooManyRequests(domainSubmission.ErrCodeRateLimitExceeded,
				fmt.Sprintf("please wait before submitting again (rate limit: %d second)", uc.rateLimitSeconds),
				uc.rateLimitSeconds)
		}
	}

	// 13. Upload to contest-specific storage path
	submissionID := uuid.New().String()
	storagePath := fmt.Sprintf("%s/%s/%s/%s%s", problem.ID, userID, in.ContestID, submissionID, cfg.ext)

	if err := uc.sourceStorage.Upload(ctx, storagePath, in.FileData); err != nil {
		return nil, err
	}

	// 14. Create and persist submission
	contestID := in.ContestID
	sub, err := domainSubmission.NewSubmission(
		submissionID,
		problem.ID,
		shared.RestoreUserID(userID),
		&contestID,
		&standingID,
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

	// 15. Publish to judging queue (fire-and-forget; log error but don't fail)
	if err := uc.submissionQueue.Publish(ctx, SubmissionQueueMessage{
		SubmissionID: submissionID,
		Priority:     queuePriority,
		EnqueuedAt:   in.SubmittedAt,
		Metadata: SubmissionQueueMetadata{
			ContestID: &contestID,
			ProblemID: problem.ID,
			UserID:    userID,
			Language:  langVO.String(),
		},
	}); err != nil {
		slog.ErrorContext(ctx, "failed to publish contest submission to queue",
			"submission_id", submissionID, "error", err)
	}

	return &SubmitContestSolutionOutput{
		ID:           submissionID,
		Status:       sub.Status().String(),
		SubmittedAt:  sub.SubmittedAt(),
		ProblemSlug:  problem.Slug,
		ProblemTitle: problem.Title,
		ContestID:    contest.ID,
		ContestName:  contest.Name,
		Language:     langVO.String(),
		Compiler:     in.Compiler,
		FileSize:     len(in.FileData),
		FileHash:     fileHash,
	}, nil
}
