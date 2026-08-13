package judge

import (
	"context"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

func newPrepareJudgingUseCase(
	sources *mockJudgingSourceProvider,
	downloader *mockSourceCodeDownloader,
	compiler *mockNativeCompiler,
	uploader *mockArtifactUploader,
	testCases *mockTestCaseProvider,
	runner *mockValidatorRunner,
) *PrepareJudgingUseCase {
	return NewPrepareJudgingUseCase(sources, downloader, compiler, uploader, testCases, runner)
}

func checkerSource() *JudgingSource {
	return &JudgingSource{Filename: "checker.cpp", FileKey: "problems/abc/checker/checker.cpp", Language: submission.RestoreLanguage("cpp20")}
}

func validatorSource() *JudgingSource {
	return &JudgingSource{Filename: "validator.py", FileKey: "problems/abc/validator/validator.py", Language: submission.RestoreLanguage("python310")}
}

func TestPrepareJudging_NoCheckerNoValidator_ReturnsEmptyOutput(t *testing.T) {
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			t.Fatal("test cases should not be fetched when there's no validator")
			return nil, nil
		},
	}
	uc := newPrepareJudgingUseCase(&mockJudgingSourceProvider{}, &mockSourceCodeDownloader{}, &mockNativeCompiler{}, &mockArtifactUploader{}, testCases, &mockValidatorRunner{})

	out, err := uc.Execute(context.Background(), PrepareJudgingInput{ProblemID: problemID, Slug: "sum-of-two-numbers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.CheckerCompiledKey != "" || out.ValidatorCompiledKey != "" || out.Failure != nil {
		t.Errorf("expected an empty output, got %+v", out)
	}
}

