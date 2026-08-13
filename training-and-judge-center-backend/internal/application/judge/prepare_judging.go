package judge

import (
	"context"
	"fmt"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

type PrepareJudgingInput struct {
	ProblemID string
	// Slug is only used to build a readable storage path for the compiled
	// artifacts — never to look anything up.
	Slug string
}

type PrepareJudgingFailureKind string

const (
	PrepareJudgingCheckerCompileError   PrepareJudgingFailureKind = "CHECKER_COMPILE_ERROR"
	PrepareJudgingValidatorCompileError PrepareJudgingFailureKind = "VALIDATOR_COMPILE_ERROR"
	PrepareJudgingInputRejected         PrepareJudgingFailureKind = "INPUT_REJECTED"
)

type PrepareJudgingFailure struct {
	Kind     PrepareJudgingFailureKind
	FileKey  string // set for the two *_COMPILE_ERROR kinds
	Log      string // set for the two *_COMPILE_ERROR kinds
	TestCase string // set for INPUT_REJECTED
	Reason   string // set for INPUT_REJECTED
}

type PrepareJudgingOutput struct {
	// CheckerCompiledKey/ValidatorCompiledKey are empty when nothing was
	// uploaded for that role — neither is an error, both are optional.
	CheckerCompiledKey   string
	ValidatorCompiledKey string
	Failure              *PrepareJudgingFailure // nil when everything compiled and every input was accepted
}

// compiledValidator carries a just-compiled validator's artifact forward to
// the input-checking step — the artifact is used in memory, straight from
// NativeCompiler's output, never re-downloaded from storage.
type compiledValidator struct {
	compiledKey string
	artifact    []byte
	filename    string
	language    submission.Language
}

type PrepareJudgingUseCase struct {
	sourceProvider   JudgingSourceProvider
	downloader       SourceCodeDownloader
	compiler         NativeCompiler
	uploader         ArtifactUploader
	testCaseProvider TestCaseProvider
	runner           ValidatorRunner
}

func NewPrepareJudgingUseCase(
	sourceProvider JudgingSourceProvider,
	downloader SourceCodeDownloader,
	compiler NativeCompiler,
	uploader ArtifactUploader,
	testCaseProvider TestCaseProvider,
	runner ValidatorRunner,
) *PrepareJudgingUseCase {
	return &PrepareJudgingUseCase{
		sourceProvider:   sourceProvider,
		downloader:       downloader,
		compiler:         compiler,
		uploader:         uploader,
		testCaseProvider: testCaseProvider,
		runner:           runner,
	}
}

// Execute compiles the checker and the validator (whichever are uploaded)
// and, if the validator compiled, runs it against every test case's input —
// stopping at the first problem it finds, same fail-fast spirit as
// ValidateSolutionsUseCase.
func (uc *PrepareJudgingUseCase) Execute(ctx context.Context, in PrepareJudgingInput) (*PrepareJudgingOutput, error) {
	checkerKey, failure, err := uc.prepareChecker(ctx, in.ProblemID, in.Slug)
	if err != nil {
		return nil, err
	}
	if failure != nil {
		return &PrepareJudgingOutput{Failure: failure}, nil
	}

	validator, failure, err := uc.prepareValidator(ctx, in.ProblemID, in.Slug)
	if err != nil {
		return nil, err
	}
	out := &PrepareJudgingOutput{CheckerCompiledKey: checkerKey}
	if failure != nil {
		out.Failure = failure
		return out, nil
	}
	if validator == nil {
		return out, nil
	}
	out.ValidatorCompiledKey = validator.compiledKey

	testCases, err := uc.testCaseProvider.GetTestCases(ctx, in.ProblemID)
	if err != nil {
		return nil, err
	}

	for _, tc := range testCases {
		result, err := uc.runner.Run(ctx, ValidatorRunRequest{
			Filename: validator.filename,
			Language: validator.language,
			Artifact: validator.artifact,
			Input:    tc.Input,
		})
		if err != nil {
			return nil, err
		}
		if !result.Accepted {
			out.Failure = &PrepareJudgingFailure{
				Kind:     PrepareJudgingInputRejected,
				TestCase: tc.Name,
				Reason:   string(truncatePreview([]byte(result.Message))),
			}
			return out, nil
		}
	}
	return out, nil
}

func (uc *PrepareJudgingUseCase) prepareChecker(ctx context.Context, problemID, slug string) (string, *PrepareJudgingFailure, error) {
	source, err := uc.sourceProvider.GetCheckerSource(ctx, problemID)
	if err != nil {
		return "", nil, err
	}
	if source == nil {
		return "", nil, nil
	}

	sourceCode, err := uc.downloader.Download(ctx, source.FileKey)
	if err != nil {
		return "", nil, err
	}

	result, err := uc.compiler.Compile(ctx, CompileArtifactRequest{
		Filename:   source.Filename,
		Language:   source.Language,
		SourceCode: sourceCode,
	})
	if err != nil {
		return "", nil, err
	}
	if !result.Success {
		return "", &PrepareJudgingFailure{
			Kind:    PrepareJudgingCheckerCompileError,
			FileKey: source.FileKey,
			Log:     string(truncatePreview([]byte(result.Log))),
		}, nil
	}

	compiledKey := fmt.Sprintf("problems/%s/checker/compiled", slug)
	if err := uc.uploader.Upload(ctx, compiledKey, result.Artifact); err != nil {
		return "", nil, err
	}
	return compiledKey, nil, nil
}

func (uc *PrepareJudgingUseCase) prepareValidator(ctx context.Context, problemID, slug string) (*compiledValidator, *PrepareJudgingFailure, error) {
	source, err := uc.sourceProvider.GetValidatorSource(ctx, problemID)
	if err != nil {
		return nil, nil, err
	}
	if source == nil {
		return nil, nil, nil
	}

	sourceCode, err := uc.downloader.Download(ctx, source.FileKey)
	if err != nil {
		return nil, nil, err
	}

	result, err := uc.compiler.Compile(ctx, CompileArtifactRequest{
		Filename:   source.Filename,
		Language:   source.Language,
		SourceCode: sourceCode,
	})
	if err != nil {
		return nil, nil, err
	}
	if !result.Success {
		return nil, &PrepareJudgingFailure{
			Kind:    PrepareJudgingValidatorCompileError,
			FileKey: source.FileKey,
			Log:     string(truncatePreview([]byte(result.Log))),
		}, nil
	}

	compiledKey := fmt.Sprintf("problems/%s/validator/compiled", slug)
	if err := uc.uploader.Upload(ctx, compiledKey, result.Artifact); err != nil {
		return nil, nil, err
	}

	return &compiledValidator{
		compiledKey: compiledKey,
		artifact:    result.Artifact,
		filename:    source.Filename,
		language:    source.Language,
	}, nil, nil
}
