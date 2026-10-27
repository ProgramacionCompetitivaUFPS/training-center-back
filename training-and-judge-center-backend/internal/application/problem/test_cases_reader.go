package problem

import (
	"archive/zip"
	"bytes"
	"io"
	"log/slog"
	"path"
	"strings"

	"context"

	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	TestCaseKindSample = "sample"
	TestCaseKindSecret = "secret"

	// TestCasePreviewMaxBytes caps what a preview returns per file: a secret case can be megabytes.
	TestCasePreviewMaxBytes = 64 << 10
)

// testCaseEntry is one case of the stored test cases ZIP: the .in/.ans pair sharing a name
// inside data/sample or data/secret. Either side can be missing in a malformed package.
type testCaseEntry struct {
	Kind   string
	Name   string
	Input  *zip.File
	Answer *zip.File
}

// testCaseArchive indexes a test cases ZIP that was already validated when it was uploaded.
// It finds the cases the same way the upload parser does: any data/sample or data/secret
// directory, whatever root folder the package has.
type testCaseArchive struct {
	entries []*testCaseEntry // samples first, then secret; each group in name order
}

func openTestCaseArchive(ctx context.Context, data []byte) (*testCaseArchive, error) {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		// Stored ZIPs passed validation on upload, so this is corruption, not user input.
		slog.ErrorContext(ctx, "test cases: stored ZIP is unreadable", "error", err)
		return nil, apperror.NewInternal()
	}

	byKey := make(map[string]*testCaseEntry)
	names := map[string][]string{TestCaseKindSample: nil, TestCaseKindSecret: nil}

	for _, f := range reader.File {
		if f.FileInfo().IsDir() || !f.Mode().IsRegular() {
			continue
		}
		clean := path.Clean(strings.ReplaceAll(f.Name, "\\", "/"))
		if strings.Contains(clean, "__MACOSX") || strings.HasSuffix(clean, ".DS_Store") || strings.HasPrefix(path.Base(clean), "._") {
			continue
		}
		ext := strings.ToLower(path.Ext(clean))
		if ext != ".in" && ext != ".ans" {
			continue
		}

		dir := path.Dir("/" + clean)
		var kind string
		switch {
		case strings.HasSuffix(dir, "/data/sample"):
			kind = TestCaseKindSample
		case strings.HasSuffix(dir, "/data/secret"):
			kind = TestCaseKindSecret
		default:
			continue
		}

		name := strings.TrimSuffix(path.Base(clean), path.Ext(clean))
		key := kind + "/" + name
		entry, ok := byKey[key]
		if !ok {
			entry = &testCaseEntry{Kind: kind, Name: name}
			byKey[key] = entry
			names[kind] = append(names[kind], name)
		}
		if ext == ".in" {
			entry.Input = f
		} else {
			entry.Answer = f
		}
	}

	archive := &testCaseArchive{}
	for _, kind := range []string{TestCaseKindSample, TestCaseKindSecret} {
		sortSampleNames(names[kind])
		for _, name := range names[kind] {
			archive.entries = append(archive.entries, byKey[kind+"/"+name])
		}
	}
	return archive, nil
}

func (a *testCaseArchive) find(kind, name string) *testCaseEntry {
	for _, e := range a.entries {
		if e.Kind == kind && e.Name == name {
			return e
		}
	}
	return nil
}

// readZipFile reads at most limit bytes of f and reports whether the file was longer.
func readZipFile(f *zip.File, limit int64) (content []byte, truncated bool, err error) {
	rc, err := f.Open()
	if err != nil {
		return nil, false, err
	}
	defer rc.Close()

	data, err := io.ReadAll(io.LimitReader(rc, limit+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > limit {
		return data[:limit], true, nil
	}
	return data, false, nil
}

func fileSize(f *zip.File) *int64 {
	if f == nil {
		return nil
	}
	size := int64(f.UncompressedSize64)
	return &size
}
