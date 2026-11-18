package judge

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
)

// TestDataLimits bound what a test case archive may hold. They come from the same
// virtual_object.json the API enforces at upload, so what the API let in is what the
// worker accepts, and anything past them is refused before any case is read.
type TestDataLimits struct {
	MaxZipBytes       int64
	MaxExtractedBytes int64
	MaxFiles          int
	MaxInputBytes     int64
	MaxAnswerBytes    int64
}

// errInvalidTestData marks an archive that will never be usable: retrying it is pointless.
var errInvalidTestData = errors.New("invalid test case archive")

// caseEntryPath follows the ICPC Problem Package Format (data/sample/ and data/secret/):
// it returns the entry's path relative to data/, e.g. "secret/001.in".
//
// Searching for the group (and not requiring it at the root) accepts a package wrapped
// in a directory, e.g. "problem-abc/data/secret/001.in". Only "data/" is stripped, so
// the sample/secret group stays part of the name.
func caseEntryPath(entryName string) (string, bool) {
	cleanPath := path.Clean(strings.ReplaceAll(entryName, "\\", "/"))
	if idx := strings.Index(cleanPath, "data/sample/"); idx >= 0 {
		return cleanPath[idx+len("data/"):], true
	}
	if idx := strings.Index(cleanPath, "data/secret/"); idx >= 0 {
		return cleanPath[idx+len("data/"):], true
	}
	return "", false
}

// verifyTestCaseZip is the worker's own check of an archive, run once when it is cached
// and not per judging: the sizes it declares are within the limits, and every entry
// really decompresses to what it declares. The standard library enforces the declared
// size and the CRC as an entry is read, so reading each one to the end settles both.
func verifyTestCaseZip(ctx context.Context, zipPath string, limits TestDataLimits) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("%w: %v", errInvalidTestData, err)
	}
	defer zr.Close()

	var total uint64
	files := 0
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		files++
		if files > limits.MaxFiles {
			return fmt.Errorf("%w: more than %d files", errInvalidTestData, limits.MaxFiles)
		}
		if f.UncompressedSize64 > uint64(limits.MaxExtractedBytes) || total > uint64(limits.MaxExtractedBytes)-f.UncompressedSize64 {
			return fmt.Errorf("%w: extracts to more than %d bytes", errInvalidTestData, limits.MaxExtractedBytes)
		}
		total += f.UncompressedSize64

		if rel, ok := caseEntryPath(f.Name); ok {
			switch strings.ToLower(filepath.Ext(rel)) {
			case ".in":
				if f.UncompressedSize64 > uint64(limits.MaxInputBytes) {
					return fmt.Errorf("%w: %s declares %d bytes, over the %d byte input limit", errInvalidTestData, f.Name, f.UncompressedSize64, limits.MaxInputBytes)
				}
			case ".ans":
				if f.UncompressedSize64 > uint64(limits.MaxAnswerBytes) {
					return fmt.Errorf("%w: %s declares %d bytes, over the %d byte answer limit", errInvalidTestData, f.Name, f.UncompressedSize64, limits.MaxAnswerBytes)
				}
			}
		}
	}

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("%w: %s: %v", errInvalidTestData, f.Name, err)
		}
		_, copyErr := io.Copy(io.Discard, rc)
		rc.Close()
		if copyErr != nil {
			return fmt.Errorf("%w: %s: %v", errInvalidTestData, f.Name, copyErr)
		}
	}
	return nil
}

type caseFiles struct {
	input  *zip.File
	answer *zip.File
}

// zipTestCaseSet serves the cases of one cached archive. It owns its own handle on the
// file, so judgings sharing an archive share nothing but the bytes on disk.
type zipTestCaseSet struct {
	zr      *zip.ReadCloser
	release func()
	refs    []appJudge.TestCaseRef
	files   map[string]caseFiles
	once    sync.Once
}

var _ appJudge.TestCaseSet = (*zipTestCaseSet)(nil)

// openZipTestCaseSet indexes the archive's directory, which is all it reads up front.
// release runs when the set is closed, or at once if the archive cannot be opened.
func openZipTestCaseSet(zipPath string, release func()) (*zipTestCaseSet, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		release()
		return nil, fmt.Errorf("%w: %v", errInvalidTestData, err)
	}

	inputs := make(map[string]*zip.File)
	answers := make(map[string]*zip.File)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rel, ok := caseEntryPath(f.Name)
		if !ok || rel == "" {
			continue
		}
		name := strings.TrimSuffix(rel, filepath.Ext(rel))
		switch strings.ToLower(filepath.Ext(rel)) {
		case ".in":
			inputs[name] = f
		case ".ans":
			answers[name] = f
		}
	}

	names := make([]string, 0, len(inputs))
	for name := range inputs {
		if _, ok := answers[name]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	set := &zipTestCaseSet{
		zr:      zr,
		release: release,
		refs:    make([]appJudge.TestCaseRef, 0, len(names)),
		files:   make(map[string]caseFiles, len(names)),
	}
	for _, name := range names {
		in, ans := inputs[name], answers[name]
		set.refs = append(set.refs, appJudge.TestCaseRef{
			Name:       name,
			InputSize:  int64(in.UncompressedSize64),
			AnswerSize: int64(ans.UncompressedSize64),
		})
		set.files[name] = caseFiles{input: in, answer: ans}
	}
	return set, nil
}

func (s *zipTestCaseSet) Cases() []appJudge.TestCaseRef { return s.refs }

func (s *zipTestCaseSet) OpenInput(name string) (io.ReadCloser, error) {
	cf, ok := s.files[name]
	if !ok {
		return nil, fmt.Errorf("test case %q not found", name)
	}
	return cf.input.Open()
}

func (s *zipTestCaseSet) OpenAnswer(name string) (io.ReadCloser, error) {
	cf, ok := s.files[name]
	if !ok {
		return nil, fmt.Errorf("test case %q not found", name)
	}
	return cf.answer.Open()
}

func (s *zipTestCaseSet) Close() error {
	var err error
	s.once.Do(func() {
		err = s.zr.Close()
		s.release()
	})
	return err
}
