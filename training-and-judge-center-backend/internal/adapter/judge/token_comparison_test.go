package judge

import (
	"context"
	"testing"
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

// The contestant output now comes off the volume, so this path reads the file
// the heavy pool left behind instead of receiving its bytes.
func TestTokenCheckerSession_Check_ComparesTheTwoOutputs(t *testing.T) {
	dir := layOutJudgingDir(t, t.TempDir(), "42")
	s := &tokenCheckerSession{judgingDir: dir}

	result, err := s.Check(context.Background(), []byte("42\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Accepted {
		t.Error("expected matching tokens to be accepted")
	}
	if err := s.Close(context.Background()); err != nil {
		t.Errorf("Close: %v", err)
	}
}
