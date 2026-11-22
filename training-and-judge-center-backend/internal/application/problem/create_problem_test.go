package problem

import (
	"context"
	"reflect"
	"strings"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func validCreateInput() CreateProblemInput {
	return CreateProblemInput{
		Slug:        testSlug,
		Title:       "Test Problem",
		CurrentUser: asCoach(authorID),
	}
}

func TestCreateProblem_Success_Coach(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	result, err := uc.Execute(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Problem.Slug != testSlug {
		t.Errorf("expected slug %q, got %q", testSlug, result.Problem.Slug)
	}
	if result.Problem.Status != "DRAFT" {
		t.Errorf("expected DRAFT status, got %q", result.Problem.Status)
	}
	if result.Problem.AuthorID != authorID {
		t.Errorf("expected authorID %q, got %q", authorID, result.Problem.AuthorID)
	}
}

func TestCreateProblem_Success_Admin(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.CurrentUser = asAdmin(authorID)

	_, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("admin should be able to create a problem, got error: %v", err)
	}
}

func TestCreateProblem_Forbidden_Contestant(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.CurrentUser = asContestant(strangerID)

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %q", appErr.Code)
	}
}

func TestCreateProblem_InvalidSlug(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.Slug = "ab" // too short (min 3 chars)

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_InvalidTitle(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.Title = "" // empty title

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_InvalidTag(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	input := validCreateInput()
	input.Tags = []string{"not-a-valid-tag"}

	_, err := uc.Execute(context.Background(), input)
	if err == nil {
		t.Fatal("expected validation error for unknown tag, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_SlugAlreadyExists(t *testing.T) {
	repo := &mockProblemRepository{
		saveFn: func(_ context.Context, _ *domainProblem.Problem) error {
			return apperror.NewConflict(domainProblem.ErrCodeSlugAlreadyExists, "slug already in use")
		},
	}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeSlugAlreadyExists {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeSlugAlreadyExists, appErr.Code)
	}
}

func TestCreateProblem_RepositoryError(t *testing.T) {
	repo := &mockProblemRepository{
		saveFn: func(_ context.Context, _ *domainProblem.Problem) error {
			return apperror.NewInternal()
		},
	}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	_, err := uc.Execute(context.Background(), validCreateInput())
	if err == nil {
		t.Fatal("expected internal error, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeInternalError {
		t.Errorf("expected INTERNAL_ERROR, got %q", appErr.Code)
	}
}

func TestCreateProblem_SavesEveryField(t *testing.T) {
	repo := &mockProblemRepository{}
	uc := NewCreateProblemUseCase(repo, newDefaultSettings())

	statement := "A statement"
	timeLimit := 2000
	memoryLimit := 256
	overrideTime := 1500
	input := validCreateInput()
	input.Statement = &statement
	input.TimeLimit = &timeLimit
	input.MemoryLimit = &memoryLimit
	input.LangOverrides = []LanguageOverrideInput{{Language: "cpp", TimeLimit: &overrideTime}}
	input.Tags = []string{"dp", "graphs"}

	result, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := result.Problem
	if got.Statement == nil || *got.Statement != statement {
		t.Errorf("statement: got %v, want %q", got.Statement, statement)
	}
	if got.TimeLimit == nil || *got.TimeLimit != timeLimit {
		t.Errorf("time limit: got %v, want %d", got.TimeLimit, timeLimit)
	}
	if got.MemoryLimit == nil || *got.MemoryLimit != memoryLimit {
		t.Errorf("memory limit: got %v, want %d", got.MemoryLimit, memoryLimit)
	}
	if len(got.LangOverrides) != 1 || got.LangOverrides[0].Language != "cpp" ||
		got.LangOverrides[0].TimeLimit == nil || *got.LangOverrides[0].TimeLimit != overrideTime {
		t.Errorf("language overrides: got %+v", got.LangOverrides)
	}
	if !reflect.DeepEqual(got.Tags, []string{"dp", "graphs"}) {
		t.Errorf("tags: got %v", got.Tags)
	}
}

func TestCreateProblem_RejectsInvalidFields(t *testing.T) {
	zero := 0
	tooHigh := 999999
	tooLong := strings.Repeat("a", 150001)

	tests := []struct {
		name   string
		mutate func(in *CreateProblemInput)
		want   []string
	}{
		{"statement too long", func(in *CreateProblemInput) { in.Statement = &tooLong }, []string{"statement: Statement must not exceed 150,000 characters"}},
		{"time limit above the global maximum", func(in *CreateProblemInput) { in.TimeLimit = &tooHigh }, []string{"timeLimit: Exceeds global maximum time limit"}},
		{"memory limit above the global maximum", func(in *CreateProblemInput) { in.MemoryLimit = &tooHigh }, []string{"memoryLimit: Exceeds global maximum memory limit"}},
		{"override without language", func(in *CreateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: ""}}
		}, []string{"languageOverrides: Language must not be empty"}},
		{"override with unsupported language", func(in *CreateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "ruby"}}
		}, []string{"languageOverrides.language: Unsupported language: ruby"}},
		{"duplicated override", func(in *CreateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "cpp"}, {Language: "cpp"}}
		}, []string{"languageOverrides.language: Duplicate language override: cpp"}},
		{"override with a non-positive time limit", func(in *CreateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "cpp", TimeLimit: &zero}}
		}, []string{"languageOverrides.timeLimit: Time limit must be positive"}},
		{"override above the global time limit", func(in *CreateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "cpp", TimeLimit: &tooHigh}}
		}, []string{"languageOverrides.timeLimit: Exceeds global maximum time limit for cpp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := NewCreateProblemUseCase(&mockProblemRepository{}, newDefaultSettings())
			input := validCreateInput()
			tt.mutate(&input)

			_, err := uc.Execute(context.Background(), input)

			if err == nil {
				t.Fatal("expected a validation error, got nil")
			}
			if got := validationDetails(t, err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCreateProblem_ReportsEveryFieldErrorInOrder(t *testing.T) {
	uc := NewCreateProblemUseCase(&mockProblemRepository{}, newDefaultSettings())
	tooHigh := 999999
	tooLong := strings.Repeat("a", 150001)

	input := validCreateInput()
	input.Title = "   "
	input.Statement = &tooLong
	input.TimeLimit = &tooHigh
	input.MemoryLimit = &tooHigh
	input.LangOverrides = []LanguageOverrideInput{{Language: "ruby"}}
	input.Tags = []string{"nope"}

	_, err := uc.Execute(context.Background(), input)

	want := []string{
		"title: Title is required",
		"statement: Statement must not exceed 150,000 characters",
		"timeLimit: Exceeds global maximum time limit",
		"memoryLimit: Exceeds global maximum memory limit",
		"languageOverrides.language: Unsupported language: ruby",
		"tags: Invalid tag: nope",
	}
	if got := validationDetails(t, err); !reflect.DeepEqual(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}
