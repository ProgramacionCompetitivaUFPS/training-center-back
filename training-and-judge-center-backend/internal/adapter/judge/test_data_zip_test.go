package judge

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// zipBytes builds an archive; stored entries keep their bytes as written, which lets a
// test change them in place.
func zipBytes(t *testing.T, files map[string]string, store bool) []byte {
	t.Helper()
	method := zip.Deflate
	if store {
		method = zip.Store
	}
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.CreateHeader(&zip.FileHeader{Name: name, Method: method})
		if err != nil {
			t.Fatalf("zipBytes: create %s: %v", name, err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatalf("zipBytes: write %s: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("zipBytes: close: %v", err)
	}
	return buf.Bytes()
}

func testLimits() TestDataLimits {
	return TestDataLimits{
		MaxZipBytes:       1 << 20,
		MaxExtractedBytes: 10 << 20,
		MaxFiles:          100,
		MaxInputBytes:     1 << 20,
		MaxAnswerBytes:    1 << 10,
	}
}

func planOf(t *testing.T, data []byte, limits TestDataLimits) (extractionPlan, error) {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("not a valid archive: %v", err)
	}
	return planExtraction(zr, limits)
}

// extract plans and extracts an archive into a fresh directory.
func extract(t *testing.T, data []byte, limits TestDataLimits) ([]cachedCase, string, error) {
	t.Helper()
	plan, err := planOf(t, data, limits)
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Join(t.TempDir(), "problem")
	cases, err := extractTestCases(context.Background(), plan, dir, dir)
	return cases, dir, err
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func readAll(t *testing.T, rc io.ReadCloser, err error) string {
	t.Helper()
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	return string(data)
}

func caseNames(plan extractionPlan) []string {
	names := make([]string, len(plan.cases))
	for i, c := range plan.cases {
		names[i] = c.name
	}
	return names
}

func TestPlanExtraction_SampleAndSecretInOrder(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"data/sample/001.in": "1 2", "data/sample/001.ans": "3",
		"data/secret/003.in": "c", "data/secret/003.ans": "C",
		"data/secret/002.in": "4 5", "data/secret/002.ans": "9",
	}, false)

	plan, err := planOf(t, data, testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := strings.Join(caseNames(plan), ",")
	if got != "sample/001,secret/002,secret/003" {
		t.Errorf("cases: got %s", got)
	}
	if plan.bytes != int64(3+1+3+1+1+1) {
		t.Errorf("bytes: got %d, want the declared sizes of the six case files", plan.bytes)
	}
}

// ICPC packages often have a top-level directory matching the problem slug.
func TestPlanExtraction_RootPrefix(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"problem-abc/data/secret/001.in": "hello", "problem-abc/data/secret/001.ans": "world",
	}, false)

	plan, err := planOf(t, data, testLimits())
	if err != nil || len(plan.cases) != 1 || plan.cases[0].name != "secret/001" {
		t.Fatalf("expected the one case secret/001, got %v, %v", caseNames(plan), err)
	}
}

func TestPlanExtraction_MissingAnswer_PairOmittedAndNotCounted(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"data/sample/001.in": "1", "data/sample/001.ans": "1",
		"data/sample/002.in": strings.Repeat("x", 500),
	}, false)

	plan, err := planOf(t, data, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.cases) != 1 {
		t.Fatalf("expected 1 case (002 omitted), got %d", len(plan.cases))
	}
	if plan.bytes != 2 {
		t.Errorf("bytes = %d: a file that will not be extracted must not reserve disk", plan.bytes)
	}
}

func TestPlanExtraction_NoTestFiles_IsEmpty(t *testing.T) {
	plan, err := planOf(t, zipBytes(t, map[string]string{"problem.yaml": "name: example\n"}, false), testLimits())
	if err != nil || len(plan.cases) != 0 {
		t.Fatalf("expected an empty plan, got %v, %v", caseNames(plan), err)
	}
}

