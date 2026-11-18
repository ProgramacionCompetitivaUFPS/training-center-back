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

func writeZipFile(t *testing.T, data []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "cases.zip")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
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

func openSet(t *testing.T, files map[string]string) (*zipTestCaseSet, *int) {
	t.Helper()
	releases := 0
	set, err := openZipTestCaseSet(writeZipFile(t, zipBytes(t, files, false)), func() { releases++ })
	if err != nil {
		t.Fatalf("openZipTestCaseSet: %v", err)
	}
	t.Cleanup(func() { _ = set.Close() })
	return set, &releases
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

func TestZipTestCaseSet_SampleAndSecret(t *testing.T) {
	set, _ := openSet(t, map[string]string{
		"data/sample/001.in":  "1 2",
		"data/sample/001.ans": "3",
		"data/secret/002.in":  "4 5",
		"data/secret/002.ans": "9",
	})

	cases := set.Cases()
	if len(cases) != 2 {
		t.Fatalf("expected 2 test cases, got %d", len(cases))
	}
	if cases[0].Name != "sample/001" || cases[1].Name != "secret/002" {
		t.Errorf("names: got %q and %q", cases[0].Name, cases[1].Name)
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

// ICPC packages often have a top-level directory matching the problem slug.
func TestZipTestCaseSet_RootPrefix(t *testing.T) {
	set, _ := openSet(t, map[string]string{
		"problem-abc/data/secret/001.in":  "hello",
		"problem-abc/data/secret/001.ans": "world",
	})

	if cases := set.Cases(); len(cases) != 1 || cases[0].Name != "secret/001" {
		t.Fatalf("expected the one case secret/001, got %+v", cases)
	}
	in, err := set.OpenInput("secret/001")
	if got := readAll(t, in, err); got != "hello" {
		t.Errorf("input: got %q", got)
	}
}

func TestZipTestCaseSet_MissingAnswer_PairOmitted(t *testing.T) {
	set, _ := openSet(t, map[string]string{
		"data/sample/001.in":  "1",
		"data/sample/001.ans": "1",
		"data/sample/002.in":  "2",
	})

	if cases := set.Cases(); len(cases) != 1 {
		t.Fatalf("expected 1 test case (002 omitted), got %d", len(cases))
	}
}

func TestZipTestCaseSet_NoTestFiles_IsEmpty(t *testing.T) {
	set, _ := openSet(t, map[string]string{"problem.yaml": "name: example\n"})

	if cases := set.Cases(); len(cases) != 0 {
		t.Errorf("expected no cases, got %d", len(cases))
	}
}

func TestZipTestCaseSet_Ordered(t *testing.T) {
	set, _ := openSet(t, map[string]string{
		"data/secret/003.in": "c", "data/secret/003.ans": "C",
		"data/secret/001.in": "a", "data/secret/001.ans": "A",
		"data/secret/002.in": "b", "data/secret/002.ans": "B",
	})

	for i, want := range []string{"secret/001", "secret/002", "secret/003"} {
		if got := set.Cases()[i].Name; got != want {
			t.Errorf("case %d: got %q, want %q", i, got, want)
		}
	}
}

func TestZipTestCaseSet_UnknownCase_IsAnError(t *testing.T) {
	set, _ := openSet(t, map[string]string{"data/secret/001.in": "a", "data/secret/001.ans": "A"})

	if _, err := set.OpenInput("secret/999"); err == nil {
		t.Error("expected an error for a case the set does not have")
	}
}

func TestZipTestCaseSet_NotAnArchive_IsInvalidAndReleases(t *testing.T) {
	path := writeZipFile(t, []byte("not a zip"))
	released := false

	_, err := openZipTestCaseSet(path, func() { released = true })

	if !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
	if !released {
		t.Error("a set that cannot be opened must still release what it held")
	}
}

func TestZipTestCaseSet_CloseReleasesOnce(t *testing.T) {
	set, releases := openSet(t, map[string]string{"data/secret/001.in": "a", "data/secret/001.ans": "A"})

	_ = set.Close()
	_ = set.Close()

	if *releases != 1 {
		t.Errorf("released %d times, want exactly 1", *releases)
	}
}

func verify(t *testing.T, data []byte, limits TestDataLimits) error {
	t.Helper()
	return verifyTestCaseZip(context.Background(), writeZipFile(t, data), limits)
}

func TestVerifyTestCaseZip_ValidArchivePasses(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"data/sample/1.in": "1", "data/sample/1.ans": "1",
		"data/secret/2.in": "2", "data/secret/2.ans": "2",
	}, false)

	if err := verify(t, data, testLimits()); err != nil {
		t.Fatalf("expected the archive to pass, got %v", err)
	}
}

func TestVerifyTestCaseZip_NotAnArchive(t *testing.T) {
	if err := verify(t, []byte("not a zip"), testLimits()); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

func TestVerifyTestCaseZip_PerFileLimits(t *testing.T) {
	limits := testLimits()
	limits.MaxInputBytes = 10
	limits.MaxAnswerBytes = 10

	t.Run("input over its limit", func(t *testing.T) {
		data := zipBytes(t, map[string]string{"data/secret/1.in": strings.Repeat("x", 11), "data/secret/1.ans": "1"}, false)
		if err := verify(t, data, limits); !errors.Is(err, errInvalidTestData) {
			t.Fatalf("expected errInvalidTestData, got %v", err)
		}
	})
	t.Run("answer over its limit", func(t *testing.T) {
		data := zipBytes(t, map[string]string{"data/secret/1.in": "1", "data/secret/1.ans": strings.Repeat("x", 11)}, false)
		if err := verify(t, data, limits); !errors.Is(err, errInvalidTestData) {
			t.Fatalf("expected errInvalidTestData, got %v", err)
		}
	})
	// The boundary is load-bearing: a file of exactly the cap is legal.
	t.Run("files exactly at their limits", func(t *testing.T) {
		data := zipBytes(t, map[string]string{"data/secret/1.in": strings.Repeat("x", 10), "data/secret/1.ans": strings.Repeat("y", 10)}, false)
		if err := verify(t, data, limits); err != nil {
			t.Fatalf("expected files at the cap to pass, got %v", err)
		}
	})
}

func TestVerifyTestCaseZip_TotalExtractedLimit(t *testing.T) {
	limits := testLimits()
	limits.MaxExtractedBytes = 100

	data := zipBytes(t, map[string]string{
		"data/secret/1.in": strings.Repeat("a", 60), "data/secret/1.ans": "1",
		"data/secret/2.in": strings.Repeat("b", 60), "data/secret/2.ans": "2",
	}, false)

	if err := verify(t, data, limits); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData for 122 bytes against a 100 limit, got %v", err)
	}
}

// Everything a package carries counts against the total, not only the test data.
func TestVerifyTestCaseZip_OtherFilesCountTowardsTheTotal(t *testing.T) {
	limits := testLimits()
	limits.MaxExtractedBytes = 100

	data := zipBytes(t, map[string]string{
		"data/secret/1.in": "1", "data/secret/1.ans": "1",
		"solutions/accepted/sol.cpp": strings.Repeat("c", 200),
	}, false)

	if err := verify(t, data, limits); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

func TestVerifyTestCaseZip_TooManyFiles(t *testing.T) {
	limits := testLimits()
	limits.MaxFiles = 3

	data := zipBytes(t, map[string]string{
		"data/secret/1.in": "1", "data/secret/1.ans": "1",
		"data/secret/2.in": "2", "data/secret/2.ans": "2",
	}, false)

	if err := verify(t, data, limits); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

// The secret group is never read by anything but a judging, so only the full pass can see this.
func TestVerifyTestCaseZip_CorruptedEntry(t *testing.T) {
	marker := "SECRETMARKER-123"
	data := zipBytes(t, map[string]string{
		"data/sample/1.in": "1", "data/sample/1.ans": "1",
		"data/secret/2.in": marker, "data/secret/2.ans": "2",
	}, true)

	corrupted := bytes.Replace(data, []byte(marker), []byte("SECRETMARKER-124"), 1)

	if err := verify(t, corrupted, testLimits()); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData for a corrupted entry, got %v", err)
	}
}

// A header that understates the size must not let the entry grow past it: the standard
// library stops the read, which is what keeps a lying header from becoming a bomb.
func TestVerifyTestCaseZip_DeclaredSizeSmallerThanReal(t *testing.T) {
	data := zipBytes(t, map[string]string{
		"data/secret/1.in": strings.Repeat("x", 1000), "data/secret/1.ans": "1",
	}, true)

	// 1000 little-endian is E8 03 00 00; both the local and the central header carry it.
	lie := bytes.ReplaceAll(data, []byte{0xE8, 0x03, 0x00, 0x00}, []byte{0x0A, 0x00, 0x00, 0x00})
	if bytes.Equal(lie, data) {
		t.Fatal("test setup: the size field was not found")
	}

	if err := verify(t, lie, testLimits()); !errors.Is(err, errInvalidTestData) {
		t.Fatalf("expected errInvalidTestData, got %v", err)
	}
}

func TestVerifyTestCaseZip_CancelledContext(t *testing.T) {
	data := zipBytes(t, map[string]string{"data/secret/1.in": "1", "data/secret/1.ans": "1"}, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := verifyTestCaseZip(ctx, writeZipFile(t, data), testLimits())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}
