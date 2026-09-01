package problem

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newValidateProblemUseCase(
	validationRepo *mockValidationRepository,
	judgingPreparer *mockJudgingPreparer,
	artifactWriter *mockJudgingArtifactWriter,
	solutionValidator *mockSolutionValidator,
	publisher *mockProblemPublisher,
) *ValidateProblemUseCase {
	return NewValidateProblemUseCase(validationRepo, judgingPreparer, artifactWriter, solutionValidator, publisher, &mockTransactionManager{})
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
