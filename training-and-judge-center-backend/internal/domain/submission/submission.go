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
	userID         shared.UserID
	contestID      *ContestID
	language       Language
	status         SubmissionStatus
	sourceCodePath string

	submittedAt time.Time
	judgedAt    *time.Time

	timeMs     *int
	memoryKb   *int
	compileLog *string
}

func (s *Submission) ID() SubmissionID              { return s.id }
func (s *Submission) ProblemID() ProblemID          { return s.problemID }
func (s *Submission) UserID() shared.UserID         { return s.userID }
func (s *Submission) ContestID() *ContestID         { return s.contestID }
func (s *Submission) Language() Language            { return s.language }
func (s *Submission) Status() SubmissionStatus      { return s.status }
func (s *Submission) SourceCodePath() string        { return s.sourceCodePath }
func (s *Submission) SubmittedAt() time.Time        { return s.submittedAt }
func (s *Submission) JudgedAt() *time.Time          { return s.judgedAt }
func (s *Submission) TimeMs() *int                  { return s.timeMs }
func (s *Submission) MemoryKb() *int                { return s.memoryKb }
func (s *Submission) CompileLog() *string           { return s.compileLog }

func NewSubmission(
	id SubmissionID,
	problemID ProblemID,
	userID shared.UserID,
	contestID *ContestID,
	lang Language,
	sourceCodePath string,
	now time.Time,
) (*Submission, error) {
	if id == "" {
		return nil, apperror.NewInternal()
	}
	return &Submission{
		id:             id,
		problemID:      problemID,
		userID:         userID,
		contestID:      contestID,
		language:       lang,
		status:         newStatusPending(),
		sourceCodePath: sourceCodePath,
		submittedAt:    now.UTC(),
	}, nil
}

func RestoreSubmission(
	id SubmissionID,
	problemID ProblemID,
	userID shared.UserID,
	contestID *ContestID,
	lang Language,
	status SubmissionStatus,
	sourceCodePath string,
	submittedAt time.Time,
	judgedAt *time.Time,
	timeMs *int,
	memoryKb *int,
	compileLog *string,
) *Submission {
	return &Submission{
		id:             id,
		problemID:      problemID,
		userID:         userID,
		contestID:      contestID,
		language:       lang,
		status:         status,
		sourceCodePath: sourceCodePath,
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

func (s *Submission) MarkAccepted(timeMs, memoryKb int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusAccepted()
	s.timeMs = &timeMs
	s.memoryKb = &memoryKb
	return nil
}

func (s *Submission) MarkWrongAnswer(timeMs, memoryKb int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusWrongAnswer()
	s.timeMs = &timeMs
	s.memoryKb = &memoryKb
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

func (s *Submission) MarkMemoryLimitExceeded(memoryKb int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusMemoryLimitExceeded()
	s.memoryKb = &memoryKb
	return nil
}

func (s *Submission) MarkRuntimeError(timeMs, memoryKb int, now time.Time) error {
	if err := s.markFinal(now); err != nil {
		return err
	}
	s.status = newStatusRuntimeError()
	s.timeMs = &timeMs
	s.memoryKb = &memoryKb
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
