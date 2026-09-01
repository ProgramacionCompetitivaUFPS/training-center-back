package judge

import (
	"archive/tar"
	"bytes"
	"io"
	"strings"
	"testing"
)

// referenceTar is archive/tar doing the whole job, header and padding included:
// the oracle the streamed tar has to match byte for byte.
func referenceTar(t *testing.T, filename string, content []byte, mode int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	if err := tw.WriteHeader(&tar.Header{Name: filename, Mode: mode, Size: int64(len(content))}); err != nil {
		t.Fatalf("reference WriteHeader: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("reference Write: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("reference Close: %v", err)
	}
	return buf.Bytes()
}

// firstDiff points at the byte that broke the comparison, or -1 when the two
// streams agree as far as the shorter one goes.
func firstDiff(a, b []byte) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return -1
}

// A tar reader stops cleanly at the end of the stream, so reading the archive
// back would accept a missing end-of-archive marker. Only the full bytes catch it.
func TestBuildTar_MatchesWhatArchiveTarWrites(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		size     int
		mode     int64
	}{
		{"empty file", "input.txt", 0, modeSource},
		{"single byte", "input.txt", 1, modeSource},
		{"one byte short of a record", "solution.cpp", 511, modeSource},
		{"exactly one record, so no padding", "checker", 512, modeExecutable},
		{"one byte past a record", "Checker.jar", 513, modeExecutable},
		{"many records", "input.txt", 100000, modeSource},
		{"name too long for the classic header", strings.Repeat("a", 120) + ".cpp", 1000, modeSource},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := bytes.Repeat([]byte{0xAB}, tt.size)
			want := referenceTar(t, tt.filename, content, tt.mode)

			got, err := io.ReadAll(buildTar(tt.filename, content, tt.mode))
			if err != nil {
				t.Fatalf("reading the streamed tar: %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("streamed tar differs from archive/tar: got %d bytes, want %d, first difference at byte %d",
					len(got), len(want), firstDiff(got, want))
			}
		})
	}
}

// The whole point of streaming: the payload stays in the caller's slice instead
// of being copied into the tar, so every file entering a container lived twice.
func TestBuildTar_DoesNotCopyTheContent(t *testing.T) {
	content := []byte("hello")

	r := buildTar("input.txt", content, modeSource)
	content[0] = 'H' // a materialized tar already holds its own copy of "hello"

	_, got := firstTarEntry(t, r)
	if string(got) != "Hello" {
		t.Errorf("content: got %q, want %q — the tar was materialized instead of streamed", got, "Hello")
	}
}
