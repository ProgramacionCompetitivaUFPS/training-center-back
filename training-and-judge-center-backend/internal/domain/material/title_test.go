package material

import (
	"strings"
	"testing"
)

func TestNewTitle(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid title", "Hello World", false},
		{"empty string", "", true},
		{"only spaces", "   ", true},
		{"exactly 200 chars", strings.Repeat("a", 200), false},
		{"201 chars", strings.Repeat("a", 201), true},
		{"unicode NFKC normalized", "ﬁle", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTitle(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTitle(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestTitle_NKFCNormalization(t *testing.T) {
	// ﬁ is a ligature that normalizes to "fi" under NFKC
	title, err := NewTitle("ﬁle")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if title.String() != "file" {
		t.Errorf("expected NFKC normalized value 'file', got %q", title.String())
	}
}