func TestPlanExtraction_PerFileLimits(t *testing.T) {
	limits := testLimits()
	limits.MaxInputBytes = 10
	limits.MaxAnswerBytes = 10

	t.Run("input over its limit", func(t *testing.T) {
		data := zipBytes(t, map[string]string{"data/secret/1.in": strings.Repeat("x", 11), "data/secret/1.ans": "1"}, false)
		if _, err := planOf(t, data, limits); !errors.Is(err, errInvalidTestData) {
			t.Fatalf("expected errInvalidTestData, got %v", err)
		}
	})
	t.Run("answer over its limit", func(t *testing.T) {
		data := zipBytes(t, map[string]string{"data/secret/1.in": "1", "data/secret/1.ans": strings.Repeat("x", 11)}, false)
		if _, err := planOf(t, data, limits); !errors.Is(err, errInvalidTestData) {
			t.Fatalf("expected errInvalidTestData, got %v", err)
		}
	})
	// The boundary is load-bearing: a file of exactly the cap is legal.
	t.Run("files exactly at their limits", func(t *testing.T) {
		data := zipBytes(t, map[string]string{"data/secret/1.in": strings.Repeat("x", 10), "data/secret/1.ans": strings.Repeat("y", 10)}, false)
		if _, err := planOf(t, data, limits); err != nil {
			t.Fatalf("expected files at the cap to pass, got %v", err)
		}
	})
}

func TestPlanExtraction_TotalExtractedLimit(t *testing.T) {
	limits := testLimits()
	limits.MaxExtractedBytes = 100

	data := zipBytes(t, map[string]string{
		"data/secret/1.in": strings.Repeat("a", 60), "data/secret/1.ans": "1",
		"data/secret/2.in": strings.Repeat("b", 60), "data/secret/2.ans": "2",
	}, false)

	if _, err := planOf(t, data, limits); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData for 122 bytes against a 100 limit, got %v", err)
	}
}

// Everything a package carries counts against the total, not only the test data.
func TestPlanExtraction_OtherFilesCountTowardsTheTotal(t *testing.T) {
	limits := testLimits()
	limits.MaxExtractedBytes = 100

	data := zipBytes(t, map[string]string{
		"data/secret/1.in": "1", "data/secret/1.ans": "1",
		"solutions/accepted/sol.cpp": strings.Repeat("c", 200),
	}, false)

	if _, err := planOf(t, data, limits); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

func TestPlanExtraction_TooManyFiles(t *testing.T) {
	limits := testLimits()
	limits.MaxFiles = 3

	data := zipBytes(t, map[string]string{
		"data/secret/1.in": "1", "data/secret/1.ans": "1",
		"data/secret/2.in": "2", "data/secret/2.ans": "2",
	}, false)

	if _, err := planOf(t, data, limits); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

func TestExtractTestCases_WritesEachCaseToAFileNamedByPosition(t *testing.T) {
	cases, dir, err := extract(t, zipBytes(t, map[string]string{
		"data/sample/001.in": "1 2", "data/sample/001.ans": "3",
		"data/secret/002.in": "4 5", "data/secret/002.ans": "9",
	}, false), testLimits())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cases) != 2 {
		t.Fatalf("expected 2 cases, got %d", len(cases))
	}
	if readFile(t, cases[0].inputPath) != "1 2" || readFile(t, cases[0].answerPath) != "3" {
		t.Errorf("case 0 content is wrong")
	}
	if readFile(t, cases[1].inputPath) != "4 5" || readFile(t, cases[1].answerPath) != "9" {
		t.Errorf("case 1 content is wrong")
	}
	if cases[0].inputSize != 3 || cases[0].answerSize != 1 {
		t.Errorf("sizes: got %d and %d", cases[0].inputSize, cases[0].answerSize)
	}
	// Names come from position, never from the entry: nothing in an archive picks where a byte lands.
	if filepath.Dir(cases[0].inputPath) != dir || filepath.Base(cases[0].inputPath) != "00000.in" {
		t.Errorf("input path = %s, want %s/00000.in", cases[0].inputPath, dir)
	}
}

// An entry's name must not be able to place a file: even one that tries to climb out of the directory
// is just a case, written under its position.
func TestExtractTestCases_ANastyEntryNameCannotChooseWhereItLands(t *testing.T) {
	root := t.TempDir()
	data := zipBytes(t, map[string]string{
		"data/secret/..\\..\\evil.in": "x", "data/secret/..\\..\\evil.ans": "y",
	}, false)
	plan, err := planOf(t, data, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "inside")

	cases, err := extractTestCases(context.Background(), plan, dir, dir)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, c := range cases {
		if filepath.Dir(c.inputPath) != dir || filepath.Dir(c.answerPath) != dir {
			t.Errorf("a file landed outside %s: %s, %s", dir, c.inputPath, c.answerPath)
		}
	}
	items, _ := os.ReadDir(root)
	if len(items) != 1 {
		t.Errorf("something was written next to the directory: %d items in %s", len(items), root)
	}
}

// The secret group is never read by anything before a judging, so extracting is what finds this.
func TestExtractTestCases_CorruptedEntry(t *testing.T) {
	marker := "SECRETMARKER-123"
	data := zipBytes(t, map[string]string{
		"data/sample/1.in": "1", "data/sample/1.ans": "1",
		"data/secret/2.in": marker, "data/secret/2.ans": "2",
	}, true)
	corrupted := bytes.Replace(data, []byte(marker), []byte("SECRETMARKER-124"), 1)

	if _, _, err := extract(t, corrupted, testLimits()); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData for a corrupted entry, got %v", err)
	}
}

