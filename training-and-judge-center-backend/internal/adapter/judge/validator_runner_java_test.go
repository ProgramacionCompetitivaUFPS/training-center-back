package judge

import (
	"context"
	"strings"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
)

const javaValidatorSource = `
import java.util.Scanner;
public class Validator {
    public static void main(String[] args) {
        Scanner sc = new Scanner(System.in);
        int x = sc.nextInt();
        if (x > 0) { System.exit(0); }
        System.err.println("value must be positive");
        System.exit(1);
    }
}
`

func compiledJavaValidator(t *testing.T) []byte {
	t.Helper()
	result, err := NewNativeCompiler().Compile(context.Background(), appJudge.CompileArtifactRequest{
		Filename:   "Validator.java",
		Language:   submission.RestoreLanguage("java17"),
		SourceCode: []byte(javaValidatorSource),
	})
	if err != nil || !result.Success {
		t.Fatalf("failed to compile test validator: err=%v log=%s", err, result.Log)
	}
	return result.Artifact
}

func TestRunValidatorJava_AcceptsValidInput(t *testing.T) {
	requireTool(t, "javac")
	requireTool(t, "java")
	artifact := compiledJavaValidator(t)

	result, err := NewValidatorRunner().Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "Validator.java",
		Language: submission.RestoreLanguage("java17"),
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

func TestRunValidatorJava_RejectsInvalidInput(t *testing.T) {
	requireTool(t, "javac")
	requireTool(t, "java")
	artifact := compiledJavaValidator(t)

	result, err := NewValidatorRunner().Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "Validator.java",
		Language: submission.RestoreLanguage("java17"),
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
