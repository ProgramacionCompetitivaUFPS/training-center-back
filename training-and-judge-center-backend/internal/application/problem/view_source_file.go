package problem

import (
	"context"
	"strings"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// SourceFilePreviewMaxBytes caps what the view returns. Solutions, checkers and validators are
// source code, so this is far above any real one; it only bounds a pathological upload.
const SourceFilePreviewMaxBytes = 256 << 10

type GetSourceFileInput struct {
	Slug     string
	FileType string // solution | checker | validator
	// FileName picks the solution (a problem can have several); ignored for checker/validator.
	FileName    string
	CurrentUser appshared.CurrentUser
}

type GetSourceFileOutput struct {
	FileType  string
	Filename  string
	Language  string
	Content   string
	Truncated bool
	SizeBytes int
}

type GetSourceFileUseCase struct {
	repo    problem.Repository
	storage ProblemFileRepository
}

func NewGetSourceFileUseCase(repo problem.Repository, storage ProblemFileRepository) *GetSourceFileUseCase {
	return &GetSourceFileUseCase{repo: repo, storage: storage}
}

func (uc *GetSourceFileUseCase) Execute(ctx context.Context, in GetSourceFileInput) (*GetSourceFileOutput, error) {
	fileType := strings.ToLower(in.FileType)
	switch fileType {
	case FileTypeSolution, FileTypeChecker, FileTypeValidator:
	case FileTypeTestCases:
		return nil, apperror.NewBadRequest(ErrCodeProblemInvalidFileType, "Test cases are downloaded as a ZIP (GET /files/testcases) or browsed case by case (GET /testcases)")
	default:
		return nil, apperror.NewBadRequest(ErrCodeProblemInvalidFileType, "Invalid file type. Allowed: solution, checker, validator")
	}
	if fileType == FileTypeSolution && in.FileName == "" {
		return nil, apperror.NewBadRequest(ErrCodeProblemMissingFilename, "fileName is required to read a solution")
	}

	p, err := loadProblemForFiles(ctx, uc.repo, in.Slug, in.CurrentUser)
	if err != nil {
		return nil, err
	}

	file := findJudgingFile(p, fileType, in.FileName)
	if file == nil {
		return nil, apperror.NewNotFound(ErrCodeProblemFileNotFound, "The problem has no such file")
	}

	data, err := uc.storage.DownloadFile(ctx, file.FileKey())
	if err != nil {
		return nil, missingFileError(err, "The file "+file.Filename())
	}

	out := &GetSourceFileOutput{
		FileType:  fileType,
		Filename:  file.Filename(),
		Language:  file.Language(),
		SizeBytes: len(data),
	}
	if len(data) > SourceFilePreviewMaxBytes {
		data = data[:SourceFilePreviewMaxBytes]
		out.Truncated = true
	}
	// A cut can land inside a multi-byte character; judge sources aren't guaranteed UTF-8 either.
	out.Content = strings.ToValidUTF8(string(data), "")
	return out, nil
}

func findJudgingFile(p *problem.Problem, fileType, fileName string) *problem.JudgingFile {
	switch fileType {
	case FileTypeChecker:
		return p.Checker()
	case FileTypeValidator:
		return p.Validator()
	}
	for _, s := range p.Solutions() {
		if s.Filename() == fileName {
			found := s
			return &found
		}
	}
	return nil
}
