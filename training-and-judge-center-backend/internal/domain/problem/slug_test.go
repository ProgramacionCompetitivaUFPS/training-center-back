package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestNewSlug(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		errCode string
	}{
		{
			name:    "Valid slug",
			input:   "sum-two-numbers",
			wantErr: false,
		},
		{
			name:    "Valid slug without hyphens",
			input:   "sum",
			wantErr: false,
		},
		{
			name:    "Valid slug with numbers",
			input:   "prob-001",
			wantErr: false,
		},
		{
			name:    "Empty slug",
			input:   "",
			wantErr: true,
			errCode: "VALIDATION_ERROR",
		},
		{
			name:    "Slug too short",
			input:   "ab",
			wantErr: true,
			errCode: "SLUG_TOO_SHORT",
		},
		{
			name:    "Slug too long",
			input:   "abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz",
			wantErr: true,
			errCode: "SLUG_TOO_LONG",
		},
		{
			name:    "Slug with uppercase",
			input:   "Sum-two-numbers",
			wantErr: true,
			errCode: "INVALID_SLUG_FORMAT",
		},
		{
			name:    "Slug starts with hyphen",
			input:   "-sum-two-numbers",
			wantErr: true,
			errCode: "INVALID_SLUG_FORMAT",
		},
		{
			name:    "Slug ends with hyphen",
			input:   "sum-two-numbers-",
			wantErr: true,
			errCode: "INVALID_SLUG_FORMAT",
		},
		{
			name:    "Slug with consecutive hyphens",
			input:   "sum--two",
			wantErr: true,
			errCode: "INVALID_SLUG_FORMAT",
		},
		{
			name:    "Slug with spaces",
			input:   "sum two numbers",
			wantErr: true,
			errCode: "INVALID_SLUG_FORMAT",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slug, err := problem.NewSlug(tt.input)

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

				if appErr.Code != tt.errCode {
					t.Errorf("expected error code %s, got %s", tt.errCode, appErr.Code)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
					return
				}

				if slug.String() != tt.input {
					t.Errorf("expected slug value %s, got %s", tt.input, slug.String())
				}
			}
		})
	}
}
