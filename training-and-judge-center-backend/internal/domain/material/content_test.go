package material_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/material"
)

func TestNewContent(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"empty string", "", false},
		{"valid content", "Hello, world!", false},
		{"exactly 50000 chars", strings.Repeat("a", 50000), false},
		{"50001 chars", strings.Repeat("a", 50001), true},
		{"multibyte chars at limit", strings.Repeat("é", 50000), false},
		{"multibyte chars over limit", strings.Repeat("é", 50001), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := material.NewContent(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContent() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
