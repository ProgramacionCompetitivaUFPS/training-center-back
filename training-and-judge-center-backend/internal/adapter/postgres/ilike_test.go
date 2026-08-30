package postgres_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/adapter/postgres"
)

func TestEscapeILIKE(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no special characters", "hello", "hello"},
		{"escapes percent", "50%", `50\%`},
		{"escapes underscore", "foo_bar", `foo\_bar`},
		{"escapes backslash", `C:\path`, `C:\\path`},
		{"escapes backslash before percent/underscore literals", `\%\_`, `\\\%\\\_`},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := postgres.EscapeILIKE(tt.input)
			if got != tt.want {
				t.Errorf("EscapeILIKE(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
