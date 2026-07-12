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
	exitCodeMLE = 137 // OOM killer sent SIGKILL (128 + 9) when cgroup memory limit was exceeded
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
}

func NewJudgeSubmissionUseCase(
	submissionUpdater SubmissionUpdater,
	sourceCodeDownloader SourceCodeDownloader,
	problemProvider ProblemProvider,
	testCaseProvider TestCaseProvider,
	executor Executor,
	outputChecker OutputChecker,
	txManager appshared.TransactionManager,
) *JudgeSubmissionUseCase {
	return &JudgeSubmissionUseCase{
		submissionUpdater:    submissionUpdater,
		sourceCodeDownloader: sourceCodeDownloader,
		problemProvider:      problemProvider,
		testCaseProvider:     testCaseProvider,
		executor:             executor,
		outputChecker:        outputChecker,
		txManager:            txManager,
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
		return err
	}

	limits, err := uc.problemProvider.GetLimits(ctx, sub.ProblemID())
	if err != nil {
		return err
	}

	testCases, err := uc.testCaseProvider.GetTestCases(ctx, sub.ProblemID())
	if err != nil {
		return err
	}

	session, err := uc.executor.BeginSession(ctx, sub.Language())
	if err != nil {
		_ = sub.MarkSystemError(now)
		return uc.persistVerdict(ctx, sub)
	}
	defer session.Close(ctx)

	compileResult, err := session.Compile(ctx, CompileRequest{
		Language:   sub.Language(),
		SourceCode: sourceCode,
	})
	if err != nil {
		_ = sub.MarkSystemError(now)
		return uc.persistVerdict(ctx, sub)
	}
	if !compileResult.Success {
		_ = sub.MarkCompilationError(compileResult.Log, now)
		return uc.persistVerdict(ctx, sub)
	}

	maxTimeMs, maxMemoryKb := 0, 0
	for _, tc := range testCases {
		runResult, err := session.RunTestCase(ctx, RunRequest{
			Input:       tc.Input,
			TimeLimitMs: limits.TimeLimitMs,
			MemoryKb:    limits.MemoryKb,
		})
		if err != nil {
			_ = sub.MarkSystemError(now)
			break
		}

		switch runResult.ExitCode {
		case exitCodeTLE:
			_ = sub.MarkTimeLimitExceeded(runResult.TimeMs, now)
		case exitCodeMLE:
			_ = sub.MarkMemoryLimitExceeded(runResult.MemoryKb, now)
		case 0:
			checkResult, err := uc.outputChecker.Check(ctx, CheckRequest{
				Input:            tc.Input,
				ExpectedOutput:   tc.ExpectedOutput,
				ContestantOutput: runResult.Output,
				CheckerPath:      limits.CheckerPath,
			})
			if err != nil {
				_ = sub.MarkSystemError(now)
				break
			}
			if !checkResult.Accepted {
				_ = sub.MarkWrongAnswer(runResult.TimeMs, runResult.MemoryKb, now)
				break
			}
			if runResult.TimeMs > maxTimeMs {
				maxTimeMs = runResult.TimeMs
			}
			if runResult.MemoryKb > maxMemoryKb {
				maxMemoryKb = runResult.MemoryKb
			}
			continue
		default:
			_ = sub.MarkRuntimeError(runResult.TimeMs, runResult.MemoryKb, now)
		}
		break
	}

	if sub.Status().IsRunning() {
		_ = sub.MarkAccepted(maxTimeMs, maxMemoryKb, now)
	}

	return uc.persistVerdict(ctx, sub)
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
