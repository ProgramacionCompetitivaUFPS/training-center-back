package judge

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── mockQuerier ───────────────────────────────────────────────────────────────

type mockQuerier struct {
	execFn     func(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
	queryFn    func(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	queryRowFn func(ctx context.Context, sql string, args ...interface{}) pgx.Row
}

func (m *mockQuerier) Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error) {
	if m.execFn != nil {
		return m.execFn(ctx, sql, args...)
	}
	return pgconn.NewCommandTag("OK"), nil
}

func (m *mockQuerier) Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, sql, args...)
	}
	return &mockRows{}, nil
}

func (m *mockQuerier) QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row {
	if m.queryRowFn != nil {
		return m.queryRowFn(ctx, sql, args...)
	}
	return &mockRow{scanFn: func(dest ...any) error { return nil }}
}

// ── mockRow ───────────────────────────────────────────────────────────────────

type mockRow struct {
	scanFn func(dest ...any) error
}

func (m *mockRow) Scan(dest ...any) error {
	if m.scanFn != nil {
		return m.scanFn(dest...)
	}
	return nil
}

// ── mockRows ──────────────────────────────────────────────────────────────────

type mockRows struct {
	scanFns []func(dest ...any) error
	idx     int
	err     error
}

func (m *mockRows) Next() bool                              { m.idx++; return m.idx <= len(m.scanFns) }
func (m *mockRows) Close()                                 {}
func (m *mockRows) Err() error                             { return m.err }
func (m *mockRows) CommandTag() pgconn.CommandTag          { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Values() ([]any, error)                 { return nil, nil }
func (m *mockRows) RawValues() [][]byte                    { return nil }
func (m *mockRows) Conn() *pgx.Conn                        { return nil }
func (m *mockRows) Scan(dest ...any) error {
	if m.idx < 1 || m.idx > len(m.scanFns) {
		return errors.New("scan called out of bounds")
	}
	return m.scanFns[m.idx-1](dest...)
}

// ── mockGCSReader ─────────────────────────────────────────────────────────────

type mockGCSReader struct {
	readObjectFn func(ctx context.Context, object string) (io.ReadCloser, error)
}

func (m *mockGCSReader) readObject(ctx context.Context, object string) (io.ReadCloser, error) {
	if m.readObjectFn != nil {
		return m.readObjectFn(ctx, object)
	}
	return io.NopCloser(bytes.NewReader(nil)), nil
}

// ── test helpers ──────────────────────────────────────────────────────────────

func assertAppErrorKind(t *testing.T, err error, want apperror.Kind) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Kind != want {
		t.Errorf("expected kind %q, got %q", want, appErr.Kind)
	}
}
