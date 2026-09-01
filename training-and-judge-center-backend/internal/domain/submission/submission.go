package submission

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type SubmissionID = string
type ProblemID = string
type ContestID = string

type Submission struct {
	id             SubmissionID
	problemID      ProblemID
	problemTitle   string
	problemSlug    string
	userID         shared.UserID
	contestID      *ContestID
	standingID     *string
	language       Language
	compiler       string
	status         SubmissionStatus
	visibility     Visibility
	sourceCodePath string
	fileHash       string
	fileSize       int

	submittedAt time.Time
	judgedAt    *time.Time

	timeMs     *int
	memoryKb   *int
	compileLog *string
}

func (s *Submission) ID() SubmissionID              { return s.id }
func (s *Submission) ProblemID() ProblemID          { return s.problemID }
func (s *Submission) ProblemTitle() string          { return s.problemTitle }
func (s *Submission) ProblemSlug() string           { return s.problemSlug }
func (s *Submission) UserID() shared.UserID         { return s.userID }
func (s *Submission) ContestID() *ContestID         { return s.contestID }
func (s *Submission) StandingID() *string           { return s.standingID }
func (s *Submission) Language() Language            { return s.language }
func (s *Submission) Compiler() string              { return s.compiler }
func (s *Submission) Status() SubmissionStatus      { return s.status }
func (s *Submission) Visibility() Visibility        { return s.visibility }
func (s *Submission) SourceCodePath() string        { return s.sourceCodePath }
func (s *Submission) FileHash() string              { return s.fileHash }
func (s *Submission) FileSize() int                 { return s.fileSize }
func (s *Submission) SubmittedAt() time.Time        { return s.submittedAt }
func (s *Submission) JudgedAt() *time.Time          { return s.judgedAt }
func (s *Submission) TimeMs() *int                  { return s.timeMs }
func (s *Submission) MemoryKb() *int                { return s.memoryKb }
func (s *Submission) CompileLog() *string           { return s.compileLog }

func (s *Submission) SetVisibility(v Visibility) { s.visibility = v }

func NewSubmission(
	id SubmissionID,
	problemID ProblemID,
	userID shared.UserID,
	contestID *ContestID,
	standingID *string,
	lang Language,
	compiler string,
	sourceCodePath string,
	fileHash string,
	fileSize int,
	problemTitle string,
	problemSlug string,
	now time.Time,
) (*Submission, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	return &Submission{
		id:             id,
		problemID:      problemID,
		problemTitle:   problemTitle,
		problemSlug:    problemSlug,
		userID:         userID,
		contestID:      contestID,
		standingID:     standingID,
		language:       lang,
		compiler:       compiler,
		status:         newStatusPending(),
		visibility:     NewVisibilityPrivate(),
		sourceCodePath: sourceCodePath,
		fileHash:       fileHash,
		fileSize:       fileSize,
		submittedAt:    now.UTC(),
	}, nil
}

func RestoreSubmission(
	id SubmissionID,
	problemID ProblemID,
	userID shared.UserID,
	contestID *ContestID,
	standingID *string,
	lang Language,
	compiler string,
	status SubmissionStatus,
	visibility Visibility,
	sourceCodePath string,
	fileHash string,
	fileSize int,
	submittedAt time.Time,
	judgedAt *time.Time,
	timeMs *int,
	memoryKb *int,
	compileLog *string,
	problemTitle string,
	problemSlug string,
) *Submission {
	return &Submission{
		id:             id,
		problemID:      problemID,
		problemTitle:   problemTitle,
		problemSlug:    problemSlug,
		userID:         userID,
		contestID:      contestID,
		standingID:     standingID,
		language:       lang,
		compiler:       compiler,
		status:         status,
		visibility:     visibility,
		sourceCodePath: sourceCodePath,
		fileHash:       fileHash,
		fileSize:       fileSize,
		submittedAt:    submittedAt,
		judgedAt:       judgedAt,
		timeMs:         timeMs,
		memoryKb:       memoryKb,
		compileLog:     compileLog,
	}
}

// Start transitions PENDING → RUNNING.
func (s *Submission) Start(now time.Time) error {
	if !s.status.IsPending() {
		return apperror.NewInternal()
	}
	s.status = newStatusRunning()
	return nil
}

// markFinal is the shared guard for all RUNNING → final transitions.
func (s *Submission) markFinal(now time.Time) error {
	if !s.status.IsRunning() {
		return apperror.NewInternal()
	}
	t := now.UTC()
	s.judgedAt = &t
	return nil
}

// memoryKb is a pointer because it may not have been measured: a zero would
// tell the contestant their solution used no memory at all.
func (s *Submission) MarkAccepted(timeMs int, memoryKb *int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusAccepted()
	s.timeMs = &timeMs
	s.memoryKb = memoryKb
	return nil
}

func (s *Submission) MarkWrongAnswer(timeMs int, memoryKb *int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusWrongAnswer()
	s.timeMs = &timeMs
	s.memoryKb = memoryKb
	return nil
}

func (s *Submission) MarkTimeLimitExceeded(timeMs int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusTimeLimitExceeded()
	s.timeMs = &timeMs
	return nil
}

func (s *Submission) MarkMemoryLimitExceeded(memoryKb *int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusMemoryLimitExceeded()
	s.memoryKb = memoryKb
	return nil
}

// MarkOutputLimitExceeded is what a run that wrote past the output limit gets.
// It is not a wrong answer: the output was cut short, so comparing it says
// nothing about whether the solution was right.
func (s *Submission) MarkOutputLimitExceeded(timeMs int, memoryKb *int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusOutputLimitExceeded()
	s.timeMs = &timeMs
	s.memoryKb = memoryKb
	return nil
}

func (s *Submission) MarkRuntimeError(timeMs int, memoryKb *int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusRuntimeError()
	s.timeMs = &timeMs
	s.memoryKb = memoryKb
	return nil
}

func (s *Submission) MarkCompilationError(log string, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusCompilationError()
	s.compileLog = &log
	return nil
}

func (s *Submission) MarkSystemError(now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusSystemError()
	return nil
}
