package judge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"testing"

	"cloud.google.com/go/storage"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestTokenCompare(t *testing.T) {
	tests := []struct {
		name       string
		expected   string
		contestant string
		accepted   bool
	}{
		{"exact match", "3 5\n10\n", "3 5\n10\n", true},
		{"CRLF vs LF", "3 5\n10\n", "3 5\r\n10\r\n", true},
		{"trailing space", "3 5\n10\n", "3 5\n10\n   ", true},
		{"double space between tokens", "3 5\n10\n", "3  5\n10\n", true},
		{"leading newline", "3\n", "\n3\n", true},
		{"count mismatch — extra token", "3 5", "3 5 7", false},
		{"count mismatch — fewer tokens", "3 5 7", "3 5", false},
		{"value mismatch", "3 5\n10\n", "3 5\n11\n", false},
		{"both empty — AC", "", "", true},
		{"expected empty contestant not", "", "42", false},
		{"expected not empty contestant empty", "42", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenCompare([]byte(tt.expected), []byte(tt.contestant))
			if result.Accepted != tt.accepted {
				t.Errorf("Accepted: got %v, want %v", result.Accepted, tt.accepted)
			}
		})
	}
}

func TestCheck_NoCheckerPath_UsesTokenCompare(t *testing.T) {
	comp := &OutputComparator{reader: &mockGCSReader{}}

	result, err := comp.Check(context.Background(), appJudge.CheckRequest{
		ExpectedOutput:   []byte("42\n"),
		ContestantOutput: []byte("42"),
		CheckerPath:      "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Accepted {
		t.Error("expected Accepted=true for matching tokens")
	}
}

func writeScript(t *testing.T, body string) string {
	t.Helper()
	f, err := os.CreateTemp("", "checker-test-*.sh")
	if err != nil {
		t.Fatalf("create temp script: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString("#!/bin/sh\n" + body + "\n"); err != nil {
		t.Fatalf("write script: %v", err)
	}
	if err := os.Chmod(f.Name(), 0o755); err != nil {
		t.Fatalf("chmod script: %v", err)
	}
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func requireSh(t *testing.T) {
	t.Helper()
	// Custom checker binaries are Linux executables; the judge worker runs on
	// Linux. Skip subprocess tests on Windows where shebangs are not supported.
	if runtime.GOOS == "windows" {
		t.Skip("subprocess checker tests require a Unix-like OS")
	}
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("requires sh (not available on this platform)")
	}
}

func TestCheck_CheckerNotInGCS_ReturnsInternal(t *testing.T) {
	comp := &OutputComparator{
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
				return nil, storage.ErrObjectNotExist
			},
		},
	}

	_, err := comp.Check(context.Background(), appJudge.CheckRequest{
		CheckerPath: "problems/abc/checker",
	})
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestCheck_CheckerGCSError_ReturnsInternal(t *testing.T) {
	comp := &OutputComparator{
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
				return nil, errors.New("network error")
			},
		},
	}

	_, err := comp.Check(context.Background(), appJudge.CheckRequest{
		CheckerPath: "problems/abc/checker",
	})
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestCheck_CustomChecker_AC(t *testing.T) {
	requireSh(t)

	script := []byte("#!/bin/sh\nexit 0\n")

	comp := &OutputComparator{
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(script)), nil
			},
		},
	}

	result, err := comp.Check(context.Background(), appJudge.CheckRequest{
		Input:            []byte("1 2"),
		ExpectedOutput:   []byte("3"),
		ContestantOutput: []byte("3"),
		CheckerPath:      "problems/abc/checker",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Accepted {
		t.Error("expected Accepted=true for exit 0 checker")
	}
}

func TestCheck_CustomChecker_WA(t *testing.T) {
	requireSh(t)

	script := []byte("#!/bin/sh\necho 'wrong answer' >&2\nexit 1\n")

	comp := &OutputComparator{
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
				return io.NopCloser(bytes.NewReader(script)), nil
			},
		},
	}

	result, err := comp.Check(context.Background(), appJudge.CheckRequest{
		Input:            []byte("1 2"),
		ExpectedOutput:   []byte("3"),
		ContestantOutput: []byte("4"),
		CheckerPath:      "problems/abc/checker",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Accepted {
		t.Error("expected Accepted=false for exit 1 checker")
	}
	if result.Message != "wrong answer" {
		t.Errorf("Message: got %q, want %q", result.Message, "wrong answer")
	}
}
