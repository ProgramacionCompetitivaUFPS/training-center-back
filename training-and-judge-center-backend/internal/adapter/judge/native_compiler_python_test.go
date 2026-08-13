package judge

import (
	"bytes"
	"context"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestCompilePython_Success_ReturnsSourceAsArtifact(t *testing.T) {
	requireTool(t, "python3")
	c := NewNativeCompiler()

	source := []byte("print('ok')\n")
	result, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "checker.py",
		Language:   submission.RestoreLanguage("python310"),
		SourceCode: source,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got Log: %s", result.Log)
	}
	if !bytes.Equal(result.Artifact, source) {
		t.Errorf("expected the artifact to be the original source, got %q", result.Artifact)
	}
}

func TestCompilePython_SyntaxError_ReturnsFailureWithLog(t *testing.T) {
	requireTool(t, "python3")
	c := NewNativeCompiler()

	result, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "checker.py",
		Language:   submission.RestoreLanguage("python310"),
		SourceCode: []byte("def broken(:\n    pass"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected Success=false for invalid syntax")
	}
	if result.Log == "" {
		t.Error("expected a non-empty compiler log")
	}
	if result.Artifact != nil {
		t.Error("expected no artifact on failure")
	}
}
