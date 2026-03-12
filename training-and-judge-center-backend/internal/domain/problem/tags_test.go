package problem_test

import (
	"reflect"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestNewTags(t *testing.T) {
	allowedTags := map[string]struct{}{
		"math":     {},
		"dp":       {},
		"graphs":   {},
		"beginner": {},
	}

	tests := []struct {
		name        string
		input       []string
		allowed     map[string]struct{}
		wantErr     bool
		expectedErr int // Expected number of FieldErrors
	}{
		{
			name:    "Valid tags",
			input:   []string{"math", "dp"},
			allowed: allowedTags,
			wantErr: false,
		},
		{
			name:    "Nil tags",
			input:   nil,
			allowed: allowedTags,
			wantErr: false,
		},
		{
			name:    "Empty tags array",
			input:   []string{},
			allowed: allowedTags,
			wantErr: false,
		},
		{
			name:        "Tag not in allowed list",
			input:       []string{"math", "unknown"},
			allowed:     allowedTags,
			wantErr:     true,
			expectedErr: 1,
		},
		{
			name:        "Empty tag string",
			input:       []string{"math", ""},
			allowed:     allowedTags,
			wantErr:     true,
			expectedErr: 1, // "Tag cannot be empty"
		},
		{
			name:        "Duplicate tags",
			input:       []string{"math", "math"},
			allowed:     allowedTags,
			wantErr:     true,
			expectedErr: 1, // "Duplicate tag: math"
		},
		{
			name:        "Multiple errors at once",
			input:       []string{"math", "math", "unknown", ""},
			allowed:     allowedTags,
			wantErr:     true,
			expectedErr: 3, // duplicate math, unknown tag, empty tag
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tags, err := problem.NewTags(tt.input, tt.allowed)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}

				appErr, ok := err.(*apperror.AppError)
				if !ok {
					t.Errorf("expected error to be of type *apperror.AppError")
					return
				}

				if appErr.Code != "VALIDATION_ERROR" {
					t.Errorf("expected code VALIDATION_ERROR, got %s", appErr.Code)
				}

				if len(appErr.Details) != tt.expectedErr {
					t.Errorf("expected %d field errors, got %d", tt.expectedErr, len(appErr.Details))
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				expected := tt.input
				if expected == nil {
					expected = []string{}
				}

				if !reflect.DeepEqual(tags.Values(), expected) {
					t.Errorf("expected tags %v, got %v", expected, tags.Values())
				}
			}
		})
	}
}
