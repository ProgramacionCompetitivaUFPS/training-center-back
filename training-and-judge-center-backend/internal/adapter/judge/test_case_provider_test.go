package judge

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func buildZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("buildZip: create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("buildZip: write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("buildZip: close: %v", err)
	}
	return buf.Bytes()
}

func TestParseTestCasesZip_SampleAndSecret(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"data/sample/001.in":  "1 2",
		"data/sample/001.ans": "3",
		"data/secret/002.in":  "4 5",
		"data/secret/002.ans": "9",
	})

	testCases, err := parseTestCasesZip(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(testCases))
	}
	if string(testCases[0].Input) != "1 2" || string(testCases[0].ExpectedOutput) != "3" {
		t.Errorf("test case 0: got input=%q expected=%q", testCases[0].Input, testCases[0].ExpectedOutput)
	}
	if string(testCases[1].Input) != "4 5" || string(testCases[1].ExpectedOutput) != "9" {
		t.Errorf("test case 1: got input=%q expected=%q", testCases[1].Input, testCases[1].ExpectedOutput)
	}
}

func TestParseTestCasesZip_RootPrefix(t *testing.T) {
	// ICPC packages often have a top-level directory matching the problem slug.
	zipData := buildZip(t, map[string]string{
		"problem-abc/data/secret/001.in":  "hello",
		"problem-abc/data/secret/001.ans": "world",
	})

	testCases, err := parseTestCasesZip(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(testCases))
	}
	if string(testCases[0].Input) != "hello" {
		t.Errorf("input: got %q, want %q", testCases[0].Input, "hello")
	}
}

func TestParseTestCasesZip_MissingAnswer_PairOmitted(t *testing.T) {
	// 001 has both .in and .ans; 002 only has .in — it must be omitted.
	zipData := buildZip(t, map[string]string{
		"data/sample/001.in":  "1",
		"data/sample/001.ans": "1",
		"data/sample/002.in":  "2",
	})

	testCases, err := parseTestCasesZip(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 1 {
		t.Fatalf("expected 1 test case (002 omitted), got %d", len(testCases))
	}
}

func TestParseTestCasesZip_NoTestFiles_ReturnsEmpty(t *testing.T) {
	zipData := buildZip(t, map[string]string{
		"problem.yaml": "name: example\n",
	})

	testCases, err := parseTestCasesZip(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 0 {
		t.Errorf("expected empty slice, got %d", len(testCases))
	}
}

func TestParseTestCasesZip_InvalidBytes_ReturnsError(t *testing.T) {
	_, err := parseTestCasesZip([]byte("not a zip"))
	if err == nil {
		t.Fatal("expected error for invalid zip, got nil")
	}
}

func TestParseTestCasesZip_Ordered(t *testing.T) {
	// Files added in reverse order; result must be sorted by base name.
	zipData := buildZip(t, map[string]string{
		"data/secret/003.in":  "c",
		"data/secret/003.ans": "C",
		"data/secret/001.in":  "a",
		"data/secret/001.ans": "A",
		"data/secret/002.in":  "b",
		"data/secret/002.ans": "B",
	})

	testCases, err := parseTestCasesZip(zipData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 3 {
		t.Fatalf("expected 3 test cases, got %d", len(testCases))
	}
	want := []struct{ in, out string }{
		{"a", "A"}, {"b", "B"}, {"c", "C"},
	}
	for i, w := range want {
		if string(testCases[i].Input) != w.in || string(testCases[i].ExpectedOutput) != w.out {
			t.Errorf("case %d: got (%q,%q), want (%q,%q)",
				i, testCases[i].Input, testCases[i].ExpectedOutput, w.in, w.out)
		}
	}
}

func TestGetTestCases_NullKey_ReturnsEmpty(t *testing.T) {
	provider := &TestCaseProvider{
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error {
					*(dest[0].(**string)) = nil
					return nil
				}}
			},
		},
		reader: &mockGCSReader{},
	}

	cases, err := provider.GetTestCases(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cases) != 0 {
		t.Errorf("expected empty, got %d", len(cases))
	}
}

func TestGetTestCases_ProblemNotFound(t *testing.T) {
	provider := &TestCaseProvider{
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error { return pgx.ErrNoRows }}
			},
		},
		reader: &mockGCSReader{},
	}

	_, err := provider.GetTestCases(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestGetTestCases_DBError_ReturnsInternal(t *testing.T) {
	provider := &TestCaseProvider{
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error { return errors.New("db failure") }}
			},
		},
		reader: &mockGCSReader{},
	}

	_, err := provider.GetTestCases(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetTestCases_GCSNotFound_ReturnsNotFound(t *testing.T) {
	key := "problems/abc/tests.zip"
	provider := &TestCaseProvider{
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error {
					k := key
					*(dest[0].(**string)) = &k
					return nil
				}}
			},
		},
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
				return nil, storage.ErrObjectNotExist
			},
		},
	}

	_, err := provider.GetTestCases(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindNotFound)
}

func TestGetTestCases_GCSError_ReturnsInternal(t *testing.T) {
	key := "problems/abc/tests.zip"
	provider := &TestCaseProvider{
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error {
					k := key
					*(dest[0].(**string)) = &k
					return nil
				}}
			},
		},
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, _ string) (io.ReadCloser, error) {
				return nil, errors.New("network error")
			},
		},
	}

	_, err := provider.GetTestCases(context.Background(), testProblemID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestGetTestCases_WithZIP(t *testing.T) {
	key := "problems/abc/tests.zip"
	zipData := buildZip(t, map[string]string{
		"data/secret/001.in":  "5",
		"data/secret/001.ans": "25",
	})

	provider := &TestCaseProvider{
		db: &mockQuerier{
			queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
				return &mockRow{scanFn: func(dest ...any) error {
					k := key
					*(dest[0].(**string)) = &k
					return nil
				}}
			},
		},
		reader: &mockGCSReader{
			readObjectFn: func(_ context.Context, object string) (io.ReadCloser, error) {
				if object != key {
					t.Errorf("readObject: got %q, want %q", object, key)
				}
				return io.NopCloser(bytes.NewReader(zipData)), nil
			},
		},
	}

	cases, err := provider.GetTestCases(context.Background(), testProblemID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cases) != 1 {
		t.Fatalf("expected 1 case, got %d", len(cases))
	}
	if string(cases[0].Input) != "5" || string(cases[0].ExpectedOutput) != "25" {
		t.Errorf("case 0: input=%q expected=%q", cases[0].Input, cases[0].ExpectedOutput)
	}
}
