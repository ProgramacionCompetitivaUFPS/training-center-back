package problem

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newValidateProblemUseCase(
	validationRepo *mockValidationRepository,
	judgingPreparer *mockJudgingPreparer,
	artifactWriter *mockJudgingArtifactWriter,
	solutionValidator *mockSolutionValidator,
	publisher *mockProblemPublisher,
) *ValidateProblemUseCase {
	return NewValidateProblemUseCase(validationRepo, judgingPreparer, artifactWriter, solutionValidator, publisher, &mockTransactionManager{}, repoWith(nil), &mockStagedFileCleaner{})
}

func TestValidateProblem_NotPending_IsIgnored(t *testing.T) {
	v := newValidationFixture()
	if err := v.Start(testNow); err != nil {
		t.Fatalf("fixture Start: %v", err)
	}

	saveCalled := false
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
		saveFn:     func(_ context.Context, _ *domainProblem.ProblemValidation) error { saveCalled = true; return nil },
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			t.Fatal("solutionValidator should not be called when the ticket isn't pending")
			return nil, nil
		},
	}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, solutionValidator, &mockProblemPublisher{})

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saveCalled {
		t.Error("expected Save to never be called")
	}
}

func TestValidateProblem_LoadFails_PropagatesError(t *testing.T) {
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return nil, apperror.NewNotFound("VALIDATION_NOT_FOUND", "validation not found")
		},
	}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, &mockSolutionValidator{}, &mockProblemPublisher{})

	err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "missing"})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_NOT_FOUND" {
		t.Errorf("expected the repository error to propagate, got %v", err)
	}
}

func TestValidateProblem_JudgingPreparationErrors_MarksSystemError(t *testing.T) {
	v := newValidationFixture()
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	judgingPreparer := &mockJudgingPreparer{
		prepareFn: func(_ context.Context, _, _ string) (*JudgingPreparationResult, error) {
			return nil, apperror.NewInternal()
		},
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			t.Fatal("solutionValidator should not be called when judging preparation fails")
			return nil, nil
		},
	}
	uc := newValidateProblemUseCase(validationRepo, judgingPreparer, &mockJudgingArtifactWriter{}, solutionValidator, &mockProblemPublisher{})

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("Status: got %v, want SYSTEM_ERROR", v.Status())
	}
}

func TestValidateProblem_CheckerCompileFails_MarksTicketFailedWithoutRunningSolutions(t *testing.T) {
	v := newValidationFixture()
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	judgingPreparer := &mockJudgingPreparer{
		prepareFn: func(_ context.Context, _, _ string) (*JudgingPreparationResult, error) {
			return &JudgingPreparationResult{
				Failure: &JudgingPreparationFailure{Kind: JudgingFailureCheckerCompileError, FileKey: "checker.cpp", Log: "syntax error"},
			}, nil
		},
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			t.Fatal("solutionValidator should not be called when the checker fails to compile")
			return nil, nil
		},
	}
	uc := newValidateProblemUseCase(validationRepo, judgingPreparer, &mockJudgingArtifactWriter{}, solutionValidator, &mockProblemPublisher{})

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Status().String() != "FAILED" {
		t.Errorf("Status: got %v, want FAILED", v.Status())
	}

	var report ValidationReport
	if err := json.Unmarshal([]byte(*v.ResultJSON()), &report); err != nil {
		t.Fatalf("could not decode ResultJSON: %v", err)
	}
	if report.CompilationErrors == nil || report.CompilationErrors.File != "checker.cpp" {
		t.Errorf("CompilationErrors: got %+v", report.CompilationErrors)
	}
}

func TestValidateProblem_JudgingPrepared_PersistsCompiledKeys(t *testing.T) {
	v := newValidationFixture()
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	judgingPreparer := &mockJudgingPreparer{
		prepareFn: func(_ context.Context, _, _ string) (*JudgingPreparationResult, error) {
			return &JudgingPreparationResult{CheckerCompiledKey: "problems/x/checker/compiled", ValidatorCompiledKey: "problems/x/validator/compiled"}, nil
		},
	}
	artifactWriter := &mockJudgingArtifactWriter{}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			return &SolutionValidationResult{Passed: true, SolutionsTested: 1}, nil
		},
	}
	uc := newValidateProblemUseCase(validationRepo, judgingPreparer, artifactWriter, solutionValidator, &mockProblemPublisher{})

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(artifactWriter.checkerCompiledKeys) != 1 || artifactWriter.checkerCompiledKeys[0] != "problems/x/checker/compiled" {
		t.Errorf("checkerCompiledKeys: got %v", artifactWriter.checkerCompiledKeys)
	}
	if len(artifactWriter.validatorCompiledKeys) != 1 || artifactWriter.validatorCompiledKeys[0] != "problems/x/validator/compiled" {
		t.Errorf("validatorCompiledKeys: got %v", artifactWriter.validatorCompiledKeys)
	}
}

