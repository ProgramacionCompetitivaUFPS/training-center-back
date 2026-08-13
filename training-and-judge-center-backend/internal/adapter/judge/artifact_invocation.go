package judge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

// artifactInvocation writes a compiled checker/validator artifact to dir in
// whatever shape its language needs to run, and returns the program plus
// leading arguments needed to invoke it — only cpp20 produces something the
// OS can run directly, java17 needs the JVM, and python310 needs the
// interpreter (same reasoning as NativeCompiler's per-language compilation).
// The caller appends whatever comes after: stdin wiring for a validator,
// trailing file arguments for a checker.
func artifactInvocation(dir, filename string, language submission.Language, artifact []byte) (program string, argsPrefix []string, err error) {
	switch language.String() {
	case "cpp20":
		binPath := filepath.Join(dir, "artifact")
		if err := os.WriteFile(binPath, artifact, 0o755); err != nil {
			return "", nil, err
		}
		return binPath, nil, nil
	case "java17":
		className := strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename))
		if err := os.WriteFile(filepath.Join(dir, className+".class"), artifact, 0o644); err != nil {
			return "", nil, err
		}
		return "java", []string{"-cp", dir, className}, nil
	case "python310":
		sourcePath := filepath.Join(dir, filepath.Base(filename))
		if err := os.WriteFile(sourcePath, artifact, 0o644); err != nil {
			return "", nil, err
		}
		return "python3", []string{sourcePath}, nil
	default:
		return "", nil, fmt.Errorf("unsupported language: %q", language.String())
	}
}
