package judge

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
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

	testCases, err := parseTestCasesZip(zipData, maxTestCaseFileBytes)
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
	if testCases[0].Name != "sample/001" {
		t.Errorf("test case 0 Name: got %q, want %q", testCases[0].Name, "sample/001")
	}
	if testCases[1].Name != "secret/002" {
		t.Errorf("test case 1 Name: got %q, want %q", testCases[1].Name, "secret/002")
	}
}

func TestParseTestCasesZip_RootPrefix(t *testing.T) {
	// ICPC packages often have a top-level directory matching the problem slug.
	zipData := buildZip(t, map[string]string{
		"problem-abc/data/secret/001.in":  "hello",
		"problem-abc/data/secret/001.ans": "world",
	})

	testCases, err := parseTestCasesZip(zipData, maxTestCaseFileBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(testCases))
	}
	if string(testCases[0].Input) != "hello" {
		t.Errorf("input: got %q, want %q", testCases[0].Input, "hello")
	}
	if testCases[0].Name != "secret/001" {
		t.Errorf("Name: got %q, want %q", testCases[0].Name, "secret/001")
	}
}

func TestParseTestCasesZip_MissingAnswer_PairOmitted(t *testing.T) {
	// 001 has both .in and .ans; 002 only has .in — it must be omitted.
	zipData := buildZip(t, map[string]string{
		"data/sample/001.in":  "1",
		"data/sample/001.ans": "1",
		"data/sample/002.in":  "2",
	})

	testCases, err := parseTestCasesZip(zipData, maxTestCaseFileBytes)
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

	testCases, err := parseTestCasesZip(zipData, maxTestCaseFileBytes)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(testCases) != 0 {
		t.Errorf("expected empty slice, got %d", len(testCases))
	}
}

func TestParseTestCasesZip_InvalidBytes_ReturnsError(t *testing.T) {
	_, err := parseTestCasesZip([]byte("not a zip"), maxTestCaseFileBytes)
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

	testCases, err := parseTestCasesZip(zipData, maxTestCaseFileBytes)
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
	// The problem stores the upload prefix; the ZIP lives inside it.
	const key = "problems/abc/testcases/2b0f"
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
	// The problem stores the upload prefix; the ZIP lives inside it.
	const key = "problems/abc/testcases/2b0f"
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
	// The problem stores the upload prefix; the ZIP lives inside it.
	const key = "problems/abc/testcases/2b0f"
	const wantObject = "problems/abc/testcases/2b0f/testcases.zip"
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
				if object != wantObject {
					t.Errorf("readObject: got %q, want %q", object, wantObject)
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

// A file past the cap must fail. io.LimitReader on its own returns no error, so
// the answer would come back cut and every submission would be judged against it.
func TestParseTestCasesZip_AFileOverTheLimitFails(t *testing.T) {
	const limit = 10

	zipData := buildZip(t, map[string]string{
		"data/secret/001.in":  "1 2",
		"data/secret/001.ans": strings.Repeat("x", limit+1),
	})

	testCases, err := parseTestCasesZip(zipData, limit)
	if err == nil {
		t.Fatalf("expected the oversized file to fail, got %d test cases", len(testCases))
	}
	if !strings.Contains(err.Error(), "001.ans") {
		t.Errorf("expected the error to name the file, got: %v", err)
	}
}

// The boundary is load-bearing: a file of exactly the cap is legal, and it has
// to arrive whole — reading maxFileBytes+1 must not leak into what is kept.
func TestParseTestCasesZip_AFileExactlyAtTheLimitIsReadWhole(t *testing.T) {
	const limit = 10
	answer := strings.Repeat("x", limit)

	zipData := buildZip(t, map[string]string{
		"data/secret/001.in":  "1 2",
		"data/secret/001.ans": answer,
	})

	testCases, err := parseTestCasesZip(zipData, limit)
	if err != nil {
		t.Fatalf("expected a file exactly at the cap to be accepted, got: %v", err)
	}
	if len(testCases) != 1 {
		t.Fatalf("expected 1 test case, got %d", len(testCases))
	}
	if string(testCases[0].ExpectedOutput) != answer {
		t.Errorf("expected the answer to arrive whole (%d bytes), got %d",
			len(answer), len(testCases[0].ExpectedOutput))
	}
}
