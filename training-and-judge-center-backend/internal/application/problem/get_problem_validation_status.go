package problem

import (
	"context"
	"encoding/json"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type GetProblemValidationStatusInput struct {
	ValidationID string
}

type GetProblemValidationStatusOutput struct {
	Terminal bool
	Passed   bool
	Status   string // the problem's current status (DRAFT/PUBLISHED); only meaningful when Terminal
	Report   ValidationReport
}

// ValidationReport mirrors the exact response shape the spec defines for
// publish — validationLogs is always present, the rest are set depending on
// which pipeline step failed (or none, on success).
type ValidationReport struct {
	ValidationLogs    []string           `json:"validationLogs"`
	ValidationSummary *ValidationSummary `json:"validationSummary,omitempty"`
	FailedTestCases   []FailedTestCase   `json:"failedTestCases,omitempty"`
	CompilationErrors *CompilationErrors `json:"compilationErrors,omitempty"`
	FailedInputs      []FailedInput      `json:"failedInputs,omitempty"`
}

type ValidationSummary struct {
	SampleCases     int  `json:"sampleCases"`
	SecretCases     int  `json:"secretCases"`
	SolutionsTested int  `json:"solutionsTested"`
	AllPassed       bool `json:"allPassed"`
}

type FailedTestCase struct {
	Case      string `json:"case"`
	Verdict   string `json:"verdict,omitempty"`
	Expected  string `json:"expected,omitempty"`
	Actual    string `json:"actual,omitempty"`
	Status    string `json:"status,omitempty"`
	Details   string `json:"details,omitempty"`
	TimeLimit *int   `json:"timeLimit,omitempty"`
}

type CompilationErrors struct {
	File   string   `json:"file"`
	Errors []string `json:"errors"`
}

type FailedInput struct {
	File   string `json:"file"`
	Reason string `json:"reason"`
}

type GetProblemValidationStatusUseCase struct {
	validationRepo problem.ProblemValidationRepository
	statusProvider ProblemStatusProvider
}

func NewGetProblemValidationStatusUseCase(validationRepo problem.ProblemValidationRepository, statusProvider ProblemStatusProvider) *GetProblemValidationStatusUseCase {
	return &GetProblemValidationStatusUseCase{validationRepo: validationRepo, statusProvider: statusProvider}
}

func (uc *GetProblemValidationStatusUseCase) Execute(ctx context.Context, in GetProblemValidationStatusInput) (*GetProblemValidationStatusOutput, error) {
	v, err := uc.validationRepo.FindByID(ctx, in.ValidationID)
	if err != nil {
		return nil, err
	}
	if !v.Status().IsFinal() {
		return &GetProblemValidationStatusOutput{Terminal: false}, nil
	}

	var report ValidationReport
	if v.ResultJSON() != nil {
		if err := json.Unmarshal([]byte(*v.ResultJSON()), &report); err != nil {
			return nil, apperror.NewInternal()
		}
	}

	status, err := uc.statusProvider.GetStatus(ctx, v.ProblemID())
	if err != nil {
		return nil, err
	}

	return &GetProblemValidationStatusOutput{
		Terminal: true,
		Passed:   v.Status().IsPassed(),
		Status:   status,
		Report:   report,
	}, nil
}
