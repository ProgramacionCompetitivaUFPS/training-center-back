package problem

import (
	"archive/zip"
	"context"
	"strings"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// loadProblemForFiles resolves a problem and checks the caller may see its files. The test
// cases (secret ones above all) and the solutions are what the judge trusts, so only the people
// who can edit the problem can read them — never a regular user, even of a published problem.
func loadProblemForFiles(ctx context.Context, repo problem.Repository, slugValue string, user appshared.CurrentUser) (*problem.Problem, error) {
	slug, err := problem.NewSlug(slugValue)
	if err != nil {
		return nil, err
	}
	p, err := repo.FindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if !p.CanBeEditedBy(shared.RestoreUserID(user.ID), user.IsAdmin()) {
		return nil, apperror.NewForbidden(ErrCodeInsufficientPermissions, "Only the problem author, Admin, or assigned modifiers can view the problem files")
	}
	return p, nil
}

// loadTestCasesZip returns the ZIP the problem's test cases were uploaded as.
func loadTestCasesZip(ctx context.Context, storage ProblemFileRepository, p *problem.Problem) ([]byte, error) {
	if p.TestCasesKey() == nil {
		return nil, apperror.NewNotFound(ErrCodeProblemFileNotFound, "This problem has no test cases uploaded")
	}
	data, err := storage.DownloadFile(ctx, TestCasesZipKey(*p.TestCasesKey()))
	if err != nil {
		return nil, missingFileError(err, "The test cases ZIP")
	}
	return data, nil
}

// missingFileError tells "the problem points at a file the storage no longer has" apart from
// "the problem doesn't exist": both come back as not-found from the storage.
func missingFileError(err error, what string) error {
	if ae, ok := err.(*apperror.AppError); ok && ae.Kind == apperror.KindNotFound {
		return apperror.NewNotFound(ErrCodeProblemFileNotFound, what+" is registered for this problem but is missing from storage")
	}
	return err
}

// --- List ---

type ListTestCasesInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type TestCaseSummary struct {
	Kind        string // TestCaseKindSample | TestCaseKindSecret
	Name        string
	InputBytes  *int64 // nil when the .in file is missing
	AnswerBytes *int64 // nil when the .ans file is missing
}

type ListTestCasesOutput struct {
	HasTestCases bool
	SampleCount  int
	SecretCount  int
	Cases        []TestCaseSummary
}

type ListTestCasesUseCase struct {
	repo    problem.Repository
	storage ProblemFileRepository
}

func NewListTestCasesUseCase(repo problem.Repository, storage ProblemFileRepository) *ListTestCasesUseCase {
	return &ListTestCasesUseCase{repo: repo, storage: storage}
}

func (uc *ListTestCasesUseCase) Execute(ctx context.Context, in ListTestCasesInput) (*ListTestCasesOutput, error) {
	p, err := loadProblemForFiles(ctx, uc.repo, in.Slug, in.CurrentUser)
	if err != nil {
		return nil, err
	}
	if p.TestCasesKey() == nil {
		return &ListTestCasesOutput{Cases: []TestCaseSummary{}}, nil
	}

	data, err := loadTestCasesZip(ctx, uc.storage, p)
	if err != nil {
		return nil, err
	}
	archive, err := openTestCaseArchive(ctx, data)
	if err != nil {
		return nil, err
	}

	out := &ListTestCasesOutput{HasTestCases: true, Cases: make([]TestCaseSummary, 0, len(archive.entries))}
	for _, e := range archive.entries {
		if e.Kind == TestCaseKindSample {
			out.SampleCount++
		} else {
			out.SecretCount++
		}
		out.Cases = append(out.Cases, TestCaseSummary{
			Kind:        e.Kind,
			Name:        e.Name,
			InputBytes:  fileSize(e.Input),
			AnswerBytes: fileSize(e.Answer),
		})
	}
	return out, nil
}

// --- Preview of one case ---

type GetTestCaseInput struct {
	Slug        string
	Kind        string
	Name        string
	CurrentUser appshared.CurrentUser
}

type GetTestCaseOutput struct {
	Kind            string
	Name            string
	Input           string
	Answer          string
	InputTruncated  bool
	AnswerTruncated bool
}

type GetTestCaseUseCase struct {
	repo    problem.Repository
	storage ProblemFileRepository
}

func NewGetTestCaseUseCase(repo problem.Repository, storage ProblemFileRepository) *GetTestCaseUseCase {
	return &GetTestCaseUseCase{repo: repo, storage: storage}
}

func (uc *GetTestCaseUseCase) Execute(ctx context.Context, in GetTestCaseInput) (*GetTestCaseOutput, error) {
	if in.Kind != TestCaseKindSample && in.Kind != TestCaseKindSecret {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "kind", Message: "kind must be sample or secret"},
		})
	}

	p, err := loadProblemForFiles(ctx, uc.repo, in.Slug, in.CurrentUser)
	if err != nil {
		return nil, err
	}
	data, err := loadTestCasesZip(ctx, uc.storage, p)
	if err != nil {
		return nil, err
	}
	archive, err := openTestCaseArchive(ctx, data)
	if err != nil {
		return nil, err
	}

	entry := archive.find(in.Kind, in.Name)
	if entry == nil {
		return nil, apperror.NewNotFound(ErrCodeProblemFileNotFound, "Test case not found")
	}

	out := &GetTestCaseOutput{Kind: entry.Kind, Name: entry.Name}
	if entry.Input != nil {
		out.Input, out.InputTruncated, err = previewText(entry.Input)
		if err != nil {
			return nil, apperror.NewInternal()
		}
	}
	if entry.Answer != nil {
		out.Answer, out.AnswerTruncated, err = previewText(entry.Answer)
		if err != nil {
			return nil, apperror.NewInternal()
		}
	}
	return out, nil
}

// previewText reads the start of a case as text. A cut can land inside a multi-byte character,
// and judge data isn't guaranteed to be UTF-8, so invalid bytes are dropped rather than sent
// as something a JSON encoder would have to mangle.
func previewText(f *zip.File) (string, bool, error) {
	data, truncated, err := readZipFile(f, TestCasePreviewMaxBytes)
	if err != nil {
		return "", false, err
	}
	return strings.ToValidUTF8(string(data), ""), truncated, nil
}

// --- Download of the uploaded ZIP ---

type DownloadTestCasesInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type DownloadTestCasesOutput struct {
	FileName string
	Data     []byte
}

type DownloadTestCasesUseCase struct {
	repo    problem.Repository
	storage ProblemFileRepository
}

func NewDownloadTestCasesUseCase(repo problem.Repository, storage ProblemFileRepository) *DownloadTestCasesUseCase {
	return &DownloadTestCasesUseCase{repo: repo, storage: storage}
}

func (uc *DownloadTestCasesUseCase) Execute(ctx context.Context, in DownloadTestCasesInput) (*DownloadTestCasesOutput, error) {
	p, err := loadProblemForFiles(ctx, uc.repo, in.Slug, in.CurrentUser)
	if err != nil {
		return nil, err
	}
	data, err := loadTestCasesZip(ctx, uc.storage, p)
	if err != nil {
		return nil, err
	}
	return &DownloadTestCasesOutput{FileName: p.Slug().String() + "-testcases.zip", Data: data}, nil
}
