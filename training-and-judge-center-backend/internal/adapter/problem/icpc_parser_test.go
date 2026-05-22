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
	p := problem.NewICPCParser(200, 2, 10000, 10, nil)

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
	p := problem.NewICPCParser(200, 2, 10000, 10, nil)

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
	p := problem.NewICPCParser(200, 2, 10000, 10, nil)

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
	p := problem.NewICPCParser(200, 2, 10000, 10, nil)

	zipData := createZipBuffer(t, map[string][]byte{
		"some_other_dir/1.in": []byte("1 2"),
	})

	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err == nil {
		t.Fatal("expected error for files outside testcases dirs")
	}
}

func TestICPCParser_TooManySamples(t *testing.T) {
	p := problem.NewICPCParser(200, 2, 10000, 1, nil)

	zipData := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in": []byte("1 2"),
		"data/sample/2.in": []byte("3"),
	})

	_, err := p.ParseTestCasesZip(context.Background(), zipData)
	if err == nil {
		t.Fatal("expected error for too many samples, got nil")
	}
}
