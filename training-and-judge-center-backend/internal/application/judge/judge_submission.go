package judge

import (
	"context"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// Exit codes produced by the Linux timeout(1) command and the kernel OOM killer.
const (
	exitCodeTLE = 124 // timeout(1) killed the process after the time limit
	// exitCodeMLE is the cgroup OOM killer's SIGKILL (128 + 9). Java never gets
	// there on its own — the JVM enforces its own heap cap and exits 1 — so its
	// runCmd in judge_config.yaml carries OnOutOfMemoryError to SIGKILL itself.
	// Removing that flag silently turns every Java MLE back into a runtime error.
	exitCodeMLE = 137
)

type JudgeSubmissionInput struct {
	SubmissionID submission.SubmissionID
}

type JudgeSubmissionUseCase struct {
	submissionUpdater    SubmissionUpdater
	sourceCodeDownloader SourceCodeDownloader
	problemProvider      ProblemProvider
	testCaseProvider     TestCaseProvider
	executor             Executor
	outputChecker        OutputChecker
	txManager            appshared.TransactionManager
	retry                RetryConfig
	sleep                func(time.Duration)
}

func NewJudgeSubmissionUseCase(
	submissionUpdater SubmissionUpdater,
	sourceCodeDownloader SourceCodeDownloader,
	problemProvider ProblemProvider,
	testCaseProvider TestCaseProvider,
	executor Executor,
	outputChecker OutputChecker,
	txManager appshared.TransactionManager,
	retry RetryConfig,
) *JudgeSubmissionUseCase {
	return &JudgeSubmissionUseCase{
		submissionUpdater:    submissionUpdater,
		sourceCodeDownloader: sourceCodeDownloader,
		problemProvider:      problemProvider,
		testCaseProvider:     testCaseProvider,
		executor:             executor,
		outputChecker:        outputChecker,
		txManager:            txManager,
		retry:                retry,
		sleep:                time.Sleep,
	}
}

func (uc *JudgeSubmissionUseCase) Execute(ctx context.Context, in JudgeSubmissionInput) error {
	now := time.Now()

	sub, err := uc.submissionUpdater.GetByID(ctx, in.SubmissionID)
	if err != nil {
		return err
	}

	if !sub.Status().IsPending() {
		return nil
	}

	if err := sub.Start(now); err != nil {
		return err
	}
	if err := uc.submissionUpdater.Update(ctx, sub); err != nil {
		return err
	}

	sourceCode, err := uc.sourceCodeDownloader.Download(ctx, sub.SourceCodePath())
	if err != nil {
		slog.ErrorContext(ctx, "failed to download source code", "error", err)
		_ = sub.MarkSystemError(now)
		return uc.persistVerdict(ctx, sub)
	}

	limits, err := uc.problemProvider.GetLimits(ctx, sub.ProblemID())
	if err != nil {
		slog.ErrorContext(ctx, "failed to get problem limits", "error", err)
		_ = sub.MarkSystemError(now)
		return uc.persistVerdict(ctx, sub)
	}

	testCases, err := uc.testCaseProvider.GetTestCases(ctx, sub.ProblemID())
	if err != nil {
		slog.ErrorContext(ctx, "failed to get test cases", "error", err)
		_ = sub.MarkSystemError(now)
		return uc.persistVerdict(ctx, sub)
	}

	var judgeErr error
	for attempt := 0; attempt < uc.retry.MaxAttempts; attempt++ {
		if attempt > 0 {
			uc.sleep(uc.retry.BackoffBase * time.Duration(1<<uint(attempt-1)))
		}
		shouldRetry, err := uc.judgeAttempt(ctx, sub, sourceCode, limits, testCases, now)
		if !shouldRetry {
			judgeErr = err
			break
		}
		judgeErr = err
		slog.ErrorContext(ctx, "judge attempt failed, will retry",
			"attempt", attempt+1,
			"max", uc.retry.MaxAttempts,
			"error", err,
		)
	}
	if judgeErr != nil {
		_ = sub.MarkSystemError(now)
	}
	return uc.persistVerdict(ctx, sub)
}

// judgeAttempt runs one whole judging attempt. It returns (true, err) when infra
// failed and sub carries no verdict, and (false, nil) once sub has a final one.
func (uc *JudgeSubmissionUseCase) judgeAttempt(
	ctx context.Context,
	sub *submission.Submission,
	sourceCode []byte,
	limits ProblemLimits,
	testCases []TestCase,
	now time.Time,
) (bool, error) {
	session, err := uc.executor.BeginSession(ctx, sub.Language(), limits.MemoryKb)
	if err != nil {
		return true, err
	}
	defer session.Close(ctx)

	compileResult, err := session.Compile(ctx, CompileRequest{
		Language:   sub.Language(),
		SourceCode: sourceCode,
	})
	if err != nil {
		return true, err
	}
	if !compileResult.Success {
		_ = sub.MarkCompilationError(compileResult.Log, now)
		return false, nil
	}

	// Opened after compiling: a submission that does not build produces no output
	// to check, so no checker container is held during the compile.
	checkerSession, err := uc.outputChecker.BeginChecking(ctx, limits.CheckerPath, limits.CheckerLanguage)
	if err != nil {
		return true, err
	}
	defer checkerSession.Close(ctx)

	maxTimeMs, maxMemoryKb := 0, 0
	for _, tc := range testCases {
		runResult, err := session.RunTestCase(ctx, RunRequest{
			Input:       tc.Input,
			TimeLimitMs: limits.TimeLimitMs,
		})
		if err != nil {
			return true, err
		}

		switch runResult.ExitCode {
		case exitCodeTLE:
			_ = sub.MarkTimeLimitExceeded(runResult.TimeMs, now)
			return false, nil
		case exitCodeMLE:
			_ = sub.MarkMemoryLimitExceeded(runResult.MemoryKb, now)
			return false, nil
		case 0:
			if runResult.TimeMs > limits.TimeLimitMs {
				_ = sub.MarkTimeLimitExceeded(runResult.TimeMs, now)
				return false, nil
			}
			checkResult, err := checkerSession.Check(ctx, CheckRequest{
				Input:            tc.Input,
				ExpectedOutput:   tc.ExpectedOutput,
				ContestantOutput: runResult.Output,
			})
			if err != nil {
				return true, err
			}
			if !checkResult.Accepted {
				_ = sub.MarkWrongAnswer(runResult.TimeMs, runResult.MemoryKb, now)
				return false, nil
			}
			if runResult.TimeMs > maxTimeMs {
				maxTimeMs = runResult.TimeMs
			}
			if runResult.MemoryKb > maxMemoryKb {
				maxMemoryKb = runResult.MemoryKb
			}
		default:
			_ = sub.MarkRuntimeError(runResult.TimeMs, runResult.MemoryKb, now)
			return false, nil
		}
	}

	_ = sub.MarkAccepted(maxTimeMs, maxMemoryKb, now)
	return false, nil
}

func (uc *JudgeSubmissionUseCase) persistVerdict(ctx context.Context, sub *submission.Submission) error {
	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.submissionUpdater.Update(txCtx, sub); err != nil {
			slog.ErrorContext(txCtx, "failed to update submission verdict", "error", err)
			return apperror.NewInternal()
		}
		return nil
	})
}