// A header that understates the size must not let the entry grow past it: the standard library stops
// the read, which is what keeps a lying header from becoming a bomb.
func TestExtractTestCases_DeclaredSizeSmallerThanReal(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"data/secret/1.in": strings.Repeat("x", 1000), "data/secret/1.ans": "1",
	}, true)

	// 1000 little-endian is E8 03 00 00; both the local and the central header carry it.
	lie := bytes.ReplaceAll(data, []byte{0xE8, 0x03, 0x00, 0x00}, []byte{0x0A, 0x00, 0x00, 0x00})
	if bytes.Equal(lie, data) {
		t.Fatal("test setup: the size field was not found")
	}

	if _, _, err := extract(t, lie, testLimits()); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

// Only the cases are read, so damage to something a judging never uses does not refuse the problem.
func TestExtractTestCases_DamageOutsideTheCasesIsNotRead(t *testing.T) {
	marker := "SOLUTIONMARKER-1"
	data := zipBytes(t, map[string]string{
		"data/secret/1.in": "1", "data/secret/1.ans": "1",
		"solutions/accepted/sol.cpp": marker,
	}, true)
	damaged := bytes.Replace(data, []byte(marker), []byte("SOLUTIONMARKER-2"), 1)

	if _, _, err := extract(t, damaged, testLimits()); err != nil {
		t.Fatalf("expected the damaged solution to be ignored, got %v", err)
	}
}

func TestExtractTestCases_CancelledContext(t *testing.T) {
	plan, err := planOf(t, zipBytes(t, map[string]string{"data/secret/1.in": "1", "data/secret/1.ans": "1"}, false), testLimits())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := filepath.Join(t.TempDir(), "p")
	_, err = extractTestCases(ctx, plan, p, p)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func newSetOf(t *testing.T, files map[string]string) (*fileTestCaseSet, *int) {
	t.Helper()
	cases, _, err := extract(t, zipBytes(t, files, false), testLimits())
	if err != nil {
		t.Fatalf("extract: %v", err)
	}
	releases := 0
	set := newFileTestCaseSet(cases, func() { releases++ })
	t.Cleanup(func() { _ = set.Close() })
	return set, &releases
}

func TestFileTestCaseSet_ServesTheCasesOneByOne(t *testing.T) {
	set, _ := newSetOf(t, map[string]string{
		"data/sample/001.in": "1 2", "data/sample/001.ans": "3",
		"data/secret/002.in": "4 5", "data/secret/002.ans": "9",
	})

	cases := set.Cases()
	if len(cases) != 2 || cases[0].Name != "sample/001" || cases[1].Name != "secret/002" {
		t.Fatalf("cases: got %+v", cases)
	}
	if cases[0].InputSize != 3 || cases[0].AnswerSize != 1 {
		t.Errorf("sizes of case 0: got %d and %d", cases[0].InputSize, cases[0].AnswerSize)
	}
	in, err := set.OpenInput("secret/002")
	if got := readAll(t, in, err); got != "4 5" {
		t.Errorf("input: got %q", got)
	}
	ans, err := set.OpenAnswer("secret/002")
	if got := readAll(t, ans, err); got != "9" {
		t.Errorf("answer: got %q", got)
	}
}

func TestFileTestCaseSet_UnknownCase_IsAnError(t *testing.T) {
	set, _ := newSetOf(t, map[string]string{"data/secret/001.in": "a", "data/secret/001.ans": "A"})

	if _, err := set.OpenInput("secret/999"); err == nil {
		t.Error("expected an error for a case the set does not have")
	}
	if _, err := set.OpenAnswer("secret/999"); err == nil {
		t.Error("expected an error for a case the set does not have")
	}
}

func TestFileTestCaseSet_CloseReleasesOnce(t *testing.T) {
	set, releases := newSetOf(t, map[string]string{"data/secret/001.in": "a", "data/secret/001.ans": "A"})

	_ = set.Close()
	_ = set.Close()

	if *releases != 1 {
		t.Errorf("released %d times, want exactly 1", *releases)
	}
}
