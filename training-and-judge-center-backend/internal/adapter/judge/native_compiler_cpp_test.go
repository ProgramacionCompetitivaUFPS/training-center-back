package judge

import (
	"context"
	"strings"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestCompileCpp_Success_ReturnsArtifact(t *testing.T) {
	requireTool(t, "g++")
	c := NewNativeCompiler()

	result, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "checker.cpp",
		Language:   submission.RestoreLanguage("cpp20"),
		SourceCode: []byte(`int main() { return 0; }`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got Log: %s", result.Log)
	}
	if len(result.Artifact) == 0 {
		t.Error("expected a non-empty compiled artifact")
	}
}

func TestCompileCpp_SyntaxError_ReturnsFailureWithLog(t *testing.T) {
	requireTool(t, "g++")
	c := NewNativeCompiler()

	result, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "checker.cpp",
		Language:   submission.RestoreLanguage("cpp20"),
		SourceCode: []byte(`int main( { this is not valid c++`),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected Success=false for invalid source")
	}
	if !strings.Contains(result.Log, "error") {
		t.Errorf("expected the compiler log to mention an error, got: %s", result.Log)
	}
	if result.Artifact != nil {
		t.Error("expected no artifact on failure")
	}
}
