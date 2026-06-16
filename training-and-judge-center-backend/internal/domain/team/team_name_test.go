package team_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/team"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestNewTeamName_Valid(t *testing.T) {
	cases := []struct {
		input   string
		wantVal string
	}{
		{"Alpha", "Alpha"},
		{"  trimmed  ", "trimmed"},
		{"Team 2025", "Team 2025"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			n, err := team.NewTeamName(tc.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n.Value() != tc.wantVal {
				t.Errorf("Value() = %q, want %q", n.Value(), tc.wantVal)
			}
		})
	}
}

func TestNewTeamName_NKFCNormalization(t *testing.T) {
	// ﬁ (U+FB01 LATIN SMALL LIGATURE FI) normalizes to "fi" under NFKC
	n, err := team.NewTeamName("ﬁnals")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Value() != "finals" {
		t.Errorf("NFKC normalization failed: got %q, want %q", n.Value(), "finals")
	}
}

func TestNewTeamName_Empty(t *testing.T) {
	for _, input := range []string{"", "   "} {
		_, err := team.NewTeamName(input)
		if err == nil {
			t.Errorf("expected error for %q, got nil", input)
			continue
		}
		ae, ok := err.(*apperror.AppError)
		if !ok || ae.Code != apperror.ErrCodeValidationError {
			t.Errorf("expected VALIDATION_ERROR, got %v", err)
		}
	}
}

func TestNewTeamName_TooLong(t *testing.T) {
	long := strings.Repeat("a", 101)
	_, err := team.NewTeamName(long)
	if err == nil {
		t.Fatal("expected error for name exceeding 100 chars")
	}
	ae, ok := err.(*apperror.AppError)
	if !ok || ae.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestNewTeamName_MaxLengthValid(t *testing.T) {
	exact := strings.Repeat("a", 100)
	_, err := team.NewTeamName(exact)
	if err != nil {
		t.Errorf("100-char name should be valid, got: %v", err)
	}
}

func TestRestoreTeamName(t *testing.T) {
	n := team.RestoreTeamName("My Team")
	if n.Value() != "My Team" {
		t.Errorf("RestoreTeamName Value() = %q, want %q", n.Value(), "My Team")
	}
}
