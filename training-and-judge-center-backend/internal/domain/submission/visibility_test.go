package submission_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func TestNewVisibility(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid PUBLIC", "PUBLIC", false},
		{"valid PRIVATE", "PRIVATE", false},
		{"lowercase public rejected", "public", true},
		{"lowercase private rejected", "private", true},
		{"empty string rejected", "", true},
		{"unknown value rejected", "UNKNOWN", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := submission.NewVisibility(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("NewVisibility(%q): expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("NewVisibility(%q): unexpected error %v", tc.input, err)
				return
			}
			if v.String() != tc.input {
				t.Errorf("NewVisibility(%q).String() = %q, want %q", tc.input, v.String(), tc.input)
			}
		})
	}
}

func TestVisibility_IsPublic_IsPrivate(t *testing.T) {
	pub := submission.NewVisibilityPublic()
	if !pub.IsPublic() {
		t.Error("expected IsPublic() true for PUBLIC")
	}
	if pub.IsPrivate() {
		t.Error("expected IsPrivate() false for PUBLIC")
	}

	priv := submission.NewVisibilityPrivate()
	if priv.IsPublic() {
		t.Error("expected IsPublic() false for PRIVATE")
	}
	if !priv.IsPrivate() {
		t.Error("expected IsPrivate() true for PRIVATE")
	}
}

func TestRestoreVisibility_DoesNotValidate(t *testing.T) {
	v := submission.RestoreVisibility("ANYTHING")
	if v.String() != "ANYTHING" {
		t.Errorf("expected ANYTHING, got %q", v.String())
	}
}
