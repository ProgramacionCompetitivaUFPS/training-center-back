package contest_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/contest"
)

func TestNewContestName(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantErr   bool
		wantField string
	}{
		{"valid name", "Weekly Contest #1", false, ""},
		{"exactly 200 runes", strings.Repeat("a", 200), false, ""},
		{"empty string", "", true, "name"},
		{"only whitespace", "   ", true, "name"},
		{"201 runes", strings.Repeat("a", 201), true, "name"},
		{"trimmed to valid", "  Valid Name  ", false, ""},
		{"unicode characters", "Concurso Ñoño 2026", false, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := contest.NewContestName(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}
			if tc.wantField != "" {
				assertValidationField(t, "NewContestName("+tc.input+")", err, tc.wantField)
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
		name      string
		input     int
		wantErr   bool
		wantField string
	}{
		{"minimum (0)", 0, false, ""},
		{"default (20)", 20, false, ""},
		{"maximum (1440)", 1440, false, ""},
		{"negative", -1, true, "penalty"},
		{"above maximum", 1441, true, "penalty"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := contest.NewPenalty(tc.input)
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
				return
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
				return
			}
			if tc.wantField != "" {
				assertValidationField(t, "NewPenalty", err, tc.wantField)
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
		{27, "AA"},
		{28, "AB"},
		{52, "AZ"},
		{53, "BA"},
		{702, "ZZ"},
		{703, "AAA"},
	}
	for _, tc := range cases {
		cp := contest.RestoreContestProblem("id", "pid", tc.order)
		if got := cp.Letter(); got != tc.letter {
			t.Errorf("order %d: expected letter %q, got %q", tc.order, tc.letter, got)
		}
	}
}
