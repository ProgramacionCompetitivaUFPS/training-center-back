package problem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	// Room for the form fields and multipart framing on top of the file itself.
	multipartOverheadBytes = 1 << 20
	maxFormFieldBytes      = 64 << 10
	// Matches the load balancer's backend timeout, so a slow upload is cut by us first.
	uploadReadTimeout       = 5 * time.Minute
	maxConcurrentZipUploads = 2
	uploadQueueWait         = 30 * time.Second
	uploadRetryAfterSeconds = "30"
	stagedUploadPattern     = "upload-*"
)

var (
	errFileTooLarge = errors.New("uploaded file exceeds the limit")
	errStaging      = errors.New("could not stage the upload")
	errUploadBusy   = errors.New("too many uploads in progress")
)

// stagedUpload is the received file, spooled to disk. Closing it deletes it.
type stagedUpload struct {
	*os.File
	size int64
}

func (s *stagedUpload) Size() int64 { return s.size }

func (s *stagedUpload) Close() error {
	closeErr := s.File.Close()
	removeErr := os.Remove(s.File.Name())
	return errors.Join(closeErr, removeErr)
}

type receivedUpload struct {
	fields   map[string]string
	fileName string
	file     *stagedUpload
}

func (u *receivedUpload) Close() {
	if u != nil && u.file != nil {
		_ = u.file.Close()
	}
}

// receiveUpload reads a multipart body without holding the file in memory: the file part
// goes straight to a temporary file and every other part is a small, capped form field.
func receiveUpload(w http.ResponseWriter, r *http.Request, maxFileBytes int64) (*receivedUpload, error) {
	// Not every ResponseWriter supports deadlines; the body cap below still applies.
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(uploadReadTimeout))
	r.Body = http.MaxBytesReader(w, r.Body, maxFileBytes+multipartOverheadBytes)

	reader, err := r.MultipartReader()
	if err != nil {
		return nil, err
	}

	up := &receivedUpload{fields: map[string]string{}}
	for {
		part, err := reader.NextPart()
		if errors.Is(err, io.EOF) {
			return up, nil
		}
		if err != nil {
			up.Close()
			return nil, err
		}

		if part.FormName() == "file" && part.FileName() != "" && up.file == nil {
			if err := up.stage(part, maxFileBytes); err != nil {
				up.Close()
				return nil, err
			}
			up.fileName = part.FileName()
			continue
		}

		value, err := io.ReadAll(io.LimitReader(part, maxFormFieldBytes))
		if err != nil {
			up.Close()
			return nil, err
		}
		up.fields[part.FormName()] = string(value)
	}
}

func (u *receivedUpload) stage(part io.Reader, maxFileBytes int64) error {
	tmp, err := os.CreateTemp("", stagedUploadPattern)
	if err != nil {
		return fmt.Errorf("%w: %v", errStaging, err)
	}
	// One byte past the cap tells an exact fit from an overflow.
	n, err := io.Copy(tmp, io.LimitReader(part, maxFileBytes+1))
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return err
		}
		return fmt.Errorf("%w: %v", errStaging, err)
	}
	if n > maxFileBytes {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return errFileTooLarge
	}
	u.file = &stagedUpload{File: tmp, size: n}
	return nil
}

// uploadGate bounds how many ZIPs are being received and checked at once. The zero
// value, and a nil gate, admit everything.
type uploadGate struct {
	slots chan struct{}
	wait  time.Duration
}

func newUploadGate(n int) *uploadGate {
	return &uploadGate{slots: make(chan struct{}, n), wait: uploadQueueWait}
}

func (g *uploadGate) acquire(ctx context.Context) (func(), error) {
	if g == nil || g.slots == nil {
		return func() {}, nil
	}
	timer := time.NewTimer(g.wait)
	defer timer.Stop()
	select {
	case g.slots <- struct{}{}:
		return func() { <-g.slots }, nil
	case <-timer.C:
		return nil, errUploadBusy
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// uploadFailure maps a receive error to the response; file-too-large is left to the caller.
func uploadFailure(w http.ResponseWriter, err error) *apperror.AppError {
	switch {
	case errors.Is(err, errUploadBusy):
		w.Header().Set("Retry-After", uploadRetryAfterSeconds)
		return apperror.NewServiceUnavailable("UPLOAD_BUSY", "Too many uploads in progress, try again shortly")
	case errors.Is(err, errStaging):
		return apperror.NewInternal()
	default:
		return multipartParseError(err)
	}
}

// RemoveStaleUploads deletes spool files a killed process left behind: an emptyDir
// outlives its container, so without this they would pile up across restarts.
func RemoveStaleUploads(dir string) (int, error) {
	matches, err := filepath.Glob(filepath.Join(dir, stagedUploadPattern))
	if err != nil {
		return 0, err
	}
	removed := 0
	for _, path := range matches {
		if err := os.Remove(path); err == nil {
			removed++
		}
	}
	return removed, nil
}
