package problem

import (
	"context"
	"log/slog"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

type replacedJudgingFiles struct {
	testCasesPrefix string
	files           []string
}

func replacedByJudgingUpdate(p *problem.Problem, change problem.PendingJudgingChange, newCompiledKey string) replacedJudgingFiles {
	var replaced replacedJudgingFiles

	if key := change.TestCasesKey(); key != nil && p.TestCasesKey() != nil && *p.TestCasesKey() != *key {
		replaced.testCasesPrefix = *p.TestCasesKey()
	}

	newChecker := change.Checker()
	oldChecker := p.Checker()
	if newChecker == nil || oldChecker == nil {
		return replaced
	}
	if oldChecker.FileKey() != newChecker.FileKey() {
		replaced.files = append(replaced.files, oldChecker.FileKey())
	}
	if old := oldChecker.CompiledKey(); old != nil && *old != newCompiledKey {
		replaced.files = append(replaced.files, *old)
	}
	return replaced
}

func (uc *ValidateProblemUseCase) dropReplacedFiles(ctx context.Context, validationID string, replaced replacedJudgingFiles) {
	if replaced.testCasesPrefix != "" {
		prefix := replaced.testCasesPrefix
		if err := withStorageRetries(func() error { return uc.stagedFiles.DeleteFilesWithPrefix(ctx, prefix) }); err != nil {
			slog.ErrorContext(ctx, "validate_problem: judging update applied but the replaced test cases were left in storage", "prefix", prefix, "error", err, "validation_id", validationID)
		}
	}
	for _, key := range replaced.files {
		key := key
		if err := withStorageRetries(func() error { return uc.stagedFiles.DeleteFile(ctx, key) }); err != nil {
			slog.ErrorContext(ctx, "validate_problem: judging update applied but a replaced file was left in storage", "key", key, "error", err, "validation_id", validationID)
		}
	}
}
