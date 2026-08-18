package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFiles lays out the three files a checker receives and returns the
// argument list pointing at them. The input is never read, but it is written
// anyway so the invocation matches a real one.
func writeFiles(t *testing.T, expected, contestant string) []string {
	t.Helper()
	dir := t.TempDir()
	paths := make([]string, 0, 3)
	for _, f := range []struct{ name, content string }{
		{"input", ""},
		{"expected", expected},
		{"contestant", contestant},
	} {
		path := filepath.Join(dir, f.name)
		if err := os.WriteFile(path, []byte(f.content), 0o644); err != nil {
			t.Fatalf("could not write %s: %v", f.name, err)
		}
		paths = append(paths, path)
	}
	return paths
}

// The cases mirror the ones that covered tokenCompare in the worker, so a
// divergence in verdict between the two implementations shows up here.
func TestRun_Verdicts(t *testing.T) {
	tests := []struct {
		name       string
		expected   string
		contestant string
		want       int
	}{
		{"exact match", "3 5\n10\n", "3 5\n10\n", exitAccepted},
		{"CRLF vs LF", "3 5\n10\n", "3 5\r\n10\r\n", exitAccepted},
		{"trailing space", "3 5\n10\n", "3 5\n10\n   ", exitAccepted},
		{"double space between tokens", "3 5\n10\n", "3  5\n10\n", exitAccepted},
		{"leading newline", "3\n", "\n3\n", exitAccepted},
		{"count mismatch — extra token", "3 5", "3 5 7", exitRejected},
		{"count mismatch — fewer tokens", "3 5 7", "3 5", exitRejected},
		{"value mismatch", "3 5\n10\n", "3 5\n11\n", exitRejected},
		{"both empty — accepted", "", "", exitAccepted},
		{"expected empty contestant not", "", "42", exitRejected},
		{"expected not empty contestant empty", "42", "", exitRejected},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer
			got := run(writeFiles(t, tt.expected, tt.contestant), &stderr)
			if got != tt.want {
				t.Errorf("exit code: got %d, want %d (stderr: %q)", got, tt.want, stderr.String())
			}
		})
	}
}

// The rejection message reaches CheckResult.Message, which may be surfaced to
// the contestant — so it must never carry the expected output.
func TestRun_RejectionMessageDoesNotLeakExpectedTokens(t *testing.T) {
	var stderr bytes.Buffer
	if got := run(writeFiles(t, "secret 12345", "secret 99999"), &stderr); got != exitRejected {
		t.Fatalf("exit code: got %d, want %d", got, exitRejected)
	}
	if msg := stderr.String(); strings.Contains(msg, "12345") {
		t.Errorf("message leaked the expected token: %q", msg)
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
// must not be reported as a rejection.
func TestRun_MissingFile_ReportsFailure(t *testing.T) {
	args := writeFiles(t, "1", "1")
	args[1] = filepath.Join(t.TempDir(), "does-not-exist")

	var stderr bytes.Buffer
	if got := run(args, &stderr); got != exitFailure {
		t.Errorf("exit code: got %d, want %d", got, exitFailure)
	}
}

// A token beyond the buffer limit is a checker failure too — reporting it as a
// wrong answer would blame the contestant for our own limit.
func TestRun_TokenTooLong_ReportsFailure(t *testing.T) {
	huge := strings.Repeat("x", maxTokenBytes+1)

	var stderr bytes.Buffer
	if got := run(writeFiles(t, huge, huge), &stderr); got != exitFailure {
		t.Errorf("exit code: got %d, want %d", got, exitFailure)
	}
}
