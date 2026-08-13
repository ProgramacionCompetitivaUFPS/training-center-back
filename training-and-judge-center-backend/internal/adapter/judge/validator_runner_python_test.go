package judge

import (
	"context"
	"strings"
	"testing"
	"time"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

const pythonValidatorSource = `
import sys
x = int(sys.stdin.read().strip())
if x > 0:
    sys.exit(0)
sys.stderr.write("value must be positive")
sys.exit(1)
`

func compiledPythonValidator(t *testing.T) []byte {
	t.Helper()
	result, err := NewNativeCompiler().Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "validator.py",
		Language:   submission.RestoreLanguage("python310"),
		SourceCode: []byte(pythonValidatorSource),
	})
	if err != nil || !result.Success {
		t.Fatalf("failed to compile test validator: err=%v log=%s", err, result.Log)
	}
	return result.Artifact
}

func TestRunValidatorPython_AcceptsValidInput(t *testing.T) {
	requireTool(t, "python3")
	artifact := compiledPythonValidator(t)

	result, err := NewValidatorRunner().Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "validator.py",
		Language: submission.RestoreLanguage("python310"),
		Artifact: artifact,
		Input:    []byte("5"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Accepted {
		t.Errorf("expected Accepted=true, got Message: %s", result.Message)
	}
}

func TestRunValidatorPython_RejectsInvalidInput(t *testing.T) {
	requireTool(t, "python3")
	artifact := compiledPythonValidator(t)

	result, err := NewValidatorRunner().Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "validator.py",
		Language: submission.RestoreLanguage("python310"),
		Artifact: artifact,
		Input:    []byte("-5"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Accepted {
		t.Fatal("expected Accepted=false")
	}
	if !strings.Contains(result.Message, "positive") {
		t.Errorf("expected the rejection message to explain why, got: %s", result.Message)
	}
}

// TestRunValidatorPython_HangingValidator_TimesOut proves
// trustedSubprocessTimeout actually stops a stuck validator instead of
// letting it run forever — a validator with a bug (e.g. waiting on input
// that never comes) is exactly the scenario this timeout exists for.
func TestRunValidatorPython_HangingValidator_TimesOut(t *testing.T) {
	requireTool(t, "python3")

	hangingSource := []byte("import time\ntime.sleep(60)\n")
	result, err := NewNativeCompiler().Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "validator.py",
		Language:   submission.RestoreLanguage("python310"),
		SourceCode: hangingSource,
	})
	if err != nil || !result.Success {
		t.Fatalf("failed to compile test validator: err=%v log=%s", err, result.Log)
	}

	runner := &ValidatorRunner{timeout: 200 * time.Millisecond}
	_, err = runner.Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "validator.py",
		Language: submission.RestoreLanguage("python310"),
		Artifact: result.Artifact,
		Input:    []byte("5"),
	})
	if err == nil {
		t.Fatal("expected an error when the validator hangs past the timeout")
	}
}
