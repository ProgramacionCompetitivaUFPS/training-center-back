package submission

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func ctx() context.Context { return context.Background() }

func TestSubmitSolution_Success(t *testing.T) {
	uc := newUseCase(publicProblem(), cleanRepo(), nil)
	out, err := uc.Execute(ctx(), validInput())
	require.NoError(t, err)
	assert.NotEmpty(t, out.ID)
	assert.Equal(t, "PENDING", out.Status)
	assert.Equal(t, testProblemSlug, out.ProblemSlug)
	assert.Equal(t, "cpp20", out.Language)
	assert.Equal(t, "g++", out.Compiler)
	assert.NotEmpty(t, out.FileHash)
	assert.Greater(t, out.FileSize, 0)
}

func TestSubmitSolution_FileTooLarge(t *testing.T) {
	uc := newUseCase(publicProblem(), cleanRepo(), nil)
	in := validInput()
	in.FileData = make([]byte, maxFileSize+1)

	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, domainSubmission.ErrCodeFileTooLarge, ae.Code)
}

func TestSubmitSolution_InvalidLanguage(t *testing.T) {
	uc := newUseCase(publicProblem(), cleanRepo(), nil)
	in := validInput()
	in.Language = "rust"

	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindValidation, ae.Kind)
}

func TestSubmitSolution_CompilerMismatch(t *testing.T) {
	uc := newUseCase(publicProblem(), cleanRepo(), nil)
	in := validInput()
	in.Compiler = "javac" // wrong for cpp20

	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, domainSubmission.ErrCodeCompilerMismatch, ae.Code)
}

func TestSubmitSolution_ExtensionMismatch(t *testing.T) {
	uc := newUseCase(publicProblem(), cleanRepo(), nil)
	in := validInput()
	in.FileName = "solution.py" // .py doesn't match cpp20

	_, err := uc.Execute(ctx(), in)
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, domainSubmission.ErrCodeCompilerMismatch, ae.Code)
}

func TestSubmitSolution_ProblemNotPublished(t *testing.T) {
	uc := newUseCase(draftProblem(), cleanRepo(), nil)
	_, err := uc.Execute(ctx(), validInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, domainSubmission.ErrCodeProblemNotPublished, ae.Code)
}

func TestSubmitSolution_PrivateProblem_NonModifier_Returns403(t *testing.T) {
	// caller is testUserID; no modifiers → forbidden
	uc := newUseCase(privateProblem(), cleanRepo(), nil)
	_, err := uc.Execute(ctx(), validInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, domainSubmission.ErrCodeProblemNotAccessible, ae.Code)
}

func TestSubmitSolution_PrivateProblem_Modifier_Succeeds(t *testing.T) {
	uc := newUseCase(privateProblem(testUserID), cleanRepo(), nil)
	out, err := uc.Execute(ctx(), validInput())
	require.NoError(t, err)
	assert.Equal(t, "PENDING", out.Status)
}

func TestSubmitSolution_DuplicateFile(t *testing.T) {
	repo := &mockSubmissionRepo{
		existsDupFn: func(_, _, _ string) (bool, error) { return true, nil },
	}
	uc := newUseCase(publicProblem(), repo, nil)
	_, err := uc.Execute(ctx(), validInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindConflict, ae.Kind)
	assert.Equal(t, domainSubmission.ErrCodeDuplicateSubmission, ae.Code)
}

func TestSubmitSolution_RateLimitExceeded(t *testing.T) {
	justNow := testNow.Add(-500 * time.Millisecond) // 0.5s ago < 1s limit
	repo := &mockSubmissionRepo{
		lastFn: func(_, _ string) (*domainSubmission.Submission, error) {
			return recentSubmission(justNow), nil
		},
	}
	uc := newUseCase(publicProblem(), repo, nil)
	_, err := uc.Execute(ctx(), validInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindTooManyRequests, ae.Kind)
	assert.Equal(t, domainSubmission.ErrCodeRateLimitExceeded, ae.Code)
}

func TestSubmitSolution_RateLimit_OldEnough_Succeeds(t *testing.T) {
	twoSecondsAgo := testNow.Add(-2 * time.Second) // 2s ago > 1s limit
	repo := &mockSubmissionRepo{
		lastFn: func(_, _ string) (*domainSubmission.Submission, error) {
			return recentSubmission(twoSecondsAgo), nil
		},
	}
	uc := newUseCase(publicProblem(), repo, nil)
	out, err := uc.Execute(ctx(), validInput())
	require.NoError(t, err)
	assert.Equal(t, "PENDING", out.Status)
}

func TestSubmitSolution_StorageFails_Returns500(t *testing.T) {
	storage := &mockSourceStorage{uploadErr: apperror.NewInternal()}
	uc := newUseCase(publicProblem(), cleanRepo(), storage)
	_, err := uc.Execute(ctx(), validInput())
	require.Error(t, err)
	ae := err.(*apperror.AppError)
	assert.Equal(t, apperror.KindInternal, ae.Kind)
}

func TestSubmitSolution_AllLanguages(t *testing.T) {
	tests := []struct {
		lang     string
		compiler string
		filename string
	}{
		{"cpp20", "g++", "sol.cpp"},
		{"java17", "javac", "Main.java"},
		{"python310", "py", "solution.py"},
	}
	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			uc := newUseCase(publicProblem(), cleanRepo(), nil)
			in := validInput()
			in.Language = tt.lang
			in.Compiler = tt.compiler
			in.FileName = tt.filename
			out, err := uc.Execute(ctx(), in)
			require.NoError(t, err)
			assert.Equal(t, tt.lang, out.Language)
			assert.Equal(t, tt.compiler, out.Compiler)
		})
	}
}
