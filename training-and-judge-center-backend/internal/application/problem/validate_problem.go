package problem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
)

type ValidateProblemInput struct {
	ValidationID string
	// Slug is only used to build a readable storage path for compiled
	// checker/validator artifacts — never to look up the ticket itself.
	Slug string
}

type ValidateProblemUseCase struct {
	validationRepo    problem.ProblemValidationRepository
	judgingPreparer   JudgingPreparer
	artifactWriter    JudgingArtifactWriter
	solutionValidator SolutionValidator
	publisher         ProblemPublisher
	txManager         appshared.TransactionManager
	problemRepo       problem.Repository
	stagedFiles       StagedFileCleaner
}

func NewValidateProblemUseCase(
	validationRepo problem.ProblemValidationRepository,
	judgingPreparer JudgingPreparer,
	artifactWriter JudgingArtifactWriter,
	solutionValidator SolutionValidator,
	publisher ProblemPublisher,
	txManager appshared.TransactionManager,
	problemRepo problem.Repository,
	stagedFiles StagedFileCleaner,
) *ValidateProblemUseCase {
	return &ValidateProblemUseCase{
		validationRepo:    validationRepo,
		judgingPreparer:   judgingPreparer,
		artifactWriter:    artifactWriter,
		solutionValidator: solutionValidator,
		publisher:         publisher,
		txManager:         txManager,
		problemRepo:       problemRepo,
		stagedFiles:       stagedFiles,
	}
}

// Execute runs one validation ticket to completion. It never returns a
// result to a caller — the worker logs whatever error comes back and moves
// on; a ticket left stuck in RUNNING because of an infra failure is the
// stale-recovery sweep's job, not this use case's.
func (uc *ValidateProblemUseCase) Execute(ctx context.Context, in ValidateProblemInput) error {
	now := time.Now()

	v, err := uc.validationRepo.FindByID(ctx, in.ValidationID)
	if err != nil {
		return err
	}
	if !v.Status().IsPending() {
		return nil
	}

	if err := v.Start(now); err != nil {
		return err
	}
	if err := uc.validationRepo.Save(ctx, v); err != nil {
		return err
	}

	if v.IsJudgingUpdate() {
		return uc.executeJudgingUpdate(ctx, v, in.Slug, now)
	}

	prep, err := uc.judgingPreparer.Prepare(ctx, v.ProblemID(), in.Slug, nil)
	if err != nil {
		slog.ErrorContext(ctx, "validate_problem: judging preparation failed", "error", err, "validation_id", v.ID())
		return uc.persistSystemError(ctx, v, now)
	}
	if prep.Failure != nil {
		report := buildJudgingFailureReport(prep.Failure)
		resultJSON, _ := json.Marshal(report)
		_ = v.MarkFailed(string(resultJSON), now)
		return uc.validationRepo.Save(ctx, v)
	}

	if prep.CheckerCompiledKey != "" {
		if err := uc.artifactWriter.SetCheckerCompiledKey(ctx, v.ProblemID(), prep.CheckerCompiledKey, now); err != nil {
			slog.ErrorContext(ctx, "validate_problem: failed to persist checker compiled key", "error", err, "validation_id", v.ID())
			return uc.persistSystemError(ctx, v, now)
		}
	}
	if prep.ValidatorCompiledKey != "" {
		if err := uc.artifactWriter.SetValidatorCompiledKey(ctx, v.ProblemID(), prep.ValidatorCompiledKey, now); err != nil {
			slog.ErrorContext(ctx, "validate_problem: failed to persist validator compiled key", "error", err, "validation_id", v.ID())
			return uc.persistSystemError(ctx, v, now)
		}
	}

	result, err := uc.solutionValidator.Validate(ctx, v.ProblemID(), nil)
	if err != nil {
		slog.ErrorContext(ctx, "validate_problem: solution validation failed", "error", err, "validation_id", v.ID())
		return uc.persistSystemError(ctx, v, now)
	}

	report := buildValidationReport(result)
	resultJSON, _ := json.Marshal(report)

	if !result.Passed {
		_ = v.MarkFailed(string(resultJSON), now)
		return uc.validationRepo.Save(ctx, v)
	}

	if !v.PublishesOnPass() {
		// "Validar": the draft passed, and publishing it is left to the author.
		_ = v.MarkPassed(string(resultJSON), now)
		return uc.validationRepo.Save(ctx, v)
	}

	// Publishing the problem and marking the ticket PASSED must happen
	// together: if MarkPublished succeeded but the ticket save failed, the
	// unique-active-validation index would leave the problem stuck unable
	// to ever start a new validation. If this transaction fails, the ticket
	// stays RUNNING in the database (nothing here mutated it on disk) for
	// the stale-recovery sweep to pick up later.
	return uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if err := uc.publisher.MarkPublished(txCtx, v.ProblemID(), now); err != nil {
			return err
		}
		_ = v.MarkPassed(string(resultJSON), now)
		return uc.validationRepo.Save(txCtx, v)
	})
}

