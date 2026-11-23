package contest_test

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
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
		cp := contest.RestoreContestProblem("id", "pid", tc.order, nil)
		if got := cp.Letter(); got != tc.letter {
			t.Errorf("order %d: expected letter %q, got %q", tc.order, tc.letter, got)
		}
	}
}

func TestContestProblemColor(t *testing.T) {
	t.Run("without color", func(t *testing.T) {
		cp := contest.RestoreContestProblem("id", "pid", 1, nil)
		if cp.Color() != nil {
			t.Errorf("Color() = %v, want nil", cp.Color())
		}
	})

	t.Run("with color", func(t *testing.T) {
		color := "#aabbcc"
		cp := contest.RestoreContestProblem("id", "pid", 1, &color)
		if cp.Color() == nil || cp.Color().Value() != "#aabbcc" {
			t.Errorf("Color() = %v, want #aabbcc", cp.Color())
		}
	})
}

func TestNewParticipationMode(t *testing.T) {
	tests := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{"INDIVIDUAL", "INDIVIDUAL", false},
		{"TEAM", "TEAM", false},
		{"MIXED", "MIXED", false},
		{"", "", true},
		{"individual", "", true},
		{"SOLO", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			got, err := contest.NewParticipationMode(tt.raw)

			if tt.wantErr {
				assertValidationField(t, tt.raw, err, "participationMode")
				return
			}
			if err != nil {
				t.Fatalf("NewParticipationMode(%q): %v", tt.raw, err)
			}
			if got.String() != tt.want {
				t.Errorf("String() = %q, want %q", got.String(), tt.want)
			}
		})
	}
}

func TestParticipationMode_AllowsTeams(t *testing.T) {
	tests := []struct {
		mode contest.ParticipationMode
		want bool
	}{
		{contest.ParticipationModeIndividual, false},
		{contest.ParticipationModeTeam, true},
		{contest.ParticipationModeMixed, true},
	}

	for _, tt := range tests {
		t.Run(tt.mode.String(), func(t *testing.T) {
			if got := tt.mode.AllowsTeams(); got != tt.want {
				t.Errorf("AllowsTeams() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRestoreParticipationMode_KeepsValueWithoutValidating(t *testing.T) {
	if got := contest.RestoreParticipationMode("LEGACY").String(); got != "LEGACY" {
		t.Errorf("String() = %q, want LEGACY", got)
	}
}

func TestNewTeamSize(t *testing.T) {
	tests := []struct {
		name       string
		min, max   int
		wantFields []string
	}{
		{"a team of exactly one", 1, 1, nil},
		{"a range", 2, 5, nil},
		{"a minimum below one", 0, 3, []string{"teamSizeMin"}},
		{"a maximum below the minimum", 3, 2, []string{"teamSizeMax"}},
		{"both wrong reports both", 0, -1, []string{"teamSizeMin", "teamSizeMax"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := contest.NewTeamSize(tt.min, tt.max)

			if tt.wantFields == nil {
				if err != nil {
					t.Fatalf("NewTeamSize(%d, %d): %v", tt.min, tt.max, err)
				}
				if got.Min() != tt.min || got.Max() != tt.max {
					t.Errorf("got %d..%d, want %d..%d", got.Min(), got.Max(), tt.min, tt.max)
				}
				return
			}
			appErr := validationDetails(t, err)
			var fields []string
			for _, d := range appErr {
				fields = append(fields, d.Field)
			}
			if !reflect.DeepEqual(fields, tt.wantFields) {
				t.Errorf("fields = %v, want %v", fields, tt.wantFields)
			}
		})
	}
}

func TestTeamSize_IsValidCount(t *testing.T) {
	size := contest.RestoreTeamSize(2, 5)
	tests := []struct {
		n    int
		want bool
	}{
		{1, false},
		{2, true},
		{5, true},
		{6, false},
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.n), func(t *testing.T) {
			if got := size.IsValidCount(tt.n); got != tt.want {
				t.Errorf("IsValidCount(%d) = %v, want %v", tt.n, got, tt.want)
			}
		})
	}
}

func TestRestoreTeamSize_KeepsValuesWithoutValidating(t *testing.T) {
	size := contest.RestoreTeamSize(0, 0)

	if size.Min() != 0 || size.Max() != 0 {
		t.Errorf("got %d..%d, want 0..0", size.Min(), size.Max())
	}
}

func TestRestorePenalty_KeepsValueWithoutValidating(t *testing.T) {
	if got := contest.RestorePenalty(5000).Value(); got != 5000 {
		t.Errorf("Value() = %d, want 5000", got)
	}
}

func TestRestoreContestName_KeepsValueWithoutTrimming(t *testing.T) {
	got := contest.RestoreContestName("  padded  ")

	if got.Value() != "  padded  " || got.String() != "  padded  " {
		t.Errorf("Value()/String() = %q/%q, want the stored text untouched", got.Value(), got.String())
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status contest.Status
		want   string
	}{
		{contest.StatusScheduled, "SCHEDULED"},
		{contest.StatusActive, "ACTIVE"},
		{contest.StatusFinished, "FINISHED"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func validationDetails(t *testing.T, err error) []apperror.FieldError {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError {
		t.Fatalf("expected a validation apperror, got %v", err)
	}
	return appErr.Details
}
