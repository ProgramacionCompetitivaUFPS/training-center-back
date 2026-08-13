package judge

import (
	"context"
	"os/exec"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// requireTool skips the test when the named executable isn't on PATH —
// these tests shell out to real compilers, which this environment may not
// have installed.
func requireTool(t *testing.T, name string) {
	t.Helper()
	if _, err := exec.LookPath(name); err != nil {
		t.Skipf("%s not available on PATH, skipping", name)
	}
}

// TestNativeCompiler_UnsupportedLanguage_ReturnsInternal doesn't need any
// real compiler — the dispatcher rejects the language before it would ever
// shell out to one. submission.RestoreLanguage bypasses validation on
// purpose, simulating a corrupted value that should never reach here in
// practice (upload already restricts languages to the known set).
func TestNativeCompiler_UnsupportedLanguage_ReturnsInternal(t *testing.T) {
	c := NewNativeCompiler()

	_, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "checker.rs",
		Language:   submission.RestoreLanguage("rust"),
		SourceCode: []byte("fn main() {}"),
	})
	assertAppErrorKind(t, err, apperror.KindInternal)
}
