package material

import (
	"testing"

	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestNewTags(t *testing.T) {
	tests := []struct {
		name    string
		input   []string
		wantErr bool
	}{
		{"nil input", nil, false},
		{"empty slice", []string{}, false},
		{"valid tags", []string{"announcement", "week-1", "data_structures", "dp2024"}, false},
		{"single char min", []string{"ab"}, false},
		{"exactly 50 chars", []string{"abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmn"}, false},
		{"uppercase", []string{"Announcement"}, true},
		{"space in tag", []string{"week one"}, true},
		{"starts with hyphen", []string{"-start"}, true},
		{"ends with hyphen", []string{"end-"}, true},
		{"starts with underscore", []string{"_start"}, true},
		{"ends with underscore", []string{"end_"}, true},
		{"consecutive hyphens", []string{"too--many"}, true},
		{"consecutive underscores", []string{"bad__tag"}, true},
		{"1 char", []string{"a"}, true},
		{"51 chars", []string{"abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmno"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTags(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewTags(%v) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestNewTags_InvalidReturnsValidationError(t *testing.T) {
	_, err := NewTags([]string{"Invalid-Tag"})
	if err == nil {
		t.Fatal("expected error for invalid tag")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %v", err)
	}
}

func TestNewTags_Deduplication(t *testing.T) {
	tags, err := NewTags([]string{"important", "important", "news"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	values := tags.Values()
	if len(values) != 2 {
		t.Errorf("expected 2 tags after dedup, got %d: %v", len(values), values)
	}
}
