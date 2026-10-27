package judge

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestGetCheckerSource_Success(t *testing.T) {
	checkerJSON := []byte(`{"filename":"checker.cpp","fileKey":"problems/abc/checker/checker.cpp","language":"cpp20"}`)

	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = checkerJSON
				return nil
			}}
		},
	})

	source, err := provider.GetCheckerSource(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source == nil {
		t.Fatal("expected a source, got nil")
	}
	if source.Filename != "checker.cpp" || source.FileKey != "problems/abc/checker/checker.cpp" {
		t.Errorf("source: got %+v", source)
	}
	if source.Language.String() != "cpp20" {
		t.Errorf("Language: got %q, want cpp20", source.Language.String())
	}
}

func TestGetCheckerSource_NoneUploaded_ReturnsNilNil(t *testing.T) {
	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = nil
				return nil
			}}
		},
	})

	source, err := provider.GetCheckerSource(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != nil {
		t.Errorf("expected nil source, got %+v", source)
	}
}

func TestGetCheckerSource_NotFound(t *testing.T) {
	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := provider.GetCheckerSource(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestGetCheckerSource_DBError_ReturnsInternal(t *testing.T) {
	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := provider.GetCheckerSource(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetCheckerSource_MalformedJSON_ReturnsInternal(t *testing.T) {
	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = []byte(`not valid json`)
				return nil
			}}
		},
	})

	_, err := provider.GetCheckerSource(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetCheckerSource_InvalidLanguage_ReturnsInternal(t *testing.T) {
	checkerJSON := []byte(`{"filename":"checker.rs","fileKey":"problems/abc/checker/checker.rs","language":"rust"}`)

	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = checkerJSON
				return nil
			}}
		},
	})

	_, err := provider.GetCheckerSource(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetValidatorSource_Success(t *testing.T) {
	validatorJSON := []byte(`{"filename":"validator.py","fileKey":"problems/abc/validator/validator.py","language":"python310"}`)

	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = validatorJSON
				return nil
			}}
		},
	})

	source, err := provider.GetValidatorSource(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source == nil || source.Language.String() != "python310" {
		t.Errorf("source: got %+v", source)
	}
}

func TestGetValidatorSource_NoneUploaded_ReturnsNilNil(t *testing.T) {
	provider := NewJudgingSourceProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = nil
				return nil
			}}
		},
	})

	source, err := provider.GetValidatorSource(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != nil {
		t.Errorf("expected nil source, got %+v", source)
	}
}
