package material

import (
	"strings"
	"testing"
)

func TestNewContent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string is valid", "", false},
		{"normal content", "Hello world", false},
		{"exactly 50000 runes", strings.Repeat("a", 50000), false},
		{"50001 runes", strings.Repeat("a", 50001), true},
		{"multibyte chars within limit", strings.Repeat("á", 50000), false},
		{"multibyte chars over limit", strings.Repeat("á", 50001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewContent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewEmptyContent(t *testing.T) {
	c := NewEmptyContent()
	if c.String() != "" {
		t.Errorf("expected empty string, got %q", c.String())
	}
}