func (uc *ValidateProblemUseCase) persistSystemError(ctx context.Context, v *problem.ProblemValidation, now time.Time) error {
	report := ValidationReport{ValidationLogs: []string{"✗ Internal error during validation"}}
	resultJSON, _ := json.Marshal(report)
	_ = v.MarkSystemError(string(resultJSON), now)
	return uc.validationRepo.Save(ctx, v)
}

func buildJudgingFailureReport(f *JudgingPreparationFailure) ValidationReport {
	switch f.Kind {
	case JudgingFailureCheckerCompileError:
		return ValidationReport{
			ValidationLogs:    []string{fmt.Sprintf("✗ Checker compilation failed: %s", f.FileKey), "  " + f.Log},
			CompilationErrors: &CompilationErrors{File: f.FileKey, Errors: []string{f.Log}},
		}
	case JudgingFailureValidatorCompileError:
		return ValidationReport{
			ValidationLogs:    []string{fmt.Sprintf("✗ Validator compilation failed: %s", f.FileKey), "  " + f.Log},
			CompilationErrors: &CompilationErrors{File: f.FileKey, Errors: []string{f.Log}},
		}
	default: // JudgingFailureInputRejected
		return ValidationReport{
			ValidationLogs: []string{fmt.Sprintf("✗ Validator rejected input: %s", f.TestCase), "  Error: " + f.Reason},
			FailedInputs:   []FailedInput{{File: f.TestCase, Reason: f.Reason}},
		}
	}
}

func buildValidationReport(result *SolutionValidationResult) ValidationReport {
	if result.Passed {
		return ValidationReport{
			ValidationLogs: []string{
				fmt.Sprintf("✓ Compiled and ran %d solution(s)", result.SolutionsTested),
				fmt.Sprintf("✓ Passed %d sample and %d secret test case(s)", result.SampleCases, result.SecretCases),
			},
			ValidationSummary: &ValidationSummary{
				SampleCases:     result.SampleCases,
				SecretCases:     result.SecretCases,
				SolutionsTested: result.SolutionsTested,
				AllPassed:       true,
			},
		}
	}

	f := result.Failure
	if f.Kind == SolutionFailureCompileError {
		return ValidationReport{
			ValidationLogs: []string{
				fmt.Sprintf("✗ Solution failed to compile: %s", f.FileKey),
				"  " + f.CompileLog,
			},
			CompilationErrors: &CompilationErrors{File: f.FileKey, Errors: []string{f.CompileLog}},
		}
	}

	tc := FailedTestCase{Case: f.TestCase, Verdict: string(f.Kind)}
	logs := []string{fmt.Sprintf("✗ Solution failed test case: %s", f.TestCase)}
	switch f.Kind {
	case SolutionFailureWrongAnswer:
		tc.Expected = string(f.Expected)
		tc.Actual = string(f.Actual)
		tc.Details = f.CheckerMessage
		logs = append(logs, fmt.Sprintf("  Expected: %s", f.Expected), fmt.Sprintf("  Got: %s", f.Actual))
		if f.CheckerMessage != "" {
			logs = append(logs, "  Checker: "+f.CheckerMessage)
		}
	case SolutionFailureTimeLimitExceeded:
		timeLimit := f.TimeLimitMs
		tc.TimeLimit = &timeLimit
		logs = append(logs, fmt.Sprintf("  Time limit: %dms", f.TimeLimitMs), "  Execution time: exceeded (killed)")
	case SolutionFailureMemoryLimitExceeded:
		logs = append(logs, "  Exceeded the memory limit")
	case SolutionFailureRuntimeError:
		logs = append(logs, "  Got: Runtime Error")
	}

	return ValidationReport{ValidationLogs: logs, FailedTestCases: []FailedTestCase{tc}}
}

