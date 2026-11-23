package judge

import (
	"bytes"
	"fmt"
	"io"
)

// TestCase is one case held in memory, for sets that are small by construction.
type TestCase struct {
	Name           string
	Input          []byte
	ExpectedOutput []byte
}

type memoryTestCaseSet struct {
	refs  []TestCaseRef
	cases map[string]TestCase
}

// NewMemoryTestCaseSet serves cases already in memory. nil is the empty set, which
// is what a problem without test cases has.
func NewMemoryTestCaseSet(cases []TestCase) TestCaseSet {
	set := &memoryTestCaseSet{
		refs:  make([]TestCaseRef, 0, len(cases)),
		cases: make(map[string]TestCase, len(cases)),
	}
	for _, tc := range cases {
		set.refs = append(set.refs, TestCaseRef{Name: tc.Name, InputSize: int64(len(tc.Input)), AnswerSize: int64(len(tc.ExpectedOutput))})
		set.cases[tc.Name] = tc
	}
	return set
}

func (s *memoryTestCaseSet) Cases() []TestCaseRef { return s.refs }

func (s *memoryTestCaseSet) OpenInput(name string) (io.ReadCloser, error) {
	tc, ok := s.cases[name]
	if !ok {
		return nil, fmt.Errorf("test case %q not found", name)
	}
	return io.NopCloser(bytes.NewReader(tc.Input)), nil
}

func (s *memoryTestCaseSet) OpenAnswer(name string) (io.ReadCloser, error) {
	tc, ok := s.cases[name]
	if !ok {
		return nil, fmt.Errorf("test case %q not found", name)
	}
	return io.NopCloser(bytes.NewReader(tc.ExpectedOutput)), nil
}

func (s *memoryTestCaseSet) Close() error { return nil }
