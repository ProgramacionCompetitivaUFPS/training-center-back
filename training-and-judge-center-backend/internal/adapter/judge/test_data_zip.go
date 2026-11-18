package judge

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// TestDataLimits bound what a test case archive may hold. They come from the same
// virtual_object.json the API enforces at upload, so what the API let in is what the
// worker accepts, and anything past them is refused before any case is extracted.
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

type plannedCase struct {
	name   string
	input  *zip.File
	answer *zip.File
}

// extractionPlan is what an archive will put on disk: only the cases that have both files,
// since nothing else in a package (statement, solutions) is ever read by a judging.
type extractionPlan struct {
	cases []plannedCase
	// bytes is the disk the extracted cases take, by their declared sizes.
	bytes int64
}

// planExtraction checks the archive's directory against the limits and lists the cases to
// extract, in the order they are judged. It reads headers only, so a package past a limit is
// refused before a single byte is decompressed.
func planExtraction(zr *zip.Reader, limits TestDataLimits) (extractionPlan, error) {
	var total uint64
	files := 0
	inputs := make(map[string]*zip.File)
	answers := make(map[string]*zip.File)

	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		files++
		if files > limits.MaxFiles {
			return extractionPlan{}, fmt.Errorf("%w: more than %d files", errInvalidTestData, limits.MaxFiles)
		}
		// Everything a package carries counts against the total, not only the test data.
		if f.UncompressedSize64 > uint64(limits.MaxExtractedBytes) || total > uint64(limits.MaxExtractedBytes)-f.UncompressedSize64 {
			return extractionPlan{}, fmt.Errorf("%w: extracts to more than %d bytes", errInvalidTestData, limits.MaxExtractedBytes)
		}
		total += f.UncompressedSize64

		rel, ok := caseEntryPath(f.Name)
		if !ok || rel == "" {
			continue
		}
		name := strings.TrimSuffix(rel, filepath.Ext(rel))
		switch strings.ToLower(filepath.Ext(rel)) {
		case ".in":
			if f.UncompressedSize64 > uint64(limits.MaxInputBytes) {
				return extractionPlan{}, fmt.Errorf("%w: %s declares %d bytes, over the %d byte input limit", errInvalidTestData, f.Name, f.UncompressedSize64, limits.MaxInputBytes)
			}
			inputs[name] = f
		case ".ans":
			if f.UncompressedSize64 > uint64(limits.MaxAnswerBytes) {
				return extractionPlan{}, fmt.Errorf("%w: %s declares %d bytes, over the %d byte answer limit", errInvalidTestData, f.Name, f.UncompressedSize64, limits.MaxAnswerBytes)
			}
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

	plan := extractionPlan{cases: make([]plannedCase, 0, len(names))}
	for _, name := range names {
		in, ans := inputs[name], answers[name]
		plan.cases = append(plan.cases, plannedCase{name: name, input: in, answer: ans})
		plan.bytes += int64(in.UncompressedSize64) + int64(ans.UncompressedSize64)
	}
	return plan, nil
}

// cachedCase is one extracted case: where its two files are and how big they are.
type cachedCase struct {
	name       string
	inputPath  string
	answerPath string
	inputSize  int64
	answerSize int64
}

// writeFailure marks an error of our own disk, as opposed to one of the archive's data.
type writeFailure struct{ err error }

func (w *writeFailure) Error() string { return w.err.Error() }
func (w *writeFailure) Unwrap() error { return w.err }

type failureWriter struct{ w io.Writer }

func (f failureWriter) Write(p []byte) (int, error) {
	n, err := f.w.Write(p)
	if err != nil {
		return n, &writeFailure{err}
	}
	return n, nil
}

// extractTestCases writes the planned cases into writeDir, which must not exist yet, and reads
// every one of them to the end: the standard library enforces the declared size and the CRC
// as an entry is read, so extracting is also what verifies the archive.
//
// writeDir is a scratch directory that is renamed to finalDir afterwards, so the paths it returns
// are the final ones: they must stay valid after the rename.
//
// Files are named by position and never by the entry's name, so nothing an archive says
// can choose where a byte lands.
func extractTestCases(ctx context.Context, plan extractionPlan, writeDir, finalDir string) ([]cachedCase, error) {
	if err := os.MkdirAll(writeDir, 0o755); err != nil {
		return nil, err
	}

	cases := make([]cachedCase, 0, len(plan.cases))
	for i, pc := range plan.cases {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		base := fmt.Sprintf("%05d", i)
		cc := cachedCase{
			name:       pc.name,
			inputPath:  filepath.Join(finalDir, base+".in"),
			answerPath: filepath.Join(finalDir, base+".ans"),
		}
		var err error
		if cc.inputSize, err = extractEntry(pc.input, filepath.Join(writeDir, base+".in")); err != nil {
			return nil, err
		}
		if cc.answerSize, err = extractEntry(pc.answer, filepath.Join(writeDir, base+".ans")); err != nil {
			return nil, err
		}
		cases = append(cases, cc)
	}
	return cases, nil
}

func extractEntry(f *zip.File, dst string) (int64, error) {
	rc, err := f.Open()
	if err != nil {
		return 0, fmt.Errorf("%w: %s: %v", errInvalidTestData, f.Name, err)
	}
	defer rc.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return 0, err
	}
	n, copyErr := io.Copy(failureWriter{out}, rc)
	closeErr := out.Close()

	var wf *writeFailure
	switch {
	case errors.As(copyErr, &wf):
		return 0, copyErr
	case copyErr != nil:
		return 0, fmt.Errorf("%w: %s: %v", errInvalidTestData, f.Name, copyErr)
	case closeErr != nil:
		return 0, closeErr
	}
	if n != int64(f.UncompressedSize64) || f.UncompressedSize64 > math.MaxInt64 {
		return 0, fmt.Errorf("%w: %s has %d bytes, declared %d", errInvalidTestData, f.Name, n, f.UncompressedSize64)
	}
	return n, nil
}
