package submission_test

import (
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC)

const (
	testSubmissionID = "aaaaaaaa-0000-0000-0000-000000000001"
	testProblemID    = "bbbbbbbb-0000-0000-0000-000000000001"
	testUserID       = "cccccccc-0000-0000-0000-000000000001"
)

func newRunningSubmission(t *testing.T) *submission.Submission {
	t.Helper()
	lang := submission.RestoreLanguage("cpp20")
	s, err := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err != nil {
		t.Fatalf("NewSubmission: %v", err)
	}
	if err := s.Start(testNow); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return s
}

func TestNewSubmission_EmptyID_ReturnsError(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	_, err := submission.NewSubmission("", testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

func TestNewSubmission_SetsInitialState(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	s, err := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !s.Status().IsPending() {
		t.Errorf("expected PENDING, got %s", s.Status())
	}
	if s.JudgedAt() != nil {
		t.Error("expected nil JudgedAt")
	}
	if s.TimeMs() != nil {
		t.Error("expected nil TimeMs")
	}
	if s.MemoryKb() != nil {
		t.Error("expected nil MemoryKb")
	}
	if s.CompileLog() != nil {
		t.Error("expected nil CompileLog")
	}
	if got := s.SubmittedAt(); !got.Equal(testNow.UTC()) {
		t.Errorf("SubmittedAt: got %v, want %v", got, testNow.UTC())
	}
}

func TestSubmission_Start_FromPending_Succeeds(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	s, _ := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "path", "abc123", 512, "", "", testNow)

	if err := s.Start(testNow); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !s.Status().IsRunning() {
		t.Errorf("expected RUNNING, got %s", s.Status())
	}
}

func TestSubmission_Start_FromRunning_Fails(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.Start(testNow); err == nil {
		t.Error("expected error when Starting from RUNNING, got nil")
	}
}

func TestSubmission_MarkAccepted_FromRunning_Succeeds(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkAccepted(150, kb(4096), testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !s.Status().IsFinal() {
		t.Error("expected final status")
	}
	if s.Status().String() != "ACCEPTED" {
		t.Errorf("expected ACCEPTED, got %s", s.Status())
	}
	if s.TimeMs() == nil || *s.TimeMs() != 150 {
		t.Errorf("unexpected TimeMs: %v", s.TimeMs())
	}
	if s.MemoryKb() == nil || *s.MemoryKb() != 4096 {
		t.Errorf("unexpected MemoryKb: %v", s.MemoryKb())
	}
	if s.JudgedAt() == nil {
		t.Error("expected non-nil JudgedAt")
	}
}

func TestSubmission_MarkAccepted_FromPending_Fails(t *testing.T) {
	lang := submission.RestoreLanguage("cpp20")
	s, _ := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, lang, "g++", "path", "abc123", 512, "", "", testNow)
	if err := s.MarkAccepted(150, kb(4096), testNow); err == nil {
		t.Error("expected error when marking ACCEPTED from PENDING, got nil")
	}
}

func TestSubmission_MarkWrongAnswer(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkWrongAnswer(200, kb(2048), testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "WRONG_ANSWER" {
		t.Errorf("expected WRONG_ANSWER, got %s", s.Status())
	}
}

func TestSubmission_MarkTimeLimitExceeded(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkTimeLimitExceeded(5000, testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "TIME_LIMIT_EXCEEDED" {
		t.Errorf("expected TIME_LIMIT_EXCEEDED, got %s", s.Status())
	}
	if s.MemoryKb() != nil {
		t.Error("expected nil MemoryKb for TLE")
	}
}

func TestSubmission_MarkMemoryLimitExceeded(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkMemoryLimitExceeded(kb(262144), testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "MEMORY_LIMIT_EXCEEDED" {
		t.Errorf("expected MEMORY_LIMIT_EXCEEDED, got %s", s.Status())
	}
	if s.TimeMs() != nil {
		t.Error("expected nil TimeMs for MLE")
	}
}

func TestSubmission_MarkRuntimeError(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkRuntimeError(300, kb(1024), testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "RUNTIME_ERROR" {
		t.Errorf("expected RUNTIME_ERROR, got %s", s.Status())
	}
}

func TestSubmission_MarkCompilationError(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkCompilationError("error: undeclared identifier", testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "COMPILATION_ERROR" {
		t.Errorf("expected COMPILATION_ERROR, got %s", s.Status())
	}
	if s.CompileLog() == nil || *s.CompileLog() != "error: undeclared identifier" {
		t.Errorf("unexpected CompileLog: %v", s.CompileLog())
	}
}

func TestSubmission_MarkSystemError(t *testing.T) {
	s := newRunningSubmission(t)
	if err := s.MarkSystemError(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("expected SYSTEM_ERROR, got %s", s.Status())
	}
}

func TestSubmission_MarkFinal_FromFinal_Fails(t *testing.T) {
	s := newRunningSubmission(t)
	_ = s.MarkAccepted(100, kb(512), testNow)
	if err := s.MarkWrongAnswer(100, kb(512), testNow); err == nil {
		t.Error("expected error when re-marking final submission, got nil")
	}
}

// kb reads better than taking the address of a local at every call site. The
// verdicts take a pointer because a run may produce no measurement at all.
func kb(v int) *int { return &v }

// A cut-off output says nothing about whether the solution was right, so it is
// its own verdict and not a wrong answer.
func TestSubmission_MarkOutputLimitExceeded(t *testing.T) {
	s := newRunningSubmission(t)

	if err := s.MarkOutputLimitExceeded(120, kb(2048), testNow); err != nil {
		t.Fatalf("MarkOutputLimitExceeded: %v", err)
	}

	if got := s.Status().String(); got != "OUTPUT_LIMIT_EXCEEDED" {
		t.Errorf("status: got %q, want OUTPUT_LIMIT_EXCEEDED", got)
	}
	if !s.Status().IsFinal() {
		t.Error("the verdict is not final, so the stale sweep would overwrite it with SYSTEM_ERROR")
	}
	if s.JudgedAt() == nil {
		t.Error("judgedAt was not set")
	}
}

// NewStatus is what restores a verdict read back from the database; a verdict
// missing from it cannot survive a round trip.
func TestNewStatus_AcceptsOutputLimitExceeded(t *testing.T) {
	got, err := submission.NewStatus("OUTPUT_LIMIT_EXCEEDED")
	if err != nil {
		t.Fatalf("NewStatus: %v", err)
	}
	if got.String() != "OUTPUT_LIMIT_EXCEEDED" {
		t.Errorf("status: got %q", got.String())
	}
}

var localNow = time.Date(2025, 1, 15, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

func intPtr(v int) *int { return &v }

func assertInternalError(t *testing.T, label string, err error) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Errorf("%s: expected an internal apperror, got %v", label, err)
	}
}

func TestNewSubmission_StoresTheGivenFieldsAndTheInstantInUTC(t *testing.T) {
	contestID, standingID := "contest-1", "standing-1"

	s, err := submission.NewSubmission(
		testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), &contestID, &standingID,
		submission.RestoreLanguage("cpp20"), "g++", "gs://bucket/code.cpp", "abc123", 512,
		"Sum of Two Numbers", "sum-of-two-numbers", localNow,
	)

	if err != nil {
		t.Fatalf("NewSubmission: %v", err)
	}
	if s.ID() != testSubmissionID || s.ProblemID() != testProblemID || s.UserID().Value() != testUserID {
		t.Errorf("ids = %s/%s/%s", s.ID(), s.ProblemID(), s.UserID().Value())
	}
	if s.ProblemTitle() != "Sum of Two Numbers" || s.ProblemSlug() != "sum-of-two-numbers" {
		t.Errorf("problem title/slug = %q/%q", s.ProblemTitle(), s.ProblemSlug())
	}
	if s.ContestID() == nil || *s.ContestID() != "contest-1" || s.StandingID() == nil || *s.StandingID() != "standing-1" {
		t.Errorf("contest/standing = %v/%v, want contest-1/standing-1", s.ContestID(), s.StandingID())
	}
	if s.Language().String() != "cpp20" || s.Compiler() != "g++" {
		t.Errorf("language/compiler = %s/%s, want cpp20/g++", s.Language(), s.Compiler())
	}
	if s.SourceCodePath() != "gs://bucket/code.cpp" || s.FileHash() != "abc123" || s.FileSize() != 512 {
		t.Errorf("source = %s/%s/%d, want gs://bucket/code.cpp/abc123/512", s.SourceCodePath(), s.FileHash(), s.FileSize())
	}
	if want := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC); !s.SubmittedAt().Equal(want) || s.SubmittedAt().Location() != time.UTC {
		t.Errorf("SubmittedAt() = %v, want %v in UTC", s.SubmittedAt(), want)
	}
	if s.IsBlockly() || s.WorkspaceXMLPath() != nil {
		t.Error("a new submission is not a block editor submission")
	}
}

func TestRestoreSubmission_KeepsEveryStoredValue(t *testing.T) {
	contestID, standingID, compileLog := "contest-1", "standing-1", "warning: unused"
	judgedAt := testNow.Add(time.Minute)

	s := submission.RestoreSubmission(
		testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), &contestID, &standingID,
		submission.RestoreLanguage("java17"), "javac", submission.RestoreStatus("ACCEPTED"), submission.RestoreVisibility("PUBLIC"),
		"gs://bucket/Main.java", "def456", 2048, testNow, &judgedAt, intPtr(120), intPtr(4096), &compileLog,
		"Sum of Two Numbers", "sum-of-two-numbers",
	)

	if s.Status().String() != "ACCEPTED" || s.Visibility().String() != "PUBLIC" || s.Language().String() != "java17" || s.Compiler() != "javac" {
		t.Errorf("status/visibility/language/compiler = %s/%s/%s/%s", s.Status(), s.Visibility(), s.Language(), s.Compiler())
	}
	if s.JudgedAt() == nil || !s.JudgedAt().Equal(judgedAt) || s.TimeMs() == nil || *s.TimeMs() != 120 || s.MemoryKb() == nil || *s.MemoryKb() != 4096 {
		t.Errorf("judgedAt/timeMs/memoryKb = %v/%v/%v", s.JudgedAt(), s.TimeMs(), s.MemoryKb())
	}
	if s.CompileLog() == nil || *s.CompileLog() != "warning: unused" || s.ProblemTitle() != "Sum of Two Numbers" || s.ProblemSlug() != "sum-of-two-numbers" {
		t.Errorf("compileLog/title/slug = %v/%q/%q", s.CompileLog(), s.ProblemTitle(), s.ProblemSlug())
	}
	if s.FileHash() != "def456" || s.FileSize() != 2048 || s.SourceCodePath() != "gs://bucket/Main.java" || !s.SubmittedAt().Equal(testNow) {
		t.Errorf("source = %s/%d/%s/%v", s.FileHash(), s.FileSize(), s.SourceCodePath(), s.SubmittedAt())
	}
}

func TestSubmission_AttachBlocklyWorkspace_MarksItAsABlockSubmission(t *testing.T) {
	s := newRunningSubmission(t)

	s.AttachBlocklyWorkspace("gs://bucket/workspace.xml")

	if !s.IsBlockly() || s.WorkspaceXMLPath() == nil || *s.WorkspaceXMLPath() != "gs://bucket/workspace.xml" {
		t.Errorf("IsBlockly/WorkspaceXMLPath = %v/%v, want true and the stored path", s.IsBlockly(), s.WorkspaceXMLPath())
	}
}

func TestSubmission_SetVisibility_ReplacesTheVisibility(t *testing.T) {
	s := newRunningSubmission(t)

	s.SetVisibility(submission.NewVisibilityPublic())
	if !s.Visibility().IsPublic() {
		t.Error("Visibility() is not PUBLIC after SetVisibility(public)")
	}

	s.SetVisibility(submission.NewVisibilityPrivate())
	if !s.Visibility().IsPrivate() {
		t.Error("Visibility() is not PRIVATE after SetVisibility(private)")
	}
}

type verdictCase struct {
	name       string
	mark       func(s *submission.Submission, now time.Time) error
	wantStatus string
	wantTime   *int
	wantMemory *int
	wantLog    *string
}

func verdictCases() []verdictCase {
	log := "error: expected ';'"
	return []verdictCase{
		{"accepted", func(s *submission.Submission, now time.Time) error { return s.MarkAccepted(120, intPtr(2048), now) }, "ACCEPTED", intPtr(120), intPtr(2048), nil},
		{"wrong answer", func(s *submission.Submission, now time.Time) error { return s.MarkWrongAnswer(120, intPtr(2048), now) }, "WRONG_ANSWER", intPtr(120), intPtr(2048), nil},
		{"time limit exceeded", func(s *submission.Submission, now time.Time) error { return s.MarkTimeLimitExceeded(2000, now) }, "TIME_LIMIT_EXCEEDED", intPtr(2000), nil, nil},
		{"memory limit exceeded", func(s *submission.Submission, now time.Time) error {
			return s.MarkMemoryLimitExceeded(intPtr(262144), now)
		}, "MEMORY_LIMIT_EXCEEDED", nil, intPtr(262144), nil},
		{"output limit exceeded", func(s *submission.Submission, now time.Time) error {
			return s.MarkOutputLimitExceeded(300, intPtr(1024), now)
		}, "OUTPUT_LIMIT_EXCEEDED", intPtr(300), intPtr(1024), nil},
		{"runtime error", func(s *submission.Submission, now time.Time) error { return s.MarkRuntimeError(50, intPtr(512), now) }, "RUNTIME_ERROR", intPtr(50), intPtr(512), nil},
		{"compilation error", func(s *submission.Submission, now time.Time) error { return s.MarkCompilationError(log, now) }, "COMPILATION_ERROR", nil, nil, &log},
		{"system error", func(s *submission.Submission, now time.Time) error { return s.MarkSystemError(now) }, "SYSTEM_ERROR", nil, nil, nil},
	}
}

func TestSubmission_Verdicts_RecordOnlyTheirOwnMeasurementsAndTheInstantInUTC(t *testing.T) {
	for _, tt := range verdictCases() {
		t.Run(tt.name, func(t *testing.T) {
			s := newRunningSubmission(t)

			err := tt.mark(s, localNow)

			if err != nil {
				t.Fatalf("mark: %v", err)
			}
			if s.Status().String() != tt.wantStatus || !s.Status().IsFinal() {
				t.Errorf("Status() = %s, want the final status %s", s.Status(), tt.wantStatus)
			}
			if want := time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC); s.JudgedAt() == nil || !s.JudgedAt().Equal(want) || s.JudgedAt().Location() != time.UTC {
				t.Errorf("JudgedAt() = %v, want %v in UTC", s.JudgedAt(), want)
			}
			if !sameInt(s.TimeMs(), tt.wantTime) || !sameInt(s.MemoryKb(), tt.wantMemory) {
				t.Errorf("timeMs/memoryKb = %v/%v, want %v/%v", s.TimeMs(), s.MemoryKb(), tt.wantTime, tt.wantMemory)
			}
			if (s.CompileLog() == nil) != (tt.wantLog == nil) || (tt.wantLog != nil && *s.CompileLog() != *tt.wantLog) {
				t.Errorf("CompileLog() = %v, want %v", s.CompileLog(), tt.wantLog)
			}
		})
	}
}

func TestSubmission_Verdicts_NotRunning_AreAnInternalErrorAndChangeNothing(t *testing.T) {
	for _, tt := range verdictCases() {
		t.Run(tt.name+" while pending", func(t *testing.T) {
			s, err := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, submission.RestoreLanguage("cpp20"), "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
			if err != nil {
				t.Fatal(err)
			}

			err = tt.mark(s, testNow)

			assertInternalError(t, tt.name, err)
			if !s.Status().IsPending() || s.JudgedAt() != nil || s.TimeMs() != nil || s.MemoryKb() != nil || s.CompileLog() != nil {
				t.Errorf("a rejected verdict changed the submission: %s/%v/%v/%v/%v", s.Status(), s.JudgedAt(), s.TimeMs(), s.MemoryKb(), s.CompileLog())
			}
		})
		t.Run(tt.name+" after a final verdict", func(t *testing.T) {
			s := newRunningSubmission(t)
			if err := s.MarkAccepted(10, intPtr(100), testNow); err != nil {
				t.Fatal(err)
			}

			err := tt.mark(s, testNow.Add(time.Hour))

			assertInternalError(t, tt.name, err)
			if s.Status().String() != "ACCEPTED" || !s.JudgedAt().Equal(testNow) || *s.TimeMs() != 10 {
				t.Errorf("a final verdict was overwritten: %s/%v/%v", s.Status(), s.JudgedAt(), *s.TimeMs())
			}
		})
	}
}

func sameInt(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestNewStatus(t *testing.T) {
	valid := []string{
		"PENDING", "RUNNING", "ACCEPTED", "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED",
		"MEMORY_LIMIT_EXCEEDED", "OUTPUT_LIMIT_EXCEEDED", "RUNTIME_ERROR", "COMPILATION_ERROR", "SYSTEM_ERROR",
	}
	for _, raw := range valid {
		t.Run(raw, func(t *testing.T) {
			got, err := submission.NewStatus(raw)

			if err != nil {
				t.Fatalf("NewStatus(%q): %v", raw, err)
			}
			if got.String() != raw {
				t.Errorf("String() = %q, want %q", got.String(), raw)
			}
		})
	}

	for _, raw := range []string{"", "accepted", "DONE"} {
		t.Run("invalid "+raw, func(t *testing.T) {
			_, err := submission.NewStatus(raw)

			var appErr *apperror.AppError
			if !errors.As(err, &appErr) || appErr.Code != apperror.ErrCodeValidationError || len(appErr.Details) != 1 || appErr.Details[0].Field != "status" {
				t.Errorf("NewStatus(%q) = %v, want a validation error on status", raw, err)
			}
		})
	}
}

func TestNewSubmission_StartsPendingAndPrivate(t *testing.T) {
	s, err := submission.NewSubmission(testSubmissionID, testProblemID, shared.RestoreUserID(testUserID), nil, nil, submission.RestoreLanguage("cpp20"), "g++", "gs://bucket/code.cpp", "abc123", 512, "", "", testNow)
	if err != nil {
		t.Fatalf("NewSubmission: %v", err)
	}

	if !s.Status().IsPending() {
		t.Errorf("Status() = %s, want PENDING", s.Status())
	}
	if !s.Visibility().IsPrivate() {
		t.Errorf("Visibility() = %s, want PRIVATE: a new submission must not expose its code", s.Visibility())
	}
}
