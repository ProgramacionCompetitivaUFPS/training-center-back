package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Test fixtures ─────────────────────────────────────────────────────────────

var testNow = time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC)

const (
	testTokenID   = "token-aaaaaaaa-0001"
	testUserID    = "user-bbbbbbbb-0001"
	testFamilyID  = "family-cccccccc-0001"
	testTokenHash = "abcd1234hash"
)

func activeRefreshToken() *domainUser.RefreshToken {
	token, err := domainUser.NewRefreshToken(testTokenID, testUserID, testFamilyID, testTokenHash, nil, nil, false, testNow)
	if err != nil {
		panic(err)
	}
	return token
}

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

// ── mockRows (unused by RefreshTokenRepository today, kept for interface parity) ──

type mockRows struct {
	scanFns []func(dest ...any) error
	idx     int
	err     error
}

func (m *mockRows) Next() bool                                   { m.idx++; return m.idx <= len(m.scanFns) }
func (m *mockRows) Close()                                       {}
func (m *mockRows) Err() error                                   { return m.err }
func (m *mockRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Values() ([]any, error)                       { return nil, nil }
func (m *mockRows) RawValues() [][]byte                          { return nil }
func (m *mockRows) Scan(dest ...any) error {
	if m.idx < 1 || m.idx > len(m.scanFns) {
		return errors.New("scan called out of bounds")
	}
	return m.scanFns[m.idx-1](dest...)
}
func (m *mockRows) Conn() *pgx.Conn { return nil }

// ── scan helper for a refresh token row ────────────────────────────────────────

func refreshTokenScanFn(dest ...any) error {
	*(dest[0].(*string)) = testTokenID
	*(dest[1].(*string)) = testUserID
	*(dest[2].(*string)) = testFamilyID
	*(dest[3].(*[]byte)) = []byte(testTokenHash)
	*(dest[4].(*time.Time)) = testNow
	*(dest[5].(*time.Time)) = testNow.Add(domainUser.DefaultSessionCeiling)
	*(dest[6].(**time.Time)) = nil
	*(dest[7].(**string)) = nil
	*(dest[8].(**string)) = nil
	*(dest[9].(**string)) = nil
	return nil
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestSave_Success(t *testing.T) {
	var capturedArgs []interface{}
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedArgs = args
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	})

	if err := repo.Save(context.Background(), activeRefreshToken()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	hashArg, ok := capturedArgs[3].([]byte)
	if !ok || string(hashArg) != testTokenHash {
		t.Errorf("expected token_hash arg %q as []byte, got %v", testTokenHash, capturedArgs[3])
	}
}

func TestSave_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.Save(context.Background(), activeRefreshToken())
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── FindByTokenHash ───────────────────────────────────────────────────────────

func TestFindByTokenHash_Success(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: refreshTokenScanFn}
		},
	})

	token, err := repo.FindByTokenHash(context.Background(), testTokenHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token.ID() != testTokenID {
		t.Errorf("expected id %q, got %q", testTokenID, token.ID())
	}
}

func TestFindByTokenHash_HashRoundTrips(t *testing.T) {
	var capturedArgs []interface{}
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, args ...interface{}) pgx.Row {
			capturedArgs = args
			return &mockRow{scanFn: refreshTokenScanFn}
		},
	})

	token, err := repo.FindByTokenHash(context.Background(), testTokenHash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	hashArg, ok := capturedArgs[0].([]byte)
	if !ok || string(hashArg) != testTokenHash {
		t.Fatalf("expected query arg %q as []byte, got %v", testTokenHash, capturedArgs[0])
	}
	if token.TokenHash() != testTokenHash {
		t.Errorf("expected scanned TokenHash() %q, got %q", testTokenHash, token.TokenHash())
	}
}

func TestFindByTokenHash_NotFound_ReturnsNil(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	token, err := repo.FindByTokenHash(context.Background(), "unknown-hash")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != nil {
		t.Errorf("expected nil token, got %v", token)
	}
}

func TestFindByTokenHash_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := repo.FindByTokenHash(context.Background(), testTokenHash)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── FindActiveByFamilyID ──────────────────────────────────────────────────────

func TestFindActiveByFamilyID_Success(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: refreshTokenScanFn}
		},
	})

	token, err := repo.FindActiveByFamilyID(context.Background(), testFamilyID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token.FamilyID() != testFamilyID {
		t.Errorf("expected familyID %q, got %q", testFamilyID, token.FamilyID())
	}
}

func TestFindActiveByFamilyID_NotFound_ReturnsNil(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	token, err := repo.FindActiveByFamilyID(context.Background(), "no-such-family")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token != nil {
		t.Errorf("expected nil token, got %v", token)
	}
}

func TestFindActiveByFamilyID_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := repo.FindActiveByFamilyID(context.Background(), testFamilyID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── Rotate ────────────────────────────────────────────────────────────────────

func successorToken() *domainUser.RefreshToken {
	old := activeRefreshToken()
	successor, err := old.Successor("new-token-id", "new-hash", nil, nil, testNow.Add(time.Minute))
	if err != nil {
		panic(err)
	}
	return successor
}

func TestRotate_Success(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	})

	rotated, err := repo.Rotate(context.Background(), testTokenHash, successorToken())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !rotated {
		t.Errorf("expected rotated=true")
	}
}

func TestRotate_ZeroRowsAffected_ReturnsFalseNoError(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 0 0"), nil
		},
	})

	rotated, err := repo.Rotate(context.Background(), testTokenHash, successorToken())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if rotated {
		t.Errorf("expected rotated=false when zero rows were affected")
	}
}

func TestRotate_DBError_ReturnsErrorNotFalse(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("connection reset")
		},
	})

	rotated, err := repo.Rotate(context.Background(), testTokenHash, successorToken())
	if err == nil {
		t.Fatalf("expected a real DB error to return err != nil, not just rotated=false")
	}
	if rotated {
		t.Errorf("expected rotated=false alongside the error")
	}
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── RevokeByFamilyID ──────────────────────────────────────────────────────────

func TestRevokeByFamilyID_Success(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 2"), nil
		},
	})

	if err := repo.RevokeByFamilyID(context.Background(), testFamilyID, testNow); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRevokeByFamilyID_ZeroRows_IsNotAnError(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	})

	if err := repo.RevokeByFamilyID(context.Background(), testFamilyID, testNow); err != nil {
		t.Fatalf("expected no error for an idempotent zero-row revoke, got %v", err)
	}
}

func TestRevokeByFamilyID_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.RevokeByFamilyID(context.Background(), testFamilyID, testNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── RevokeAllByUserID ─────────────────────────────────────────────────────────

func TestRevokeAllByUserID_Success(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 3"), nil
		},
	})

	if err := repo.RevokeAllByUserID(context.Background(), testUserID, testNow); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestRevokeAllByUserID_ZeroRows_IsNotAnError(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	})

	if err := repo.RevokeAllByUserID(context.Background(), testUserID, testNow); err != nil {
		t.Fatalf("expected no error for an idempotent zero-row revoke, got %v", err)
	}
}

func TestRevokeAllByUserID_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.RevokeAllByUserID(context.Background(), testUserID, testNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── DeleteRevokedOrExpiredBefore ──────────────────────────────────────────────

func TestDeleteRevokedOrExpiredBefore_Success(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("DELETE 5"), nil
		},
	})

	if err := repo.DeleteRevokedOrExpiredBefore(context.Background(), testNow); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteRevokedOrExpiredBefore_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRefreshTokenRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.DeleteRevokedOrExpiredBefore(context.Background(), testNow)
	assertAppErrorKind(t, err, apperror.KindInternal)
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
