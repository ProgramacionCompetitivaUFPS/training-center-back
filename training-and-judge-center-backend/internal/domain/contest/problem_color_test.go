package contest_test

import (
	"errors"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestNewProblemColor(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"lowercase hex", "#a1b2c3", "#a1b2c3", false},
		{"uppercase is lowercased", "#AABBCC", "#aabbcc", false},
		{"mixed case is lowercased", "#aAbBcC", "#aabbcc", false},
		{"black", "#000000", "#000000", false},
		{"empty", "", "", true},
		{"color name", "red", "", true},
		{"short hex", "#FFF", "", true},
		{"missing hash", "AABBCC", "", true},
		{"non-hex digit", "#GGGGGG", "", true},
		{"too long", "#aabbccd", "", true},
		{"alpha channel", "#aabbccdd", "", true},
		{"surrounding space", " #aabbcc ", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := contest.NewProblemColor(tc.input)
			if tc.wantErr {
				var appErr *apperror.AppError
				if !errors.As(err, &appErr) {
					t.Fatalf("expected *apperror.AppError, got %v", err)
				}
				if appErr.Code != contest.ErrCodeInvalidProblemColor {
					t.Errorf("Code = %q, want %q", appErr.Code, contest.ErrCodeInvalidProblemColor)
				}
				if appErr.Kind != apperror.KindBadRequest {
					t.Errorf("Kind = %q, want %q", appErr.Kind, apperror.KindBadRequest)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Value() != tc.want {
				t.Errorf("Value() = %q, want %q", got.Value(), tc.want)
			}
		})
	}
}

func TestRestoreProblemColor_KeepsValueWithoutValidating(t *testing.T) {
	if got := contest.RestoreProblemColor("#123456").Value(); got != "#123456" {
		t.Errorf("Value() = %q, want %q", got, "#123456")
	}
}
