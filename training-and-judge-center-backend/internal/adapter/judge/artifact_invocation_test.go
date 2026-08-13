package judge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestArtifactInvocation_Cpp_WritesExecutableAndReturnsItDirectly(t *testing.T) {
	dir := t.TempDir()

	program, argsPrefix, err := artifactInvocation(dir, "checker.cpp", submission.RestoreLanguage("cpp20"), []byte("fake binary"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(argsPrefix) != 0 {
		t.Errorf("argsPrefix: got %v, want none — cpp runs directly", argsPrefix)
	}
	info, statErr := os.Stat(program)
	if statErr != nil {
		t.Fatalf("expected the artifact to exist at %q: %v", program, statErr)
	}
	if info.Mode()&0o111 == 0 {
		t.Error("expected the artifact to be executable")
	}
}

func TestArtifactInvocation_Java_DerivesClassNameFromFilename(t *testing.T) {
	dir := t.TempDir()

	program, argsPrefix, err := artifactInvocation(dir, "Checker.java", submission.RestoreLanguage("java17"), []byte("fake bytecode"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program != "java" {
		t.Errorf("program: got %q, want java", program)
	}
	wantArgs := []string{"-cp", dir, "Checker"}
	if len(argsPrefix) != len(wantArgs) {
		t.Fatalf("argsPrefix: got %v, want %v", argsPrefix, wantArgs)
	}
	for i := range wantArgs {
		if argsPrefix[i] != wantArgs[i] {
			t.Errorf("argsPrefix[%d]: got %q, want %q", i, argsPrefix[i], wantArgs[i])
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "Checker.class")); err != nil {
		t.Errorf("expected Checker.class to exist: %v", err)
	}
}

func TestArtifactInvocation_Python_WritesSourceAndReturnsInterpreterCommand(t *testing.T) {
	dir := t.TempDir()

	program, argsPrefix, err := artifactInvocation(dir, "checker.py", submission.RestoreLanguage("python310"), []byte("print('ok')"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if program != "python3" {
		t.Errorf("program: got %q, want python3", program)
	}
	wantPath := filepath.Join(dir, "checker.py")
	if len(argsPrefix) != 1 || argsPrefix[0] != wantPath {
		t.Errorf("argsPrefix: got %v, want [%q]", argsPrefix, wantPath)
	}
	if _, err := os.Stat(wantPath); err != nil {
		t.Errorf("expected checker.py to exist: %v", err)
	}
}

func TestArtifactInvocation_UnsupportedLanguage_ReturnsError(t *testing.T) {
	dir := t.TempDir()

	_, _, err := artifactInvocation(dir, "checker.rs", submission.RestoreLanguage("rust"), []byte("whatever"))
	if err == nil {
		t.Error("expected an error for an unsupported language")
	}
}
