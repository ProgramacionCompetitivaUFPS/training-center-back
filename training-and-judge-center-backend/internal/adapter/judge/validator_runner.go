package judge

import (
	"bytes"
	"context"
	"log/slog"
	"os"
	"os/exec"
	"time"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ValidatorRunner struct {
	timeout time.Duration
}

func NewValidatorRunner() *ValidatorRunner {
	return &ValidatorRunner{timeout: trustedSubprocessTimeout}
}

// Run writes the compiled validator to disk via artifactInvocation and runs
// it with req.Input on stdin — testlib convention: exit code 0 means
// accepted. Running is identical across languages once artifactInvocation
// resolves how to invoke each one; only that resolution is language-specific.
func (r *ValidatorRunner) Run(ctx context.Context, req appJudge.ValidatorRunRequest) (appJudge.ValidatorRunResult, error) {
	tmpDir, err := os.MkdirTemp("", "validator-run-*")
	if err != nil {
		slog.ErrorContext(ctx, "validator_runner: failed to create temp dir", "error", err)
		return appJudge.ValidatorRunResult{}, apperror.NewInternal()
	}
	defer os.RemoveAll(tmpDir)

	program, argsPrefix, err := artifactInvocation(tmpDir, req.Filename, req.Language, req.Artifact)
	if err != nil {
		slog.ErrorContext(ctx, "validator_runner: failed to prepare artifact", "error", err, "language", req.Language.String())
		return appJudge.ValidatorRunResult{}, apperror.NewInternal()
	}

	runCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(runCtx, program, argsPrefix...)
	cmd.Stdin = bytes.NewReader(req.Input)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isTimeoutErr(runCtx, err) {
			slog.ErrorContext(ctx, "validator_runner: validator timed out", "error", runCtx.Err())
			return appJudge.ValidatorRunResult{}, apperror.NewInternal()
		}
		if _, ok := err.(*exec.ExitError); ok {
			return appJudge.ValidatorRunResult{Accepted: false, Message: stderr.String()}, nil
		}
		slog.ErrorContext(ctx, "validator_runner: failed to run validator", "error", err)
		return appJudge.ValidatorRunResult{}, apperror.NewInternal()
	}
	return appJudge.ValidatorRunResult{Accepted: true}, nil
}
