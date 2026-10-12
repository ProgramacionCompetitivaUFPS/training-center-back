package problem

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

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
