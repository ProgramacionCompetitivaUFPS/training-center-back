package problem

import (
	"context"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	TestCasePartInput  = "input"
	TestCasePartAnswer = "answer"
)

type GetTestCasePartInput struct {
	Slug        string
	Kind        string // sample | secret
	Name        string
	Part        string // input | answer
	CurrentUser appshared.CurrentUser
}

// GetTestCasePartUseCase returns one file of a case in full. The preview endpoint cuts each file
// at 64 KiB because it answers inside a JSON; a test input can be up to 64 MB, so the full text of
// a case is served as plain text, one part at a time, and the client decides how to draw it.
type GetTestCasePartUseCase struct {
	repo    problem.Repository
	storage ProblemFileRepository
}

func NewGetTestCasePartUseCase(repo problem.Repository, storage ProblemFileRepository) *GetTestCasePartUseCase {
	return &GetTestCasePartUseCase{repo: repo, storage: storage}
}

func (uc *GetTestCasePartUseCase) Execute(ctx context.Context, in GetTestCasePartInput) ([]byte, error) {
	if in.Kind != TestCaseKindSample && in.Kind != TestCaseKindSecret {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "kind", Message: "kind must be sample or secret"},
		})
	}
	if in.Part != TestCasePartInput && in.Part != TestCasePartAnswer {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "part", Message: "part must be input or answer"},
		})
	}

	p, err := loadProblemForFiles(ctx, uc.repo, in.Slug, in.CurrentUser)
	if err != nil {
		return nil, err
	}
	zipData, err := loadTestCasesZip(ctx, uc.storage, p)
	if err != nil {
		return nil, err
	}
	archive, err := openTestCaseArchive(ctx, zipData)
	if err != nil {
		return nil, err
	}

	entry := archive.find(in.Kind, in.Name)
	if entry == nil {
		return nil, apperror.NewNotFound(ErrCodeProblemFileNotFound, "Test case not found")
	}
	file := entry.Input
	if in.Part == TestCasePartAnswer {
		file = entry.Answer
	}
	if file == nil {
		return nil, apperror.NewNotFound(ErrCodeProblemFileNotFound, "This test case has no "+in.Part+" file")
	}

	content, err := readWholeZipFile(file)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	return content, nil
}
