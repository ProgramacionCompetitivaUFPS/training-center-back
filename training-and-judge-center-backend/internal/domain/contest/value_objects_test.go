package contest_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/contest"
)

func TestNewContestName(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid name", "Weekly Contest #1", false},
		{"exactly 200 runes", strings.Repeat("a", 200), false},
		{"empty string", "", true},
		{"only whitespace", "   ", true},
		{"201 runes", strings.Repeat("a", 201), true},
		{"trimmed to valid", "  Valid Name  ", false},
		{"unicode characters", "Concurso Ñoño 2026", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := contest.NewContestName(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestNewContestName_Trimming(t *testing.T) {
	n, err := contest.NewContestName("  My Contest  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n.Value() != "My Contest" {
		t.Errorf("expected trimmed value, got %q", n.Value())
	}
}

func TestNewPenalty(t *testing.T) {
	cases := []struct {
		name    string
		input   int
		wantErr bool
	}{
		{"minimum (0)", 0, false},
		{"default (20)", 20, false},
		{"maximum (1440)", 1440, false},
		{"negative", -1, true},
		{"above maximum", 1441, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := contest.NewPenalty(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestContestProblemLetter(t *testing.T) {
	cases := []struct {
		order  int
		letter string
	}{
		{1, "A"},
		{2, "B"},
		{3, "C"},
		{26, "Z"},
	}
	for _, tc := range cases {
		cp := contest.RestoreContestProblem("id", "pid", tc.order)
		if got := cp.Letter(); got != tc.letter {
			t.Errorf("order %d: expected letter %q, got %q", tc.order, tc.letter, got)
		}
	}
}
