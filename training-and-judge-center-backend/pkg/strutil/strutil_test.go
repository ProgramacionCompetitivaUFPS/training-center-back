package strutil_test

import (
	"testing"
	"unicode/utf8"

	"github.com/training-judge-center/backend/pkg/strutil"
)

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxBytes int
		want     string
	}{
		{"shorter than the limit", "abc", 10, "abc"},
		{"exactly the limit", "abc", 3, "abc"},
		{"longer than the limit", "abcdef", 3, "abc"},
		{"zero limit", "abc", 0, ""},
		// "ñ" is two bytes: cutting at 2 would leave half of it.
		{"cut inside a multi-byte rune", "añ", 2, "a"},
		{"cut right after a multi-byte rune", "añ", 3, "añ"},
		{"every byte of the last rune dropped", "€", 2, ""},
		// U+FFFD is a legitimate three-byte rune, not a broken one.
		{"keeps a real replacement char", "a�", 4, "a�"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strutil.Truncate(tt.input, tt.maxBytes)

			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
			if !utf8.ValidString(got) {
				t.Errorf("got invalid UTF-8: %q", got)
			}
		})
	}
}
