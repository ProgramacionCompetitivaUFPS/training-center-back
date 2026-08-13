package judge

import (
	"context"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestCompileJava_Success_ReturnsArtifact(t *testing.T) {
	requireTool(t, "javac")
	c := NewNativeCompiler()

	result, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "Checker.java",
		Language:   submission.RestoreLanguage("java17"),
		SourceCode: []byte("public class Checker { public static void main(String[] args) {} }"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Success {
		t.Fatalf("expected Success=true, got Log: %s", result.Log)
	}
	if len(result.Artifact) == 0 {
		t.Error("expected a non-empty .class artifact")
	}
}

func TestCompileJava_SyntaxError_ReturnsFailureWithLog(t *testing.T) {
	requireTool(t, "javac")
	c := NewNativeCompiler()

	result, err := c.Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "Checker.java",
		Language:   submission.RestoreLanguage("java17"),
		SourceCode: []byte("public class Checker { this is not valid java"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Success {
		t.Fatal("expected Success=false for invalid source")
	}
	if result.Log == "" {
		t.Error("expected a non-empty compiler log")
	}
	if result.Artifact != nil {
		t.Error("expected no artifact on failure")
	}
}
