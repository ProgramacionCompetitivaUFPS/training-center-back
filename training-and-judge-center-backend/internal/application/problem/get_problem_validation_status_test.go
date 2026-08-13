package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newGetStatusUseCase(validationRepo *mockValidationRepository, statusProvider *mockProblemStatusProvider) *GetProblemValidationStatusUseCase {
	return NewGetProblemValidationStatusUseCase(validationRepo, statusProvider)
}

func newValidationFixture() *domainProblem.ProblemValidation {
	v, err := domainProblem.NewProblemValidation("validation-1", testProbID, shared.RestoreUserID(authorID), testNow)
	if err != nil {
		panic(err)
	}
	return v
}

func repoReturning(v *domainProblem.ProblemValidation) *mockValidationRepository {
	return &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return v, nil
		},
	}
}

func TestGetProblemValidationStatus_NotFinal_ReturnsNotTerminal(t *testing.T) {
	statusProvider := &mockProblemStatusProvider{
		getStatusFn: func(_ context.Context, _ string) (string, error) {
			t.Fatal("statusProvider should not be called before the validation reaches a final state")
			return "", nil
		},
	}
	uc := newGetStatusUseCase(repoReturning(newValidationFixture()), statusProvider) // PENDING

	out, err := uc.Execute(context.Background(), GetProblemValidationStatusInput{ValidationID: "validation-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Terminal {
		t.Error("Terminal: got true, want false")
	}
}

func TestGetProblemValidationStatus_RepositoryError_Propagates(t *testing.T) {
	wantErr := apperror.NewNotFound("VALIDATION_NOT_FOUND", "validation not found")
	validationRepo := &mockValidationRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, error) {
			return nil, wantErr
		},
	}
	uc := newGetStatusUseCase(validationRepo, &mockProblemStatusProvider{})

	_, err := uc.Execute(context.Background(), GetProblemValidationStatusInput{ValidationID: "missing"})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_NOT_FOUND" {
		t.Errorf("expected the repository error to propagate, got %v", err)
	}
}

func TestGetProblemValidationStatus_Passed_DecodesReportAndFetchesProblemStatus(t *testing.T) {
	v := newValidationFixture()
	if err := v.Start(testNow); err != nil {
		t.Fatalf("fixture Start: %v", err)
	}
	result := `{"validationLogs":["compiled ok","3/3 test cases passed"],"validationSummary":{"sampleCases":1,"secretCases":2,"solutionsTested":1,"allPassed":true}}`
	if err := v.MarkPassed(result, testNow); err != nil {
		t.Fatalf("fixture MarkPassed: %v", err)
	}

	statusProvider := &mockProblemStatusProvider{
		getStatusFn: func(_ context.Context, problemID string) (string, error) {
			if problemID != testProbID {
				t.Errorf("GetStatus problemID: got %q, want %q", problemID, testProbID)
			}
			return "PUBLISHED", nil
		},
	}
	uc := newGetStatusUseCase(repoReturning(v), statusProvider)

	out, err := uc.Execute(context.Background(), GetProblemValidationStatusInput{ValidationID: "validation-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Terminal {
		t.Fatal("Terminal: got false, want true")
	}
	if !out.Passed {
		t.Error("Passed: got false, want true")
	}
	if out.Status != "PUBLISHED" {
		t.Errorf("Status: got %q, want PUBLISHED", out.Status)
	}
	if len(out.Report.ValidationLogs) != 2 {
		t.Errorf("ValidationLogs: got %v, want 2 entries", out.Report.ValidationLogs)
	}
	if out.Report.ValidationSummary == nil || !out.Report.ValidationSummary.AllPassed {
		t.Errorf("ValidationSummary: got %v, want AllPassed=true", out.Report.ValidationSummary)
	}
}

func TestGetProblemValidationStatus_Failed_DecodesFailureDetails(t *testing.T) {
	v := newValidationFixture()
	if err := v.Start(testNow); err != nil {
		t.Fatalf("fixture Start: %v", err)
	}
	result := `{"validationLogs":["compiled ok"],"failedTestCases":[{"case":"secret/02","verdict":"WRONG_ANSWER"}]}`
	if err := v.MarkFailed(result, testNow); err != nil {
		t.Fatalf("fixture MarkFailed: %v", err)
	}

	uc := newGetStatusUseCase(repoReturning(v), &mockProblemStatusProvider{})

	out, err := uc.Execute(context.Background(), GetProblemValidationStatusInput{ValidationID: "validation-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Terminal {
		t.Fatal("Terminal: got false, want true")
	}
	if out.Passed {
		t.Error("Passed: got true, want false")
	}
	if len(out.Report.FailedTestCases) != 1 || out.Report.FailedTestCases[0].Case != "secret/02" {
		t.Errorf("FailedTestCases: got %v, want one entry for secret/02", out.Report.FailedTestCases)
	}
}

func TestGetProblemValidationStatus_MalformedResultJSON_ReturnsInternal(t *testing.T) {
	v := newValidationFixture()
	if err := v.Start(testNow); err != nil {
		t.Fatalf("fixture Start: %v", err)
	}
	if err := v.MarkSystemError("{not valid json", testNow); err != nil {
		t.Fatalf("fixture MarkSystemError: %v", err)
	}

	uc := newGetStatusUseCase(repoReturning(v), &mockProblemStatusProvider{})

	_, err := uc.Execute(context.Background(), GetProblemValidationStatusInput{ValidationID: "validation-1"})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected an internal error for malformed result JSON, got %v", err)
	}
}

func TestGetProblemValidationStatus_StatusProviderError_Propagates(t *testing.T) {
	v := newValidationFixture()
	if err := v.Start(testNow); err != nil {
		t.Fatalf("fixture Start: %v", err)
	}
	if err := v.MarkPassed(`{"validationLogs":[]}`, testNow); err != nil {
		t.Fatalf("fixture MarkPassed: %v", err)
	}

	statusProvider := &mockProblemStatusProvider{
		getStatusFn: func(_ context.Context, _ string) (string, error) {
			return "", apperror.NewInternal()
		},
	}
	uc := newGetStatusUseCase(repoReturning(v), statusProvider)

	_, err := uc.Execute(context.Background(), GetProblemValidationStatusInput{ValidationID: "validation-1"})
	if err == nil {
		t.Error("expected the statusProvider error to propagate, got nil")
	}
}
