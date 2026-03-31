package parser_test

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/infrastructure/parser"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func createZipBuffer(files map[string][]byte) []byte {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	for name, content := range files {
		f, _ := w.Create(name)
		f.Write(content)
	}

	w.Close()
	return buf.Bytes()
}

func TestICPCParser_ValidArchive(t *testing.T) {
	p := parser.NewICPCParser(200, 2, 10000, 10, nil)

	zipData := createZipBuffer(map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.ans": []byte("3"),
		"data/secret/2.in":  []byte("10 20"),
		"data/secret/2.ans": []byte("30"),
	})

	extracted, err := p.ParseTestCasesZip(zipData)
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
	p := parser.NewICPCParser(200, 2, 10000, 10, nil)

	zipData := createZipBuffer(map[string][]byte{
		"data/sample/1.in":  []byte("1 2"),
		"data/sample/1.txt": []byte("3"),
	})

	_, err := p.ParseTestCasesZip(zipData)
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
	p := parser.NewICPCParser(200, 2, 10000, 10, nil)

	zipData := createZipBuffer(map[string][]byte{
		"data/sample/1.in":            []byte("1 2"),
		"data/sample/../../etc/hosts": []byte("malicious"),
	})

	_, err := p.ParseTestCasesZip(zipData)
	if err == nil {
		t.Fatal("expected error for malicious path, got nil")
	}
}

func TestICPCParser_MissingContentDir(t *testing.T) {
	p := parser.NewICPCParser(200, 2, 10000, 10, nil)

	zipData := createZipBuffer(map[string][]byte{
		"some_other_dir/1.in": []byte("1 2"),
	})

	_, err := p.ParseTestCasesZip(zipData)
	if err == nil {
		t.Fatal("expected error for files outside testcases dirs")
	}
}

func TestICPCParser_TooManySamples(t *testing.T) {
	p := parser.NewICPCParser(200, 2, 10000, 1, nil)

	zipData := createZipBuffer(map[string][]byte{
		"data/sample/1.in": []byte("1 2"),
		"data/sample/2.in": []byte("3"),
	})

	_, err := p.ParseTestCasesZip(zipData)
	if err == nil {
		t.Fatal("expected error for too many samples, got nil")
	}
}
