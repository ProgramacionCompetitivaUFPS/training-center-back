package problem

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func multipartRequest(t *testing.T, build func(mw *multipart.Writer)) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	build(mw)
	_ = mw.Close()
	r := httptest.NewRequest(http.MethodPost, "/upload", &buf)
	r.Header.Set("Content-Type", mw.FormDataContentType())
	return r
}

func TestReceiveUpload_StagesTheFileOnDiskAndKeepsTheFields(t *testing.T) {
	r := multipartRequest(t, func(mw *multipart.Writer) {
		part, _ := mw.CreateFormFile("file", "cases.zip")
		_, _ = part.Write([]byte("payload"))
		// After the file on purpose: field order is the client's choice.
		_ = mw.WriteField("fileType", "testcases")
	})

	up, err := receiveUpload(httptest.NewRecorder(), r, 1024)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if up.fields["fileType"] != "testcases" || up.fileName != "cases.zip" {
		t.Errorf("fields/filename lost: %+v", up)
	}
	if up.file.Size() != int64(len("payload")) {
		t.Errorf("size = %d", up.file.Size())
	}
	path := up.file.Name()
	buf := make([]byte, up.file.Size())
	if _, err := up.file.ReadAt(buf, 0); err != nil || string(buf) != "payload" {
		t.Errorf("staged content = %q, err %v", buf, err)
	}

	up.Close()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("the temporary file must be deleted on Close, stat err = %v", err)
	}
}

func TestReceiveUpload_FileOverTheCap_IsRefusedAndLeavesNothingOnDisk(t *testing.T) {
	r := multipartRequest(t, func(mw *multipart.Writer) {
		part, _ := mw.CreateFormFile("file", "big.zip")
		_, _ = part.Write([]byte(strings.Repeat("a", 2048)))
	})

	_, err := receiveUpload(httptest.NewRecorder(), r, 1024)
	if !errors.Is(err, errFileTooLarge) {
		t.Fatalf("expected errFileTooLarge, got %v", err)
	}
}

func TestReceiveUpload_FileExactlyAtTheCap_IsAccepted(t *testing.T) {
	r := multipartRequest(t, func(mw *multipart.Writer) {
		part, _ := mw.CreateFormFile("file", "edge.zip")
		_, _ = part.Write([]byte(strings.Repeat("a", 1024)))
	})

	up, err := receiveUpload(httptest.NewRecorder(), r, 1024)
	if err != nil {
		t.Fatalf("a file exactly at the cap must pass, got %v", err)
	}
	up.Close()
}

func TestReceiveUpload_BodyFarBeyondTheCap_StopsReadingEarly(t *testing.T) {
	huge := strings.NewReader(strings.Repeat("a", 8<<20))
	r := httptest.NewRequest(http.MethodPost, "/upload", huge)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=xyz")

	_, err := receiveUpload(httptest.NewRecorder(), r, 1024)
	if err == nil {
		t.Fatal("expected an error")
	}
	if huge.Len() < 7<<20 {
		t.Errorf("read %d bytes of an 8 MiB body: the cap must stop it near %d", (8<<20)-huge.Len(), 1024+multipartOverheadBytes)
	}
}

func TestUploadGate_LimitsConcurrencyAndTimesOutWithBusy(t *testing.T) {
	g := newUploadGate(1)
	g.wait = 50 * time.Millisecond

	release, err := g.acquire(context.Background())
	if err != nil {
		t.Fatalf("first acquire: %v", err)
	}
	if _, err := g.acquire(context.Background()); !errors.Is(err, errUploadBusy) {
		t.Fatalf("second acquire should time out as busy, got %v", err)
	}

	release()
	release2, err := g.acquire(context.Background())
	if err != nil {
		t.Fatalf("a freed slot must be reusable, got %v", err)
	}
	release2()
}

func TestUploadGate_Nil_AdmitsEverything(t *testing.T) {
	var g *uploadGate
	release, err := g.acquire(context.Background())
	if err != nil {
		t.Fatalf("nil gate must admit, got %v", err)
	}
	release()
}

func TestUploadGate_CancelledContext_ReturnsTheContextError(t *testing.T) {
	g := newUploadGate(1)
	release, _ := g.acquire(context.Background())
	defer release()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := g.acquire(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestRemoveStaleUploads_RemovesOnlySpoolFiles(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "upload-123")
	other := filepath.Join(dir, "keep.txt")
	for _, p := range []string{stale, other} {
		if err := os.WriteFile(p, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	removed, err := RemoveStaleUploads(dir)
	if err != nil || removed != 1 {
		t.Fatalf("removed = %d, err = %v; want 1, nil", removed, err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("the spool file must be gone, stat err = %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("an unrelated file must stay: %v", err)
	}
}
