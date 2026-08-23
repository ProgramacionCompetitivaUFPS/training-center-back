package problem

import (
	"context"
	"errors"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// TestUploadProblemFiles_ActiveValidation_ReturnsConflict only checks that
// the guard runs before any file-type-specific logic — zipParser is nil and
// never reached, since the conflict short-circuits before the file-type
// switch. The rest of this use case's behavior has no test coverage yet
// (pre-existing gap, out of scope for closing this race condition).
func TestUploadProblemFiles_ActiveValidation_ReturnsConflict(t *testing.T) {
	repo := repoWith(newDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}
	uc := NewUploadProblemFilesUseCase(repo, validationRepo, &mockFileStorage{}, nil, newDefaultSettings())

	_, err := uc.Execute(context.Background(), UploadProblemFilesInput{
		Slug:        testSlug,
		FileType:    FileTypeSolution,
		FileName:    "sol.cpp",
		FileData:    []byte("int main(){}"),
		CurrentUser: asCoach(authorID),
	})
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected ErrCodeValidationInProgress, got %v", err)
	}
}

// A verifier is stored under the fixed name the judge writes it as inside the
// sandbox, so the bucket can be walked without the database. The name the
// problem setter chose survives in the record, for display.
func TestUploadProblemFiles_VerifierIsStoredUnderItsFixedName(t *testing.T) {
	tests := []struct {
		name       string
		fileType   string
		uploadedAs string
		wantKey    string
		stored     func(*domainProblem.Problem) *domainProblem.JudgingFile
	}{
		{
			name:       "checker",
			fileType:   FileTypeChecker,
			uploadedAs: "MiChecker_v2.cpp",
			wantKey:    "problems/" + testSlug + "/checker/Checker.cpp",
			stored:     func(p *domainProblem.Problem) *domainProblem.JudgingFile { return p.Checker() },
		},
		{
			name:       "validator",
			fileType:   FileTypeValidator,
			uploadedAs: "gen_validator.cpp",
			wantKey:    "problems/" + testSlug + "/validator/Validator.cpp",
			stored:     func(p *domainProblem.Problem) *domainProblem.JudgingFile { return p.Validator() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotKey string
			storage := &mockFileStorage{
				uploadFileFn: func(_ context.Context, path string, _ []byte) error {
					gotKey = path
					return nil
				},
			}
			p := newDraftProblem()
			uc := NewUploadProblemFilesUseCase(repoWith(p), &mockValidationRepository{}, storage, nil, newDefaultSettings())

			_, err := uc.Execute(context.Background(), UploadProblemFilesInput{
				Slug:        testSlug,
				FileType:    tt.fileType,
				FileName:    tt.uploadedAs,
				FileData:    []byte("int main(){}"),
				CurrentUser: asCoach(authorID),
			})

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotKey != tt.wantKey {
				t.Errorf("storage key: got %q, want %q", gotKey, tt.wantKey)
			}
			saved := tt.stored(p)
			if saved == nil {
				t.Fatal("nothing was recorded on the problem")
			}
			if saved.FileKey() != tt.wantKey {
				t.Errorf("recorded key: got %q, want %q", saved.FileKey(), tt.wantKey)
			}
			if saved.Filename() != tt.uploadedAs {
				t.Errorf("recorded filename: got %q, want the uploaded %q", saved.Filename(), tt.uploadedAs)
			}
		})
	}
}

// The judge rebuilds the ZIP path from the prefix the problem stores, so the two
// have to agree: if they drift, no judging of that problem finds its test cases.
func TestUploadProblemFiles_TestCasesZipSitsInsideTheStoredPrefix(t *testing.T) {
	var uploaded []string
	storage := &mockFileStorage{
		uploadFileFn: func(_ context.Context, path string, _ []byte) error {
			uploaded = append(uploaded, path)
			return nil
		},
	}
	parser := &mockZipParser{
		parseTestCasesZipFn: func(_ context.Context, _ []byte) ([]ParsedFile, error) {
			return []ParsedFile{{Path: "data/sample/01.in", Content: []byte("1 2")}}, nil
		},
	}
	p := newDraftProblem()
	uc := NewUploadProblemFilesUseCase(repoWith(p), &mockValidationRepository{}, storage, parser, newDefaultSettings())

	_, err := uc.Execute(context.Background(), UploadProblemFilesInput{
		Slug:        testSlug,
		FileType:    FileTypeTestCases,
		FileName:    "tc.zip",
		FileData:    []byte("PK"),
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	prefix := p.TestCasesKey()
	if prefix == nil {
		t.Fatal("no test cases key was recorded on the problem")
	}
	want := *prefix + "/testcases.zip"
	for _, k := range uploaded {
		if k == want {
			return
		}
	}
	t.Errorf("ZIP key %q not among the uploaded keys %v", want, uploaded)
}
