package problem

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testArtifactNow = time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)

const testArtifactProblemID = "bbbbbbbb-0000-0000-0000-000000000001"

func TestSetCheckerCompiledKey_Success(t *testing.T) {
	existing := []byte(`{"filename":"checker.cpp","fileKey":"problems/abc/checker/checker.cpp","language":"cpp20"}`)
	var updatedJSON []byte

	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = existing
				return nil
			}}
		},
		execFn: func(_ context.Context, _ string, args ...interface{}) (pgconn.CommandTag, error) {
			updatedJSON = args[1].([]byte)
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	})

	err := writer.SetCheckerCompiledKey(context.Background(), testArtifactProblemID, "problems/abc/checker/compiled", testArtifactNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got dbJudgingFile
	if err := json.Unmarshal(updatedJSON, &got); err != nil {
		t.Fatalf("could not decode updated JSON: %v", err)
	}
	if got.Filename != "checker.cpp" || got.FileKey != "problems/abc/checker/checker.cpp" {
		t.Errorf("expected filename/fileKey preserved, got %+v", got)
	}
	if got.CompiledKey == nil || *got.CompiledKey != "problems/abc/checker/compiled" {
		t.Errorf("CompiledKey: got %v, want set", got.CompiledKey)
	}
	if got.CompiledAt == nil || !got.CompiledAt.Equal(testArtifactNow) {
		t.Errorf("CompiledAt: got %v, want %v", got.CompiledAt, testArtifactNow)
	}
}

func TestSetCheckerCompiledKey_ProblemNotFound(t *testing.T) {
	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	err := writer.SetCheckerCompiledKey(context.Background(), testArtifactProblemID, "key", testArtifactNow)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestSetCheckerCompiledKey_SelectError_ReturnsInternal(t *testing.T) {
	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	err := writer.SetCheckerCompiledKey(context.Background(), testArtifactProblemID, "key", testArtifactNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestSetCheckerCompiledKey_NoCheckerUploaded_ReturnsInternal(t *testing.T) {
	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = nil
				return nil
			}}
		},
	})

	err := writer.SetCheckerCompiledKey(context.Background(), testArtifactProblemID, "key", testArtifactNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestSetCheckerCompiledKey_MalformedJSON_ReturnsInternal(t *testing.T) {
	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = []byte(`not valid json`)
				return nil
			}}
		},
	})

	err := writer.SetCheckerCompiledKey(context.Background(), testArtifactProblemID, "key", testArtifactNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestSetCheckerCompiledKey_UpdateError_ReturnsInternal(t *testing.T) {
	existing := []byte(`{"filename":"checker.cpp","fileKey":"problems/abc/checker/checker.cpp","language":"cpp20"}`)

	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = existing
				return nil
			}}
		},
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := writer.SetCheckerCompiledKey(context.Background(), testArtifactProblemID, "key", testArtifactNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestSetValidatorCompiledKey_Success(t *testing.T) {
	existing := []byte(`{"filename":"validator.py","fileKey":"problems/abc/validator/validator.py","language":"python310"}`)
	var updatedJSON []byte

	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*[]byte)) = existing
				return nil
			}}
		},
		execFn: func(_ context.Context, _ string, args ...interface{}) (pgconn.CommandTag, error) {
			updatedJSON = args[1].([]byte)
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	})

	err := writer.SetValidatorCompiledKey(context.Background(), testArtifactProblemID, "problems/abc/validator/compiled", testArtifactNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got dbJudgingFile
	if err := json.Unmarshal(updatedJSON, &got); err != nil {
		t.Fatalf("could not decode updated JSON: %v", err)
	}
	if got.CompiledKey == nil || *got.CompiledKey != "problems/abc/validator/compiled" {
		t.Errorf("CompiledKey: got %v, want set", got.CompiledKey)
	}
}

func TestSetValidatorCompiledKey_ProblemNotFound(t *testing.T) {
	writer := NewJudgingArtifactWriter(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	err := writer.SetValidatorCompiledKey(context.Background(), testArtifactProblemID, "key", testArtifactNow)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}