// executeJudgingUpdate swaps the staged files in, together with the ticket's PASSED mark, only if every solution still passes; until then the problem is judged as before.
func (uc *ValidateProblemUseCase) executeJudgingUpdate(ctx context.Context, v *problem.ProblemValidation, slug string, now time.Time) error {
	change := v.PendingChange()
	candidate := judgingCandidateOf(*change, slug, v.ID())

	prep, err := uc.judgingPreparer.Prepare(ctx, v.ProblemID(), slug, candidate)
	if err != nil {
		slog.ErrorContext(ctx, "validate_problem: judging update preparation failed", "error", err, "validation_id", v.ID())
		return uc.failJudgingUpdate(ctx, v, slug, nil, now)
	}
	if prep.Failure != nil {
		report := buildJudgingFailureReport(prep.Failure)
		return uc.failJudgingUpdate(ctx, v, slug, &report, now)
	}

	result, err := uc.solutionValidator.Validate(ctx, v.ProblemID(), candidate)
	if err != nil {
		slog.ErrorContext(ctx, "validate_problem: judging update solution validation failed", "error", err, "validation_id", v.ID())
		return uc.failJudgingUpdate(ctx, v, slug, nil, now)
	}
	if !result.Passed {
		report := buildValidationReport(result)
		return uc.failJudgingUpdate(ctx, v, slug, &report, now)
	}

	report := buildValidationReport(result)
	report.ValidationLogs = append(report.ValidationLogs, "✓ Applied: "+describeChange(*change))

	err = uc.txManager.WithTx(ctx, func(txCtx context.Context) error {
		parsedSlug, err := problem.NewSlug(slug)
		if err != nil {
			return err
		}
		p, err := uc.problemRepo.FindBySlug(txCtx, parsedSlug)
		if err != nil {
			return err
		}
		logPreviousJudging(ctx, p, v.ID())

		if key := change.TestCasesKey(); key != nil {
			p.SetTestCases(*key, now)
		}
		if c := change.Checker(); c != nil {
			compiledKey := stagedCheckerCompiledKey(slug, v.ID())
			p.SetChecker(problem.RestoreJudgingFile(c.Filename(), c.FileKey(), c.Language(), &compiledKey, &now), now)
		}
		if err := uc.problemRepo.Save(txCtx, p); err != nil {
			return err
		}

		resultJSON, _ := json.Marshal(report)
		_ = v.MarkPassed(string(resultJSON), now)
		return uc.validationRepo.Save(txCtx, v)
	})
	if err != nil {
		slog.ErrorContext(ctx, "validate_problem: applying judging update failed", "error", err, "validation_id", v.ID())
		return uc.failJudgingUpdate(ctx, v, slug, nil, now)
	}
	return nil
}

// failJudgingUpdate closes the ticket without touching the problem: FAILED with a report, SYSTEM_ERROR without.
func (uc *ValidateProblemUseCase) failJudgingUpdate(ctx context.Context, v *problem.ProblemValidation, slug string, report *ValidationReport, now time.Time) error {
	uc.discardStagedFiles(ctx, v, slug)
	if report == nil {
		return uc.persistSystemError(ctx, v, now)
	}
	resultJSON, _ := json.Marshal(report)
	_ = v.MarkFailed(string(resultJSON), now)
	return uc.validationRepo.Save(ctx, v)
}

func (uc *ValidateProblemUseCase) discardStagedFiles(ctx context.Context, v *problem.ProblemValidation, slug string) {
	change := v.PendingChange()
	if key := change.TestCasesKey(); key != nil {
		if err := uc.stagedFiles.DeleteFilesWithPrefix(ctx, *key); err != nil {
			slog.ErrorContext(ctx, "validate_problem: could not drop staged test cases", "prefix", *key, "error", err, "validation_id", v.ID())
		}
	}
	if change.Checker() != nil {
		prefix := stagedCheckerPrefix(slug, v.ID())
		if err := uc.stagedFiles.DeleteFilesWithPrefix(ctx, prefix); err != nil {
			slog.ErrorContext(ctx, "validate_problem: could not drop staged checker", "prefix", prefix, "error", err, "validation_id", v.ID())
		}
	}
}

func judgingCandidateOf(change problem.PendingJudgingChange, slug, validationID string) *JudgingCandidate {
	c := &JudgingCandidate{}
	if key := change.TestCasesKey(); key != nil {
		c.TestCasesKey = *key
	}
	if checker := change.Checker(); checker != nil {
		c.CheckerFileKey = checker.FileKey()
		c.CheckerLanguage = checker.Language()
		c.CheckerCompiledKey = stagedCheckerCompiledKey(slug, validationID)
	}
	return c
}

func describeChange(change problem.PendingJudgingChange) string {
	switch {
	case change.TestCasesKey() != nil && change.Checker() != nil:
		return "test cases and checker"
	case change.TestCasesKey() != nil:
		return "test cases"
	default:
		return "checker"
	}
}

// The replaced files stay in storage: a judging already in flight may still be reading them.
func logPreviousJudging(ctx context.Context, p *problem.Problem, validationID string) {
	previousTestCases, previousChecker := "", ""
	if p.TestCasesKey() != nil {
		previousTestCases = *p.TestCasesKey()
	}
	if p.Checker() != nil {
		previousChecker = fmt.Sprintf("%s (compiled: %v)", p.Checker().FileKey(), p.Checker().CompiledKey())
	}
	slog.InfoContext(ctx, "validate_problem: judging update replaces",
		"validation_id", validationID, "problem_id", p.ID(),
		"previous_test_cases", previousTestCases, "previous_checker", previousChecker)
}
