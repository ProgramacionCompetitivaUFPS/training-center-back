package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/pkg/judgelimits"
)

// writeFiles lays out the three files a checker receives, in the order it
// receives them, and returns the argument list pointing at them. The input is
// never read, but it is written anyway so the invocation matches a real one.
func writeFiles(t *testing.T, contestant, answer string) []string {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, 0, 3)
	for _, f := range []struct{ name, content string }{
		{"input", ""},
		{"output", contestant},
		{"answer", answer},
	} {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
			t.Fatalf("could not write %s: %v", f.name, err)
		}
		paths = append(paths, path)
	}
	return paths
}

// The cases mirror the token comparison this binary replaced in the worker, so
// a divergence in verdict from the behaviour that shipped shows up here.
func TestRun_Verdicts(t *testing.T) {
	tests := []struct {
		name       string
		contestant string
		answer     string
		want       int
	}{
		{"exact match", "3 5\n10\n", "3 5\n10\n", exitAccepted},
		{"CRLF vs LF", "3 5\r\n10\r\n", "3 5\n10\n", exitAccepted},
		{"trailing space", "3 5\n10\n   ", "3 5\n10\n", exitAccepted},
		{"double space between tokens", "3  5\n10\n", "3 5\n10\n", exitAccepted},
		{"leading newline", "\n3\n", "3\n", exitAccepted},
		{"count mismatch — extra token", "3 5 7", "3 5", exitRejected},
		{"count mismatch — fewer tokens", "3 5", "3 5 7", exitRejected},
		{"value mismatch", "3 5\n11\n", "3 5\n10\n", exitRejected},
		{"both empty — accepted", "", "", exitAccepted},
		{"contestant printed something, answer empty", "42", "", exitRejected},
		{"contestant printed nothing, answer not empty", "", "42", exitRejected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got := run(writeFiles(t, tt.contestant, tt.answer), &stderr)
			if got != tt.want {
				t.Errorf("exit code: got %d, want %d (stderr: %q)", got, tt.want, stderr.String())
			}
		})
	}
}

// The rejection message reaches CheckResult.Message, which may be surfaced to
// the contestant — so it must never carry a token of the jury's answer.
func TestRun_RejectionMessageDoesNotLeakTheJurysTokens(t *testing.T) {
	var stderr bytes.Buffer
	if got := run(writeFiles(t, "secret 99999", "secret 12345"), &stderr); got != exitRejected {
		t.Fatalf("exit code: got %d, want %d", got, exitRejected)
	}
	if msg := stderr.String(); strings.Contains(msg, "12345") {
		t.Errorf("message leaked a token of the jury's answer: %q", msg)
	}
}

func TestRun_WrongArgumentCount_ReportsFailure(t *testing.T) {
	var stderr bytes.Buffer
	if got := run([]string{"only", "two"}, &stderr); got != exitFailure {
		t.Errorf("exit code: got %d, want %d", got, exitFailure)
	}
	if stderr.Len() == 0 {
		t.Error("expected a usage message on stderr")
	}
}

// A missing file is the checker failing, not the contestant being wrong, so it
// must not be reported as a rejection. Which file the message names is also
// what pins the argument order: the verdict itself is symmetric.
func TestRun_MissingFile_ReportsFailure(t *testing.T) {
	tests := []struct {
		name       string
		position   int
		wantPrefix string
	}{
		{"the contestant's output", 1, "reading the contestant's output"},
		{"the jury's answer", 2, "reading the jury's answer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := writeFiles(t, "1", "1")
			args[tt.position] = filepath.Join(t.TempDir(), "does-not-exist")

			var stderr bytes.Buffer
			if got := run(args, &stderr); got != exitFailure {
				t.Errorf("exit code: got %d, want %d", got, exitFailure)
			}
			// HasPrefix and not Contains: t.TempDir() puts the subtest name in the
			// path, so the words being looked for appear in the error either way.
			if msg := stderr.String(); !strings.HasPrefix(msg, tt.wantPrefix) {
				t.Errorf("message: got %q, want it to start with %q", msg, tt.wantPrefix)
			}
		})
	}
}

// A token beyond the buffer limit is a checker failure too — reporting it as a
// wrong answer would blame the contestant for our own limit.
func TestRun_TokenTooLong_ReportsFailure(t *testing.T) {
	huge := strings.Repeat("x", judgelimits.MaxTokenBytes+1)

	var stderr bytes.Buffer
	if got := run(writeFiles(t, huge, huge), &stderr); got != exitFailure {
		t.Errorf("exit code: got %d, want %d", got, exitFailure)
	}
}
