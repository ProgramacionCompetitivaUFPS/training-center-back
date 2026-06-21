package submission

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Test fixtures ─────────────────────────────────────────────────────────────

var testNow = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

const (
	testSubID     = "sub-aaaaaaaa-0001"
	testProblemID = "bbbbbbbb-0000-0000-0000-000000000001"
	testUserID    = "cccccccc-0000-0000-0000-000000000001"
	testContestID = "dddddddd-0000-0000-0000-000000000001"
)

func pendingSubmission() *domainsubmission.Submission {
	return domainsubmission.RestoreSubmission(
		testSubID,
		testProblemID,
		shared.RestoreUserID(testUserID),
		nil,
		nil,
		domainsubmission.RestoreLanguage("cpp20"),
		"g++",
		domainsubmission.RestoreStatus("PENDING"),
		domainsubmission.RestoreVisibility("PRIVATE"),
		"gs://bucket/code.cpp",
		"",
		0,
		testNow,
		nil, nil, nil, nil,
	)
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

// ── mockRows ──────────────────────────────────────────────────────────────────

type mockRows struct {
	scanFns []func(dest ...any) error
	idx     int
	err     error
}

func (m *mockRows) Next() bool        { m.idx++; return m.idx <= len(m.scanFns) }
func (m *mockRows) Close()            {}
func (m *mockRows) Err() error        { return m.err }
func (m *mockRows) CommandTag() pgconn.CommandTag { return pgconn.CommandTag{} }
func (m *mockRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockRows) Values() ([]any, error) { return nil, nil }
func (m *mockRows) RawValues() [][]byte    { return nil }
func (m *mockRows) Scan(dest ...any) error {
	if m.idx < 1 || m.idx > len(m.scanFns) {
		return errors.New("scan called out of bounds")
	}
	return m.scanFns[m.idx-1](dest...)
}
func (m *mockRows) Conn() *pgx.Conn { return nil }

// ── helper: scan fn para una submission pendiente ─────────────────────────────

func submissionScanFn(dest ...any) error {
	*(dest[0].(*string)) = testSubID              // id
	*(dest[1].(*string)) = testProblemID          // problem_id
	*(dest[2].(*string)) = testUserID             // user_id
	*(dest[3].(**string)) = nil                    // contest_id
	*(dest[4].(**string)) = nil                    // standing_id
	*(dest[5].(*string)) = "cpp20"                // language
	*(dest[6].(*string)) = "g++"                  // compiler
	*(dest[7].(*string)) = "PENDING"              // status
	*(dest[8].(*string)) = "PRIVATE"              // visibility
	*(dest[9].(*string)) = "gs://bucket/code.cpp" // source_code_path
	*(dest[10].(**string)) = nil                   // file_hash
	*(dest[11].(**int)) = nil                      // file_size
	*(dest[12].(*time.Time)) = testNow            // submitted_at
	*(dest[13].(**time.Time)) = nil                // judged_at
	*(dest[14].(**int)) = nil                      // time_ms
	*(dest[15].(**int)) = nil                      // memory_kb
	*(dest[16].(**string)) = nil                   // compile_log
	return nil
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestSave_Success(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	})

	if err := repo.Save(context.Background(), pendingSubmission()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestSave_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.Save(context.Background(), pendingSubmission())
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── FindByID ──────────────────────────────────────────────────────────────────

func TestFindByID_Success(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: submissionScanFn}
		},
	})

	s, err := repo.FindByID(context.Background(), testSubID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if s.ID() != testSubID {
		t.Errorf("expected id %q, got %q", testSubID, s.ID())
	}
	if s.Status().String() != "PENDING" {
		t.Errorf("expected PENDING status, got %q", s.Status().String())
	}
}

func TestFindByID_NotFound_ReturnsNotFound(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := repo.FindByID(context.Background(), "nonexistent")
	assertAppErrorKind(t, err, apperror.KindNotFound)
	assertAppErrorCode(t, err, domainsubmission.ErrCodeSubmissionNotFound)
}

func TestFindByID_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := repo.FindByID(context.Background(), testSubID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestGetByID_NotFound_DelegatesToFindByID(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
		},
	})

	_, err := repo.GetByID(context.Background(), "nonexistent")
	assertAppErrorKind(t, err, apperror.KindNotFound)
	assertAppErrorCode(t, err, domainsubmission.ErrCodeSubmissionNotFound)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestUpdate_Success(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 1"), nil
		},
	})

	if err := repo.Update(context.Background(), pendingSubmission()); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestUpdate_NotFound_ReturnsNotFound(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.NewCommandTag("UPDATE 0"), nil
		},
	})

	err := repo.Update(context.Background(), pendingSubmission())
	assertAppErrorKind(t, err, apperror.KindNotFound)
	assertAppErrorCode(t, err, domainsubmission.ErrCodeSubmissionNotFound)
}

func TestUpdate_DBError_ReturnsInternal(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.Update(context.Background(), pendingSubmission())
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestList_Success_NoFilters(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int)) = 1
				return nil
			}}
		},
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return &mockRows{scanFns: []func(dest ...any) error{submissionScanFn}}, nil
		},
	})

	filters := domainsubmission.ListFilters{Page: 1, Limit: 10}
	result, total, err := repo.List(context.Background(), filters)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 submission, got %d", len(result))
	}
}

func TestList_EmptyResult_ReturnsNonNilSlice(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int)) = 0
				return nil
			}}
		},
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return &mockRows{}, nil
		},
	})

	result, total, err := repo.List(context.Background(), domainsubmission.ListFilters{Page: 1, Limit: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if result == nil {
		t.Error("expected non-nil slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d items", len(result))
	}
}

func TestList_WithUserIDFilter(t *testing.T) {
	uid := shared.RestoreUserID(testUserID)
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, sql string, args ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int)) = 1
				return nil
			}}
		},
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return &mockRows{scanFns: []func(dest ...any) error{submissionScanFn}}, nil
		},
	})

	filters := domainsubmission.ListFilters{UserID: &uid, Page: 1, Limit: 10}
	result, total, err := repo.List(context.Background(), filters)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if total != 1 || len(result) != 1 {
		t.Errorf("expected 1 result, got total=%d len=%d", total, len(result))
	}
}

func TestList_CountError_ReturnsInternal(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
		},
	})

	_, _, err := repo.List(context.Background(), domainsubmission.ListFilters{Page: 1, Limit: 10})
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestList_QueryError_ReturnsInternal(t *testing.T) {
	repo := NewRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(dest ...any) error {
				*(dest[0].(*int)) = 5
				return nil
			}}
		},
		queryFn: func(_ context.Context, _ string, _ ...interface{}) (pgx.Rows, error) {
			return nil, errors.New("db failure")
		},
	})

	_, _, err := repo.List(context.Background(), domainsubmission.ListFilters{Page: 1, Limit: 10})
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

func assertAppErrorCode(t *testing.T, err error, want string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Code != want {
		t.Errorf("expected code %q, got %q", want, appErr.Code)
	}
}
