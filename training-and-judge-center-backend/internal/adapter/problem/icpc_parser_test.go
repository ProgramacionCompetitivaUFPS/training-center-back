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

// testParserConfig uses deliberately small caps: these tests pin behaviour, and
// asserting against the shipped 64/8 would compare a mutation to itself.
func testParserConfig() problem.ICPCParserConfig {
	return problem.ICPCParserConfig{
		MaxUncompressedSizeMB: 200,
		MaxTestCaseInputMB:    2,
		MaxTestCaseAnswerMB:   1,
		MaxMetadataFileSizeMB: 2,
		MaxFiles:              10000,
		MaxSampleFiles:        10,
	}
}

func createZipBuffer(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatalf("createZipBuffer: failed to create entry %q: %v", name, err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatalf("createZipBuffer: failed to write entry %q: %v", name, err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatalf("createZipBuffer: failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

func TestICPCParser_ValidArchive(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": []byte("3"),
		"data/secret/2.in":  []byte("10 20"),
		"data/secret/2.ans": []byte("30"),
	})

	extracted, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err != nil {
		if appErr, ok := err.(*apperror.AppError); ok && len(appErr.Details) > 0 {
			t.Fatalf("expected no error, got detail: %s", appErr.Details[0].Message)
		}
		t.Fatalf("expected no error, got: %v", err)
	}

	if len(extracted) != 2 {
		t.Errorf("expected 2 files extracted (only samples), got %d", len(extracted))
	}
}

func TestICPCParser_InvalidExtension(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.txt": []byte("3"),
	})

	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err == nil {
		t.Fatal("expected error for invalid extension, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok || len(appErr.Details) == 0 {
		t.Fatalf("expected apperror.AppError with Details, got: %v", err)
	}

	if !strings.Contains(appErr.Details[0].Message, "Invalid file extension") {
		t.Errorf("expected extension error in Details, got: %v", appErr.Details[0].Message)
	}
}

func TestICPCParser_PathTraversal(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":            []byte("1 2"),
		"data/sample/../../etc/hosts": []byte("malicious"),
	})

	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err == nil {
		t.Fatal("expected error for malicious path, got nil")
	}
}

func TestICPCParser_MissingContentDir(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig())

	zipData := createZipBuffer(t, map[string][]byte{
		"some_other_dir/1.in": []byte("1 2"),
	})

	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err == nil {
		t.Fatal("expected error for files outside testcases dirs")
	}
}

func TestICPCParser_TooManySamples(t *testing.T) {
	cfg := testParserConfig()
	cfg.MaxSampleFiles = 1
	p := problem.NewICPCParser(cfg)

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in": []byte("1 2"),
		"data/sample/2.in": []byte("3"),
	})

	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err == nil {
		t.Fatal("expected error for too many samples, got nil")
	}
}

// rejectionMessage fails unless the parse was rejected, and returns the reason.
func rejectionMessage(t *testing.T, err error) string {
	t.Helper()
	if err == nil {
		t.Fatal("expected the package to be rejected, got nil")
	}
	appErr, ok := err.(*apperror.AppError)
	if !ok || len(appErr.Details) == 0 {
		t.Fatalf("expected apperror.AppError with Details, got: %v", err)
	}
	return appErr.Details[0].Message
}

func mb(n float64) []byte { return bytes.Repeat([]byte("x"), int(n*1024*1024)) }

func TestICPCParser_InputOverItsOwnCapIsRejected(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig()) // input cap 2 MB

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  mb(3),
		"data/sample/1.ans": []byte("3"),
	})

	msg := rejectionMessage(t, mustFailParse(t, p, zipData))
	// HasPrefix and not Contains: the file path travels inside the message.
	if !strings.HasPrefix(msg, "Test case input") {
		t.Errorf("expected the input cap to be the one that fired, got: %s", msg)
	}
}

func TestICPCParser_AnswerOverItsOwnCapIsRejected(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig()) // answer cap 1 MB

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": mb(1.5),
	})

	msg := rejectionMessage(t, mustFailParse(t, p, zipData))
	if !strings.HasPrefix(msg, "Test case answer") {
		t.Errorf("expected the answer cap to be the one that fired, got: %s", msg)
	}
}

// The two caps are not interchangeable: 1.5 MB sits under the input cap and
// over the answer cap, so every wrong limit (input, metadata, total) lets it
// through and only the right one rejects it.
func TestICPCParser_TheTwoTestCaseCapsAreNotInterchangeable(t *testing.T) {
	t.Run("an answer of that size is rejected", func(t *testing.T) {
		p := problem.NewICPCParser(testParserConfig())
		zipData := createZipBuffer(t, map[string][]byte{
			"data/sample/1.in":  []byte("1 2"),
			"data/sample/1.ans": mb(1.5),
		})
		msg := rejectionMessage(t, mustFailParse(t, p, zipData))
		if !strings.HasPrefix(msg, "Test case answer") {
			t.Errorf("expected the answer cap to be the one that fired, got: %s", msg)
		}
	})

	t.Run("an input of that size is accepted", func(t *testing.T) {
		p := problem.NewICPCParser(testParserConfig())
		zipData := createZipBuffer(t, map[string][]byte{
			"data/sample/1.in":  mb(1.5),
			"data/sample/1.ans": []byte("3"),
		})
		if _, err := p.ParseTestCasesZip(context.Background(), zipData); err != nil {
			t.Fatalf("expected the input to fit under its own cap, got: %v", err)
		}
	})
}

func TestICPCParser_AuxiliaryFilesKeepTheirOwnCap(t *testing.T) {
	p := problem.NewICPCParser(testParserConfig()) // metadata cap 2 MB

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": []byte("3"),
		"README.md":         mb(3),
	})

	msg := rejectionMessage(t, mustFailParse(t, p, zipData))
	if !strings.HasPrefix(msg, "Auxiliary file") {
		t.Errorf("expected the metadata cap to be the one that fired, got: %s", msg)
	}
}

// A cap of N MB means N MB is legal. cmd/compare's maxTokenBytes is sized one
// notch above this, so a token filling a whole answer file still scans.
func TestICPCParser_AFileExactlyAtItsCapIsAccepted(t *testing.T) {
	oneMB := 1 * 1024 * 1024 // testParserConfig caps an answer at 1 MB

	t.Run("exactly at the cap", func(t *testing.T) {
		p := problem.NewICPCParser(testParserConfig())
		zipData := createZipBuffer(t, map[string][]byte{
			"data/sample/1.in":  []byte("1 2"),
			"data/sample/1.ans": bytes.Repeat([]byte("x"), oneMB),
		})
		if _, err := p.ParseTestCasesZip(context.Background(), zipData); err != nil {
			t.Fatalf("expected a file exactly at its cap to be accepted, got: %v", err)
		}
	})

	t.Run("one byte over the cap", func(t *testing.T) {
		p := problem.NewICPCParser(testParserConfig())
		zipData := createZipBuffer(t, map[string][]byte{
			"data/sample/1.in":  []byte("1 2"),
			"data/sample/1.ans": bytes.Repeat([]byte("x"), oneMB+1),
		})
		msg := rejectionMessage(t, mustFailParse(t, p, zipData))
		if !strings.HasPrefix(msg, "Test case answer") {
			t.Errorf("expected the answer cap to be the one that fired, got: %s", msg)
		}
	})
}

func mustFailParse(t *testing.T, p *problem.ICPCParser, zipData []byte) error {
	t.Helper()
	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	return err
}
