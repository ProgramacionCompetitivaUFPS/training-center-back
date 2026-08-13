package judge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"

	"cloud.google.com/go/storage"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type OutputComparator struct {
	reader gcsReader
}

func NewOutputComparator(client *storage.Client, bucket string) *OutputComparator {
	return &OutputComparator{reader: newGCSReader(client, bucket)}
}

func (c *OutputComparator) Check(ctx context.Context, req appJudge.CheckRequest) (appJudge.CheckResult, error) {
	if req.CheckerPath == "" {
		return tokenCompare(req.ExpectedOutput, req.ContestantOutput), nil
	}
	return c.customCheckerCompare(ctx, req)
}

// tokenCompare splits both outputs into whitespace-delimited tokens and
// compares them element by element. strings.Fields handles \r\n, multiple
// spaces, and trailing whitespace, so semantically equivalent outputs
// produced on different platforms are treated as equal.
func tokenCompare(expected, contestant []byte) appJudge.CheckResult {
	expTokens := strings.Fields(string(expected))
	conTokens := strings.Fields(string(contestant))
	if len(expTokens) != len(conTokens) {
		return appJudge.CheckResult{Accepted: false}
	}
	for i := range expTokens {
		if expTokens[i] != conTokens[i] {
			return appJudge.CheckResult{Accepted: false}
		}
	}
	return appJudge.CheckResult{Accepted: true}
}

func (c *OutputComparator) customCheckerCompare(ctx context.Context, req appJudge.CheckRequest) (appJudge.CheckResult, error) {
	tmpDir, err := os.MkdirTemp("", "judge-checker-*")
	if err != nil {
		slog.ErrorContext(ctx, "output_comparator: failed to create temp dir", "error", err)
		return appJudge.CheckResult{}, apperror.NewInternal()
	}
	defer os.RemoveAll(tmpDir)

	// Download the checker binary from GCS.
	rc, err := c.reader.readObject(ctx, req.CheckerPath)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.ErrorContext(ctx, "output_comparator: checker not found in GCS", "path", req.CheckerPath)
			return appJudge.CheckResult{}, apperror.NewInternal()
		}
		slog.ErrorContext(ctx, "output_comparator: failed to open checker", "path", req.CheckerPath, "error", err)
		return appJudge.CheckResult{}, apperror.NewInternal()
	}
	checkerBin, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		slog.ErrorContext(ctx, "output_comparator: failed to read checker binary", "error", err)
		return appJudge.CheckResult{}, apperror.NewInternal()
	}

	// Write the input/expected/contestant files to the temp directory — the
	// checker artifact itself is written by artifactInvocation, which knows
	// what shape each language needs it in.
	inputPath := tmpDir + "/input"
	expectedPath := tmpDir + "/expected"
	contestantPath := tmpDir + "/contestant"

	for _, entry := range []struct {
		path    string
		content []byte
	}{
		{inputPath, req.Input},
		{expectedPath, req.ExpectedOutput},
		{contestantPath, req.ContestantOutput},
	} {
		if err := os.WriteFile(entry.path, entry.content, 0o644); err != nil {
			slog.ErrorContext(ctx, "output_comparator: failed to write temp file", "path", entry.path, "error", err)
			return appJudge.CheckResult{}, apperror.NewInternal()
		}
	}

	program, argsPrefix, err := artifactInvocation(tmpDir, req.CheckerFilename, req.CheckerLanguage, checkerBin)
	if err != nil {
		slog.ErrorContext(ctx, "output_comparator: failed to prepare checker artifact", "error", err, "path", req.CheckerPath)
		return appJudge.CheckResult{}, apperror.NewInternal()
	}

	// Run the checker. The subprocess is purely local — the only network call
	// was the GCS download above.
	runCtx, cancel := context.WithTimeout(ctx, trustedSubprocessTimeout)
	defer cancel()

	var stderr bytes.Buffer
	cmd := exec.CommandContext(runCtx, program, append(argsPrefix, inputPath, expectedPath, contestantPath)...)
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if isTimeoutErr(runCtx, err) {
			slog.ErrorContext(ctx, "output_comparator: checker timed out", "path", req.CheckerPath, "error", runCtx.Err())
			return appJudge.CheckResult{}, apperror.NewInternal()
		}
		if _, ok := err.(*exec.ExitError); ok {
			return appJudge.CheckResult{
				Accepted: false,
				Message:  strings.TrimSpace(stderr.String()),
			}, nil
		}
		slog.ErrorContext(ctx, "output_comparator: checker execution failed", "path", req.CheckerPath, "error", err)
		return appJudge.CheckResult{}, apperror.NewInternal()
	}

	return appJudge.CheckResult{Accepted: true}, nil
}