func TestValidateProblem_Passed_MarksPublishedAndTicketPassed(t *testing.T) {
	v := newValidationFixture()
	var savedStatuses []string
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
		saveFn: func(_ context.Context, saved *domainProblem.ProblemValidation) error {
			savedStatuses = append(savedStatuses, saved.Status().String())
			return nil
		},
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, gotProblemID string) (*SolutionValidationResult, error) {
			if gotProblemID != testProbID {
				t.Errorf("Validate problemID: got %q, want %q", gotProblemID, testProbID)
			}
			return &SolutionValidationResult{SampleCases: 1, SecretCases: 2, SolutionsTested: 1, Passed: true}, nil
		},
	}
	publisher := &mockProblemPublisher{}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, solutionValidator, publisher)

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publisher.calls != 1 {
		t.Errorf("expected MarkPublished called once, got %d", publisher.calls)
	}
	if len(savedStatuses) != 2 || savedStatuses[0] != "RUNNING" || savedStatuses[1] != "PASSED" {
		t.Errorf("savedStatuses: got %v, want [RUNNING PASSED]", savedStatuses)
	}

	if v.ResultJSON() == nil {
		t.Fatal("expected ResultJSON to be set")
	}
	var report ValidationReport
	if err := json.Unmarshal([]byte(*v.ResultJSON()), &report); err != nil {
		t.Fatalf("could not decode ResultJSON: %v", err)
	}
	if report.ValidationSummary == nil || !report.ValidationSummary.AllPassed || report.ValidationSummary.SecretCases != 2 {
		t.Errorf("ValidationSummary: got %+v", report.ValidationSummary)
	}
}

func TestValidateProblem_Failed_MarksTicketFailedWithoutPublishing(t *testing.T) {
	v := newValidationFixture()
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			return &SolutionValidationResult{
				Passed: false,
				Failure: &SolutionValidationFailure{
					FileKey: "sol.cpp", Kind: SolutionFailureWrongAnswer, TestCase: "secret/01",
					Expected: []byte("5"), Actual: []byte("3"),
				},
			}, nil
		},
	}
	publisher := &mockProblemPublisher{}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, solutionValidator, publisher)

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publisher.calls != 0 {
		t.Errorf("expected MarkPublished never called, got %d calls", publisher.calls)
	}
	if v.Status().String() != "FAILED" {
		t.Errorf("Status: got %v, want FAILED", v.Status())
	}

	var report ValidationReport
	if err := json.Unmarshal([]byte(*v.ResultJSON()), &report); err != nil {
		t.Fatalf("could not decode ResultJSON: %v", err)
	}
	if len(report.FailedTestCases) != 1 || report.FailedTestCases[0].Case != "secret/01" || report.FailedTestCases[0].Expected != "5" {
		t.Errorf("FailedTestCases: got %+v", report.FailedTestCases)
	}
}

func TestValidateProblem_SolutionValidatorErrors_MarksSystemError(t *testing.T) {
	v := newValidationFixture()
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			return nil, apperror.NewInternal()
		},
	}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, solutionValidator, &mockProblemPublisher{})

	if err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("Status: got %v, want SYSTEM_ERROR", v.Status())
	}
}

