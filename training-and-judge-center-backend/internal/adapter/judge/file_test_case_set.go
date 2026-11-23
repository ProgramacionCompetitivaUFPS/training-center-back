package judge

import (
	"fmt"
	"io"
	"os"
	"sync"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
)

// fileTestCaseSet serves the cases of one cached problem straight from its extracted files.
// It keeps no file open: each case is opened when it is used, so a set holds nothing but its
// place in the cache.
type fileTestCaseSet struct {
	refs    []appJudge.TestCaseRef
	files   map[string]cachedCase
	release func()
	once    sync.Once
}

var _ appJudge.TestCaseSet = (*fileTestCaseSet)(nil)

func newFileTestCaseSet(cases []cachedCase, release func()) *fileTestCaseSet {
	set := &fileTestCaseSet{
		refs:    make([]appJudge.TestCaseRef, 0, len(cases)),
		files:   make(map[string]cachedCase, len(cases)),
		release: release,
	}
	for _, c := range cases {
		set.refs = append(set.refs, appJudge.TestCaseRef{Name: c.name, InputSize: c.inputSize, AnswerSize: c.answerSize})
		set.files[c.name] = c
	}
	return set
}

func (s *fileTestCaseSet) Cases() []appJudge.TestCaseRef { return s.refs }

func (s *fileTestCaseSet) OpenInput(name string) (io.ReadCloser, error) {
	c, ok := s.files[name]
	if !ok {
		return nil, fmt.Errorf("test case %q not found", name)
	}
	return os.Open(c.inputPath)
}

func (s *fileTestCaseSet) OpenAnswer(name string) (io.ReadCloser, error) {
	c, ok := s.files[name]
	if !ok {
		return nil, fmt.Errorf("test case %q not found", name)
	}
	return os.Open(c.answerPath)
}

func (s *fileTestCaseSet) Close() error {
	s.once.Do(s.release)
	return nil
}
