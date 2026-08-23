package judge

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type ValidateSolutionsInput struct {
	ProblemID string
}

type FailureKind string

const (
	FailureCompileError        FailureKind = "COMPILE_ERROR"
	FailureWrongAnswer         FailureKind = "WRONG_ANSWER"
	FailureTimeLimitExceeded   FailureKind = "TIME_LIMIT_EXCEEDED"
	FailureMemoryLimitExceeded FailureKind = "MEMORY_LIMIT_EXCEEDED"
	FailureOutputLimitExceeded FailureKind = "OUTPUT_LIMIT_EXCEEDED"
	FailureRuntimeError        FailureKind = "RUNTIME_ERROR"
)

// maxFailurePreviewBytes bounds CompileLog/Expected/Actual on a
// SolutionFailure. These end up stored in Postgres and returned over HTTP —
// a multi-megabyte test case output has no business showing up whole in
// either place, a short preview is enough for a human to see what went wrong.
const maxFailurePreviewBytes = 2000

func truncatePreview(data []byte) []byte {
	if len(data) <= maxFailurePreviewBytes {
		return data
	}
	preview := make([]byte, maxFailurePreviewBytes)
	copy(preview, data[:maxFailurePreviewBytes])
	return append(preview, []byte("... (truncated)")...)
}

// SolutionFailure describes the first problem ValidateSolutionsUseCase found,
// across every solution it tried. Only the fields relevant to Kind are set.
type SolutionFailure struct {
	FileKey        string
	Kind           FailureKind
	CompileLog     string // set only when Kind == FailureCompileError
	TestCase       string // set for every kind except FailureCompileError
	TimeMs         int    // set only when Kind == FailureTimeLimitExceeded
	TimeLimitMs    int    // set only when Kind == FailureTimeLimitExceeded
	Expected       []byte // set only when Kind == FailureWrongAnswer
	Actual         []byte // set only when Kind == FailureWrongAnswer
	CheckerMessage string // why the checker rejected; set only when Kind == FailureWrongAnswer
}

type ValidateSolutionsOutput struct {
	SampleCases     int
	SecretCases     int
	SolutionsTested int
	Passed          bool
	Failure         *SolutionFailure // nil when Passed
}

type ValidateSolutionsUseCase struct {
	solutionProvider     SolutionProvider
	sourceCodeDownloader SourceCodeDownloader
	problemProvider      ProblemProvider
	testCaseProvider     TestCaseProvider
	executor             Executor
	outputChecker        OutputChecker
	retry                RetryConfig
	sleep                func(time.Duration)
}

func NewValidateSolutionsUseCase(
	solutionProvider SolutionProvider,
	sourceCodeDownloader SourceCodeDownloader,
	problemProvider ProblemProvider,
	testCaseProvider TestCaseProvider,
	executor Executor,
	outputChecker OutputChecker,
	retry RetryConfig,
) *ValidateSolutionsUseCase {
	return &ValidateSolutionsUseCase{
		solutionProvider:     solutionProvider,
		sourceCodeDownloader: sourceCodeDownloader,
		problemProvider:      problemProvider,
		testCaseProvider:     testCaseProvider,
		executor:             executor,
		outputChecker:        outputChecker,
		retry:                retry,
		sleep:                time.Sleep,
	}
}

// Execute compiles and runs every solution against every test case, stopping
// at the first one that doesn't compile or doesn't pass. It never touches a
// ProblemValidation ticket — the caller decides what a returned error or
// failure means for that ticket.
func (uc *ValidateSolutionsUseCase) Execute(ctx context.Context, in ValidateSolutionsInput) (*ValidateSolutionsOutput, error) {
	solutions, err := uc.solutionProvider.GetSolutions(ctx, in.ProblemID)
	if err != nil {
		return nil, err
	}
	if len(solutions) == 0 {
		return nil, apperror.NewInternal()
	}

	limits, err := uc.problemProvider.GetLimits(ctx, in.ProblemID)
	if err != nil {
		return nil, err
	}

	testCases, err := uc.testCaseProvider.GetTestCases(ctx, in.ProblemID)
	if err != nil {
		return nil, err
	}
	sampleCases, secretCases := countByGroup(testCases)

	for i, sol := range solutions {
		failure, err := uc.checkSolutionWithRetry(ctx, sol, limits, testCases)
		if err != nil {
			return nil, err
		}
		if failure != nil {
			return &ValidateSolutionsOutput{
				SampleCases:     sampleCases,
				SecretCases:     secretCases,
				SolutionsTested: i + 1,
				Passed:          false,
				Failure:         failure,
			}, nil
		}
	}

	return &ValidateSolutionsOutput{
		SampleCases:     sampleCases,
		SecretCases:     secretCases,
		SolutionsTested: len(solutions),
		Passed:          true,
	}, nil
}