func TestPrepareJudging_CheckerCompiles_UploadsAndReturnsKey(t *testing.T) {
	sources := &mockJudgingSourceProvider{
		getCheckerSourceFn: func(_ context.Context, _ string) (*JudgingSource, error) { return checkerSource(), nil },
	}
	uploader := &mockArtifactUploader{}
	uc := newPrepareJudgingUseCase(sources, &mockSourceCodeDownloader{}, &mockNativeCompiler{}, uploader, &mockTestCaseProvider{}, &mockValidatorRunner{})

	out, err := uc.Execute(context.Background(), PrepareJudgingInput{ProblemID: problemID, Slug: "sum-of-two-numbers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantKey := "problems/sum-of-two-numbers/checker/compiled"
	if out.CheckerCompiledKey != wantKey {
		t.Errorf("CheckerCompiledKey: got %q, want %q", out.CheckerCompiledKey, wantKey)
	}
	if _, ok := uploader.uploaded[wantKey]; !ok {
		t.Errorf("expected the compiled checker to be uploaded at %q", wantKey)
	}
	if out.Failure != nil {
		t.Errorf("expected no failure, got %+v", out.Failure)
	}
}

func TestPrepareJudging_CheckerCompileFails_StopsBeforeValidator(t *testing.T) {
	sources := &mockJudgingSourceProvider{
		getCheckerSourceFn: func(_ context.Context, _ string) (*JudgingSource, error) { return checkerSource(), nil },
		getValidatorSourceFn: func(_ context.Context, _ string) (*JudgingSource, error) {
			t.Fatal("validator source should not be checked when the checker already failed")
			return nil, nil
		},
	}
	compiler := &mockNativeCompiler{
		compileFn: func(_ context.Context, _ CompileArtifactRequest) (CompileArtifactResult, error) {
			return CompileArtifactResult{Success: false, Log: "checker.cpp:3:1: error: expected ';'"}, nil
		},
	}
	uc := newPrepareJudgingUseCase(sources, &mockSourceCodeDownloader{}, compiler, &mockArtifactUploader{}, &mockTestCaseProvider{}, &mockValidatorRunner{})

	out, err := uc.Execute(context.Background(), PrepareJudgingInput{ProblemID: problemID, Slug: "sum-of-two-numbers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != PrepareJudgingCheckerCompileError {
		t.Errorf("Failure: got %+v", out.Failure)
	}
	if out.Failure.FileKey != "problems/abc/checker/checker.cpp" {
		t.Errorf("FileKey: got %q", out.Failure.FileKey)
	}
}

func TestPrepareJudging_ValidatorCompileFails_ReturnsFailure(t *testing.T) {
	sources := &mockJudgingSourceProvider{
		getValidatorSourceFn: func(_ context.Context, _ string) (*JudgingSource, error) { return validatorSource(), nil },
	}
	compiler := &mockNativeCompiler{
		compileFn: func(_ context.Context, _ CompileArtifactRequest) (CompileArtifactResult, error) {
			return CompileArtifactResult{Success: false, Log: "SyntaxError: invalid syntax"}, nil
		},
	}
	uc := newPrepareJudgingUseCase(sources, &mockSourceCodeDownloader{}, compiler, &mockArtifactUploader{}, &mockTestCaseProvider{}, &mockValidatorRunner{})

	out, err := uc.Execute(context.Background(), PrepareJudgingInput{ProblemID: problemID, Slug: "sum-of-two-numbers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != PrepareJudgingValidatorCompileError {
		t.Errorf("Failure: got %+v", out.Failure)
	}
}

func TestPrepareJudging_ValidatorCompiles_AllInputsAccepted(t *testing.T) {
	sources := &mockJudgingSourceProvider{
		getValidatorSourceFn: func(_ context.Context, _ string) (*JudgingSource, error) { return validatorSource(), nil },
	}
	uploader := &mockArtifactUploader{}
	uc := newPrepareJudgingUseCase(sources, &mockSourceCodeDownloader{}, &mockNativeCompiler{}, uploader, &mockTestCaseProvider{}, &mockValidatorRunner{})

	out, err := uc.Execute(context.Background(), PrepareJudgingInput{ProblemID: problemID, Slug: "sum-of-two-numbers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wantKey := "problems/sum-of-two-numbers/validator/compiled"
	if out.ValidatorCompiledKey != wantKey {
		t.Errorf("ValidatorCompiledKey: got %q, want %q", out.ValidatorCompiledKey, wantKey)
	}
	if _, ok := uploader.uploaded[wantKey]; !ok {
		t.Error("expected the compiled validator to be uploaded")
	}
	if out.Failure != nil {
		t.Errorf("expected no failure, got %+v", out.Failure)
	}
}

func TestPrepareJudging_ValidatorRejectsInput_StopsAtFirstRejection(t *testing.T) {
	sources := &mockJudgingSourceProvider{
		getValidatorSourceFn: func(_ context.Context, _ string) (*JudgingSource, error) { return validatorSource(), nil },
	}
	testCases := &mockTestCaseProvider{
		getTestCasesFn: func(_ context.Context, _ string) ([]TestCase, error) {
			return []TestCase{
				{Name: "secret/001", Input: []byte("1")},
				{Name: "secret/002", Input: []byte("2")},
			}, nil
		},
	}
	runCalls := 0
	runner := &mockValidatorRunner{
		runFn: func(_ context.Context, _ ValidatorRunRequest) (ValidatorRunResult, error) {
			runCalls++
			return ValidatorRunResult{Accepted: false, Message: "value exceeds constraint"}, nil
		},
	}
	uc := newPrepareJudgingUseCase(sources, &mockSourceCodeDownloader{}, &mockNativeCompiler{}, &mockArtifactUploader{}, testCases, runner)

	out, err := uc.Execute(context.Background(), PrepareJudgingInput{ProblemID: problemID, Slug: "sum-of-two-numbers"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Failure == nil || out.Failure.Kind != PrepareJudgingInputRejected {
		t.Fatalf("Failure: got %+v", out.Failure)
	}
	if out.Failure.TestCase != "secret/001" {
		t.Errorf("TestCase: got %q, want secret/001", out.Failure.TestCase)
	}
	if out.Failure.Reason != "value exceeds constraint" {
		t.Errorf("Reason: got %q", out.Failure.Reason)
	}
	if runCalls != 1 {
		t.Errorf("expected the runner to stop after the first rejection, got %d calls", runCalls)
	}
}