func TestValidateProblem_PublishFails_ReturnsErrorWithoutMarkingTicketPassed(t *testing.T) {
	v := newValidationFixture()
	saveCalls := 0
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) { return v, nil },
		saveFn:     func(_ context.Context, _ *domainProblem.ProblemValidation) error { saveCalls++; return nil },
	}
	solutionValidator := &mockSolutionValidator{
		validateFn: func(_ context.Context, _ string) (*SolutionValidationResult, error) {
			return &SolutionValidationResult{Passed: true, SolutionsTested: 1}, nil
		},
	}
	publisher := &mockProblemPublisher{
		markPublishedFn: func(_ context.Context, _ string, _ time.Time) error {
			return apperror.NewInternal()
		},
	}
	uc := newValidateProblemUseCase(validationRepo, &mockJudgingPreparer{}, &mockJudgingArtifactWriter{}, solutionValidator, publisher)

	err := uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1"})
	if err == nil {
		t.Fatal("expected an error when MarkPublished fails")
	}
	// Save should only have happened once, marking RUNNING — MarkPublished
	// failing must stop us before ever marking (and saving) the ticket
	// PASSED, so a stuck ticket is left RUNNING for the stale-recovery sweep.
	if saveCalls != 1 {
		t.Errorf("expected Save called only once (marking RUNNING), got %d", saveCalls)
	}
	if !v.Status().IsRunning() {
		t.Errorf("expected the ticket to remain RUNNING, got %v", v.Status())
	}
}

const stagedTestCasesKey = "problems/" + testSlug + "/testcases/staged"

func newJudgingUpdateTicket(testCases *string, checker *domainProblem.JudgingFile) *domainProblem.ProblemValidation {
	change, err := domainProblem.NewPendingJudgingChange(testCases, checker)
	if err != nil {
		panic(err)
	}
	v, err := domainProblem.NewJudgingUpdateValidation("validation-1", testProbID, shared.RestoreUserID(authorID), change, testNow)
	if err != nil {
		panic(err)
	}
	return v
}

func stagedChecker() *domainProblem.JudgingFile {
	f := domainProblem.RestoreJudgingFile("Checker.cpp", "problems/"+testSlug+"/checker/validation-1/Checker.cpp", "cpp20", nil, nil)
	return &f
}

type judgingUpdateFixture struct {
	uc        *ValidateProblemUseCase
	ticket    *domainProblem.ProblemValidation
	preparer  *mockJudgingPreparer
	validator *mockSolutionValidator
	staged    *mockStagedFileCleaner
	published *mockProblemPublisher
	problem   *domainProblem.Problem
	saved     []*domainProblem.Problem
}

func newJudgingUpdateFixture(ticket *domainProblem.ProblemValidation) *judgingUpdateFixture {
	f := &judgingUpdateFixture{
		ticket:    ticket,
		preparer:  &mockJudgingPreparer{},
		validator: &mockSolutionValidator{},
		staged:    &mockStagedFileCleaner{},
		published: &mockProblemPublisher{},
		problem:   newPublishedProblemWithTestCases(),
	}
	repo := repoWith(f.problem)
	repo.saveFn = func(_ context.Context, p *domainProblem.Problem) error {
		f.saved = append(f.saved, p)
		return nil
	}
	validationRepo := &mockValidationRepository{
		findByIDFn: func(context.Context, string) (*domainProblem.ProblemValidation, error) { return ticket, nil },
	}
	f.uc = NewValidateProblemUseCase(validationRepo, f.preparer, &mockJudgingArtifactWriter{}, f.validator, f.published, &mockTransactionManager{}, repo, f.staged)
	return f
}

func (f *judgingUpdateFixture) run(t *testing.T) {
	t.Helper()
	if err := f.uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1", Slug: testSlug}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProblem_JudgingUpdate_Passes_AppliesTheStagedFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, stagedChecker()))

	f.run(t)

	if f.ticket.Status().String() != "PASSED" {
		t.Fatalf("ticket status: got %v, want PASSED", f.ticket.Status())
	}
	if len(f.saved) != 1 {
		t.Fatalf("expected the problem saved once, got %d", len(f.saved))
	}
	p := f.saved[0]
	if p.TestCasesKey() == nil || *p.TestCasesKey() != stagedTestCasesKey {
		t.Errorf("TestCasesKey: got %v, want %s", p.TestCasesKey(), stagedTestCasesKey)
	}
	wantCompiled := "problems/" + testSlug + "/checker/validation-1/compiled"
	if p.Checker() == nil || p.Checker().CompiledKey() == nil || *p.Checker().CompiledKey() != wantCompiled {
		t.Errorf("checker compiled key: got %v, want %s", p.Checker(), wantCompiled)
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt must move so the affected submissions can be rejudged")
	}
	if len(f.staged.deletedPrefixes) != 0 {
		t.Errorf("nothing staged should be dropped on success, got %v", f.staged.deletedPrefixes)
	}
	if f.published.calls != 0 {
		t.Error("a judging update must not run the publish step")
	}
}