// checkSolutionWithRetry retries a whole solution attempt (compile + every
// test case) on infra failures, same granularity as judgeAttempt's retry for
// submissions. A nil error with a nil failure means the solution passed
// everything; a nil error with a non-nil failure is a definitive verdict,
// not retried.
func (uc *ValidateSolutionsUseCase) checkSolutionWithRetry(ctx context.Context, sol Solution, limits ProblemLimits, testCases []TestCase) (*SolutionFailure, error) {
	var lastErr error
	for attempt := 0; attempt < uc.retry.MaxAttempts; attempt++ {
		if attempt > 0 {
			uc.sleep(uc.retry.BackoffBase * time.Duration(1<<uint(attempt-1)))
		}
		failure, err := uc.checkSolution(ctx, sol, limits, testCases)
		if err == nil {
			return failure, nil
		}
		lastErr = err
		slog.ErrorContext(ctx, "validate_solutions: attempt failed, will retry",
			"attempt", attempt+1,
			"max", uc.retry.MaxAttempts,
			"file_key", sol.FileKey,
			"error", err,
		)
	}
	return nil, lastErr
}

// checkSolution compiles and runs one solution against every test case,
// stopping at the first problem it finds. A nil, nil return means it passed
// everything. A non-nil error means infra failed (retryable); it is never
// combined with a non-nil failure.
func (uc *ValidateSolutionsUseCase) checkSolution(ctx context.Context, sol Solution, limits ProblemLimits, testCases []TestCase) (*SolutionFailure, error) {
	sourceCode, err := uc.sourceCodeDownloader.Download(ctx, sol.FileKey)
	if err != nil {
		return nil, err
	}

	// A fresh unguessable name per solution: the name is what isolates one
	// judging directory from another.
	judgingID := uuid.NewString()

	session, err := uc.executor.BeginSession(ctx, sol.Language, limits.MemoryKb, judgingID)
	if err != nil {
		return nil, err
	}
	defer session.Close(ctx)

	compileResult, err := session.Compile(ctx, CompileRequest{Language: sol.Language, SourceCode: sourceCode})
	if err != nil {
		return nil, err
	}
	if !compileResult.Success {
		return &SolutionFailure{FileKey: sol.FileKey, Kind: FailureCompileError, CompileLog: string(truncatePreview([]byte(compileResult.Log)))}, nil
	}

	// Opened after compiling: a solution that does not build produces no output
	// to check, so no checker container is held during the compile.
	checkerSession, err := uc.outputChecker.BeginChecking(ctx, limits.CheckerPath, limits.CheckerLanguage, judgingID)
	if err != nil {
		return nil, err
	}
	defer checkerSession.Close(ctx)

	for _, tc := range testCases {
		runResult, err := session.RunTestCase(ctx, RunRequest{
			Input:       tc.Input,
			TimeLimitMs: limits.TimeLimitMs,
		})
		if err != nil {
			return nil, err
		}

		// Ahead of the exit code: see judgeAttempt for why it cannot carry this.
		if runResult.OutputLimitExceeded {
			return &SolutionFailure{FileKey: sol.FileKey, Kind: FailureOutputLimitExceeded, TestCase: tc.Name}, nil
		}

		switch runResult.ExitCode {
		case exitCodeTLE:
			return &SolutionFailure{FileKey: sol.FileKey, Kind: FailureTimeLimitExceeded, TestCase: tc.Name, TimeMs: runResult.TimeMs, TimeLimitMs: limits.TimeLimitMs}, nil
		case exitCodeMLE:
			return &SolutionFailure{FileKey: sol.FileKey, Kind: FailureMemoryLimitExceeded, TestCase: tc.Name}, nil
		case 0:
			if runResult.TimeMs > limits.TimeLimitMs {
				return &SolutionFailure{FileKey: sol.FileKey, Kind: FailureTimeLimitExceeded, TestCase: tc.Name, TimeMs: runResult.TimeMs, TimeLimitMs: limits.TimeLimitMs}, nil
			}
			checkResult, err := checkerSession.Check(ctx, tc.ExpectedOutput)
			if err != nil {
				return nil, err
			}
			if !checkResult.Accepted {
				return &SolutionFailure{
					FileKey:        sol.FileKey,
					Kind:           FailureWrongAnswer,
					TestCase:       tc.Name,
					Expected:       truncatePreview(tc.ExpectedOutput),
					Actual:         truncatePreview(runResult.OutputPreview),
					CheckerMessage: checkResult.Message,
				}, nil
			}
		default:
			return &SolutionFailure{FileKey: sol.FileKey, Kind: FailureRuntimeError, TestCase: tc.Name}, nil
		}
	}

	return nil, nil
}

func countByGroup(testCases []TestCase) (sample, secret int) {
	for _, tc := range testCases {
		switch {
		case strings.HasPrefix(tc.Name, "sample/"):
			sample++
		case strings.HasPrefix(tc.Name, "secret/"):
			secret++
		}
	}
	return sample, secret
}
