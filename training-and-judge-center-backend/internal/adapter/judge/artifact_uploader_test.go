package judge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestArtifactUploader_Local_WritesFile(t *testing.T) {
	dir := t.TempDir()
	uploader := NewArtifactUploaderLocal(dir)

	err := uploader.Upload(context.Background(), "problems/abc/checker/compiled", []byte("binary content"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "problems/abc/checker/compiled"))
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if string(got) != "binary content" {
		t.Errorf("content: got %q, want %q", got, "binary content")
	}
}

func TestArtifactUploader_Local_OverwritesExisting(t *testing.T) {
	dir := t.TempDir()
	uploader := NewArtifactUploaderLocal(dir)
	ctx := context.Background()

	if err := uploader.Upload(ctx, "checker/compiled", []byte("first")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := uploader.Upload(ctx, "checker/compiled", []byte("second")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "checker/compiled"))
	if err != nil {
		t.Fatalf("expected file to exist: %v", err)
	}
	if string(got) != "second" {
		t.Errorf("content: got %q, want %q", got, "second")
	}
}
