package problem_test

import (
	"testing"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

func TestNewSolutionFile(t *testing.T) {
	tests := []struct {
		name    string
		lang    string
		wantErr bool
	}{
		{"valid language returns file", "cpp20", false},
		{"empty language returns error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := problem.NewSolutionFile("solution.cpp", "file-key", tt.lang)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if !tt.wantErr {
				if f.Filename() != "solution.cpp" {
					t.Errorf("Filename: got %q, want solution.cpp", f.Filename())
				}
				if f.FileKey() != "file-key" {
					t.Errorf("FileKey: got %q, want file-key", f.FileKey())
				}
				if f.Language() != tt.lang {
					t.Errorf("Language: got %q, want %q", f.Language(), tt.lang)
				}
			}
		})
	}
}

func TestRestoreJudgingFile_PreservesFields(t *testing.T) {
	f := problem.RestoreJudgingFile("sol.cpp", "key-1", "cpp20")
	if f.Filename() != "sol.cpp" {
		t.Errorf("Filename(): got %q, want sol.cpp", f.Filename())
	}
	if f.FileKey() != "key-1" {
		t.Errorf("FileKey(): got %q, want key-1", f.FileKey())
	}
	if f.Language() != "cpp20" {
		t.Errorf("Language(): got %q, want cpp20", f.Language())
	}
}

func TestNewVerifierFile(t *testing.T) {
	tests := []struct {
		name    string
		lang    string
		wantErr bool
	}{
		{"valid language returns file", "cpp20", false},
		{"empty language returns error", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := problem.NewVerifierFile("checker.cpp", "file-key", tt.lang)
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