func TestValidateProblem_JudgingUpdate_ValidatesTheCandidateNotTheLiveFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, stagedChecker()))

	f.run(t)

	want := &JudgingCandidate{
		TestCasesKey:       stagedTestCasesKey,
		CheckerFileKey:     "problems/" + testSlug + "/checker/validation-1/Checker.cpp",
		CheckerLanguage:    "cpp20",
		CheckerCompiledKey: "problems/" + testSlug + "/checker/validation-1/compiled",
	}
	for name, got := range map[string][]*JudgingCandidate{"preparer": f.preparer.candidates, "validator": f.validator.candidates} {
		if len(got) != 1 || got[0] == nil || *got[0] != *want {
			t.Errorf("%s candidate: got %+v, want %+v", name, got, want)
		}
	}
}

func TestValidateProblem_JudgingUpdate_TestCasesOnly_LeavesTheCheckerAlone(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, nil))

	f.run(t)

	if got := f.preparer.candidates[0]; got.CheckerFileKey != "" {
		t.Errorf("a test-cases-only update must not carry a checker, got %+v", got)
	}
	if f.saved[0].Checker() != nil {
		t.Errorf("checker: got %+v, want it untouched", f.saved[0].Checker())
	}
}

func TestValidateProblem_JudgingUpdate_SolutionFails_KeepsTheProblemAndDropsTheStagedFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, stagedChecker()))
	f.validator.validateFn = func(context.Context, string) (*SolutionValidationResult, error) {
		return &SolutionValidationResult{
			Passed:  false,
			Failure: &SolutionValidationFailure{Kind: SolutionFailureWrongAnswer, TestCase: "secret/1"},
		}, nil
	}

	f.run(t)

	if f.ticket.Status().String() != "FAILED" {
		t.Fatalf("ticket status: got %v, want FAILED", f.ticket.Status())
	}
	if len(f.saved) != 0 {
		t.Error("the live problem must be left untouched when the update fails")
	}
	want := []string{stagedTestCasesKey, "problems/" + testSlug + "/checker/validation-1/"}
	if len(f.staged.deletedPrefixes) != 2 || f.staged.deletedPrefixes[0] != want[0] || f.staged.deletedPrefixes[1] != want[1] {
		t.Errorf("dropped prefixes: got %v, want %v", f.staged.deletedPrefixes, want)
	}
}

func TestValidateProblem_JudgingUpdate_CheckerDoesNotCompile_FailsBeforeRunningSolutions(t *testing.T) {
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(nil, stagedChecker()))
	f.preparer.prepareFn = func(context.Context, string, string) (*JudgingPreparationResult, error) {
		return &JudgingPreparationResult{
			Failure: &JudgingPreparationFailure{Kind: JudgingFailureCheckerCompileError, FileKey: "Checker.cpp", Log: "syntax error"},
		}, nil
	}
	f.validator.validateFn = func(context.Context, string) (*SolutionValidationResult, error) {
		t.Fatal("solutions must not run when the candidate checker does not compile")
		return nil, nil
	}

	f.run(t)

	if f.ticket.Status().String() != "FAILED" {
		t.Errorf("ticket status: got %v, want FAILED", f.ticket.Status())
	}
	if len(f.saved) != 0 {
		t.Error("the live problem must be left untouched")
	}
}

func TestValidateProblem_JudgingUpdate_InfraError_MarksSystemErrorAndDropsTheStagedFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, nil))
	f.preparer.prepareFn = func(context.Context, string, string) (*JudgingPreparationResult, error) {
		return nil, context.DeadlineExceeded
	}

	f.run(t)

	if f.ticket.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("ticket status: got %v, want SYSTEM_ERROR", f.ticket.Status())
	}
	if len(f.staged.deletedPrefixes) != 1 || f.staged.deletedPrefixes[0] != stagedTestCasesKey {
		t.Errorf("dropped prefixes: got %v", f.staged.deletedPrefixes)
	}
}
