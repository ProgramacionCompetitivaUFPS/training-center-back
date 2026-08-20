package judge

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const testProblemID = "bbbbbbbb-0000-0000-0000-000000000001"

func TestGetLimits_CheckerCompiled_ReturnsCompiledPathAndLanguage(t *testing.T) {
	tl := 2000
	mb := 256
	checkerJSON := []byte(`{"filename":"checker.cpp","fileKey":"problems/abc/checker/checker.cpp","language":"cpp20","compiledKey":"problems/abc/checker/compiled"}`)

	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(**int)) = &tl
				*(dest[1].(**int)) = &mb
				*(dest[2].(*[]byte)) = checkerJSON
				return nil
			}}
		},
	})

	limits, err := provider.GetLimits(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limits.TimeLimitMs != 2000 {
		t.Errorf("TimeLimitMs: got %d, want 2000", limits.TimeLimitMs)
	}
	if limits.MemoryKb != 256*1024 {
		t.Errorf("MemoryKb: got %d, want %d", limits.MemoryKb, 256*1024)
	}
	if !limits.HasCustomChecker {
		t.Error("HasCustomChecker: got false, want true")
	}
	if limits.CheckerPath != "problems/abc/checker/compiled" {
		t.Errorf("CheckerPath: got %q, want the compiled key, not the source fileKey", limits.CheckerPath)
	}
	if limits.CheckerLanguage.String() != "cpp20" {
		t.Errorf("CheckerLanguage: got %q, want cpp20", limits.CheckerLanguage.String())
	}
}

// TestGetLimits_CheckerNotYetCompiled_FallsBackToTokenComparison covers an
// anomaly that publish validation should prevent (a checker that reached
// PUBLISHED without ever compiling) — CheckerPath stays empty rather than
// pointing at unusable source, so judging falls back to token comparison
// instead of failing every submission for this problem.
func TestGetLimits_CheckerNotYetCompiled_FallsBackToTokenComparison(t *testing.T) {
	tl := 2000
	mb := 256
	checkerJSON := []byte(`{"filename":"checker.cpp","fileKey":"problems/abc/checker/checker.cpp","language":"cpp20"}`)

	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(**int)) = &tl
				*(dest[1].(**int)) = &mb
				*(dest[2].(*[]byte)) = checkerJSON
				return nil
			}}
		},
	})

	limits, err := provider.GetLimits(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !limits.HasCustomChecker {
		t.Error("HasCustomChecker: got false, want true")
	}
	if limits.CheckerPath != "" {
		t.Errorf("CheckerPath: got %q, want empty", limits.CheckerPath)
	}
}

func TestGetLimits_CheckerInvalidLanguage_ReturnsInternal(t *testing.T) {
	tl := 2000
	mb := 256
	checkerJSON := []byte(`{"filename":"checker.rs","fileKey":"problems/abc/checker/checker.rs","language":"rust","compiledKey":"problems/abc/checker/compiled"}`)

	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(**int)) = &tl
				*(dest[1].(**int)) = &mb
				*(dest[2].(*[]byte)) = checkerJSON
				return nil
			}}
		},
	})

	_, err := provider.GetLimits(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetLimits_NullChecker(t *testing.T) {
	tl := 1000
	mb := 128

	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(**int)) = &tl
				*(dest[1].(**int)) = &mb
				*(dest[2].(*[]byte)) = nil
				return nil
			}}
		},
	})

	limits, err := provider.GetLimits(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limits.HasCustomChecker {
		t.Error("HasCustomChecker: got true, want false")
	}
	if limits.CheckerPath != "" {
		t.Errorf("CheckerPath: got %q, want empty", limits.CheckerPath)
	}
}

func TestGetLimits_NullLimits(t *testing.T) {
	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(**int)) = nil
				*(dest[1].(**int)) = nil
				*(dest[2].(*[]byte)) = nil
				return nil
			}}
		},
	})

	limits, err := provider.GetLimits(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if limits.TimeLimitMs != 0 {
		t.Errorf("TimeLimitMs: got %d, want 0", limits.TimeLimitMs)
	}
	if limits.MemoryKb != 0 {
		t.Errorf("MemoryKb: got %d, want 0", limits.MemoryKb)
	}
}

func TestGetLimits_NotFound(t *testing.T) {
	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := provider.GetLimits(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestGetLimits_MalformedCheckerJSON_ReturnsInternal(t *testing.T) {
	tl := 1000
	mb := 128
	badJSON := []byte(`not valid json`)

	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(**int)) = &tl
				*(dest[1].(**int)) = &mb
				*(dest[2].(*[]byte)) = badJSON
				return nil
			}}
		},
	})

	_, err := provider.GetLimits(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetLimits_DBError_ReturnsInternal(t *testing.T) {
	provider := NewProblemProvider(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := provider.GetLimits(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}
