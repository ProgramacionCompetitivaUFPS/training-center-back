package judge

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetSolutions_Success(t *testing.T) {
	solutionsJSON := []byte(`[{"filename":"sol.cpp","fileKey":"problems/abc/sol.cpp","language":"cpp20"}]`)

	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = solutionsJSON
				return nil
			}}
		},
	})

	solutions, err := provider.GetSolutions(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(solutions) != 1 {
		t.Fatalf("expected 1 solution, got %d", len(solutions))
	}
	if solutions[0].FileKey != "problems/abc/sol.cpp" {
		t.Errorf("FileKey: got %q, want %q", solutions[0].FileKey, "problems/abc/sol.cpp")
	}
	if solutions[0].Language.String() != "cpp20" {
		t.Errorf("Language: got %q, want %q", solutions[0].Language.String(), "cpp20")
	}
}

func TestGetSolutions_MultipleLanguages(t *testing.T) {
	solutionsJSON := []byte(`[
		{"filename":"sol.cpp","fileKey":"problems/abc/sol.cpp","language":"cpp20"},
		{"filename":"Sol.java","fileKey":"problems/abc/Sol.java","language":"java17"}
	]`)

	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = solutionsJSON
				return nil
			}}
		},
	})

	solutions, err := provider.GetSolutions(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(solutions) != 2 {
		t.Fatalf("expected 2 solutions, got %d", len(solutions))
	}
	if solutions[1].Language.String() != "java17" {
		t.Errorf("solutions[1].Language: got %q, want %q", solutions[1].Language.String(), "java17")
	}
}

func TestGetSolutions_NullColumn_ReturnsEmpty(t *testing.T) {
	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = nil
				return nil
			}}
		},
	})

	solutions, err := provider.GetSolutions(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(solutions) != 0 {
		t.Errorf("expected empty slice, got %d", len(solutions))
	}
}

func TestGetSolutions_NotFound(t *testing.T) {
	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := provider.GetSolutions(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestGetSolutions_DBError_ReturnsInternal(t *testing.T) {
	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := provider.GetSolutions(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetSolutions_MalformedJSON_ReturnsInternal(t *testing.T) {
	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = []byte(`not valid json`)
				return nil
			}}
		},
	})

	_, err := provider.GetSolutions(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetSolutions_InvalidLanguage_ReturnsInternal(t *testing.T) {
	solutionsJSON := []byte(`[{"filename":"sol.rs","fileKey":"problems/abc/sol.rs","language":"rust"}]`)

	provider := NewSolutionProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = solutionsJSON
				return nil
			}}
		},
	})

	_, err := provider.GetSolutions(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}
