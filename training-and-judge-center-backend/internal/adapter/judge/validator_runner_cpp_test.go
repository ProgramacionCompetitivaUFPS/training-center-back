package judge

import (
	"context"
	"strings"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

const cppValidatorSource = `
#include <iostream>
int main() {
    int x;
    std::cin >> x;
    if (x > 0) return 0;
    std::cerr << "value must be positive";
    return 1;
}
`

func compiledCppValidator(t *testing.T) []byte {
	t.Helper()
	result, err := NewNativeCompiler().Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "validator.cpp",
		Language:   submission.RestoreLanguage("cpp20"),
		SourceCode: []byte(cppValidatorSource),
	})
	if err != nil || !result.Success {
		t.Fatalf("failed to compile test validator: err=%v log=%s", err, result.Log)
	}
	return result.Artifact
}

func TestRunValidatorCpp_AcceptsValidInput(t *testing.T) {
	requireTool(t, "g++")
	artifact := compiledCppValidator(t)

	result, err := NewValidatorRunner().Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "validator.cpp",
		Language: submission.RestoreLanguage("cpp20"),
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

func TestRunValidatorCpp_RejectsInvalidInput(t *testing.T) {
	requireTool(t, "g++")
	artifact := compiledCppValidator(t)

	result, err := NewValidatorRunner().Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "validator.cpp",
		Language: submission.RestoreLanguage("cpp20"),
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
