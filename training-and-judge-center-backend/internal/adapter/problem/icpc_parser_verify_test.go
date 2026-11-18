package problem_test

import (
	"archive/zip"
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/adapter/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// storedZip keeps entries uncompressed so a test can corrupt their bytes in place.
func storedZip(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, err := w.CreateHeader(&zip.FileHeader{Name: name, Method: zip.Store})
		if err != nil {
			t.Fatalf("create %q: %v", name, err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatalf("write %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	return buf.Bytes()
}

func parseTestCases(p *problem.ICPCParser, data []byte) error {
	_, err := p.ParseTestCasesZip(context.Background(), bytes.NewReader(data), int64(len(data)))
	return err
}

func TestICPCParser_CorruptedEntryData_IsRefusedAtUpload(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())
	marker := []byte("SECRETMARKER-123")
	data := storedZip(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": []byte("3"),
		"data/secret/2.in":  marker,
		"data/secret/2.ans": []byte("30"),
	})

	if err := parseTestCases(p, data); err != nil {
		t.Fatalf("the intact archive must pass, got %v", err)
	}

	// The secret group is never extracted by the parser, so only the full pass can see this.
	corrupted := bytes.Replace(data, marker, []byte("SECRETMARKER-124"), 1)
	err := parseTestCases(p, corrupted)
	appErr, ok := err.(*apperror.AppError)
	if !ok || appErr.Kind != apperror.KindValidation {
		t.Fatalf("expected a validation error for a corrupted entry, got %v", err)
	}
}

func TestICPCParser_TruncatedEntryData_IsRefused(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())
	data := storedZip(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": []byte("3"),
		"data/secret/2.in":  bytes.Repeat([]byte("a"), 4096),
		"data/secret/2.ans": []byte("30"),
	})

	// Cutting the directory off the end leaves no valid archive at all.
	if err := parseTestCases(p, data[:len(data)/2]); err == nil {
		t.Fatal("expected an error for a truncated archive")
	}
}

func TestICPCParser_DeclaredSizeSmallerThanReal_IsRefused(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())
	content := bytes.Repeat([]byte("x"), 1000)
	data := storedZip(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": []byte("3"),
		"data/secret/2.in":  content,
		"data/secret/2.ans": []byte("30"),
	})

	// Rewrites the declared uncompressed size (1000) in both headers to a smaller one.
	lie := bytes.ReplaceAll(data, []byte{0xE8, 0x03, 0x00, 0x00}, []byte{0x0A, 0x00, 0x00, 0x00})
	if bytes.Equal(lie, data) {
		t.Fatal("test setup: the size field was not found")
	}
	if err := parseTestCases(p, lie); err == nil {
		t.Fatal("expected an error when the data outgrows its declared size")
	}
}

func TestICPCParser_ExtractedSizeTooLarge_MessageSaysHowFarOverTheLimitItIs(t *testing.T) {
	cfg := testParserConfig()
	cfg.MaxUncompressedSizeMB = 1
	p := problem.NewICPCParser(cfg)
	data := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  mb(0.6),
		"data/sample/1.ans": mb(0.6),
	})

	err := parseTestCases(p, data)

	appErr, ok := err.(*apperror.AppError)
	if !ok || len(appErr.Details) == 0 {
		t.Fatalf("expected a validation error with details, got %v", err)
	}
	msg := appErr.Details[0].Message
	if !strings.Contains(msg, "1.2 MiB") || !strings.Contains(msg, "1 MiB") {
		t.Errorf("the message must give the real size and the limit, got %q", msg)
	}
}

// The package is still read to the end of its directory when it is over the limit, so a ZIP
// that is both too big and has a bad path still reports the path, which is the more serious one.
func TestICPCParser_ExtractedSizeTooLarge_StillRefusesAMaliciousPath(t *testing.T) {
	cfg := testParserConfig()
	cfg.MaxUncompressedSizeMB = 1
	p := problem.NewICPCParser(cfg)
	data := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  mb(2),
		"../escape.txt":     []byte("x"),
		"data/sample/1.ans": []byte("1"),
	})

	err := parseTestCases(p, data)

	appErr, ok := err.(*apperror.AppError)
	if !ok || len(appErr.Details) == 0 || !strings.Contains(appErr.Details[0].Message, "malicious") {
		t.Fatalf("expected the malicious path to be reported, got %v", err)
	}
}
