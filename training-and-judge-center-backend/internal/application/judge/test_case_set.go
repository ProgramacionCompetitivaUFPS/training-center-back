package judge

import "io"

// TestCaseRef names one case and says how big its two files are, so a consumer can
// size what it sends without reading them first.
type TestCaseRef struct {
	// Name is e.g. "sample/001" or "secret/002".
	Name       string
	InputSize  int64
	AnswerSize int64
}

// TestCaseSet gives the cases of one problem one at a time. Neither file is ever
// held whole by the set, so a problem's size does not set the judge's memory.
type TestCaseSet interface {
	// Cases lists the cases that have both files, in the order they are judged.
	Cases() []TestCaseRef
	OpenInput(name string) (io.ReadCloser, error)
	OpenAnswer(name string) (io.ReadCloser, error)
	// Close releases what backs the set. Safe to call more than once.
	Close() error
}
