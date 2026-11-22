package problem

import (
	"context"
	"reflect"
	"strings"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func TestUpdateProblem_Success_Author(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "Updated Title"
	result, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("author should be able to update own draft, got: %v", err)
	}
	if result.Problem.Title != newTitle {
		t.Errorf("expected title %q, got %q", newTitle, result.Problem.Title)
	}
}

func TestUpdateProblem_Success_Modifier(t *testing.T) {
	repo := repoWith(newDraftProblemWithModifier())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "Modifier's Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(modifierID),
	})
	if err != nil {
		t.Fatalf("modifier should be able to update draft, got: %v", err)
	}
}

func TestUpdateProblem_Success_Admin(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "Admin Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asAdmin(strangerID),
	})
	if err != nil {
		t.Fatalf("admin should be able to update any draft, got: %v", err)
	}
}

func TestUpdateProblem_Forbidden_Stranger(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "Stolen Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asContestant(strangerID),
	})
	if err == nil {
		t.Fatal("stranger should not update the problem, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeInsufficientPermissions {
		t.Errorf("expected INSUFFICIENT_PERMISSIONS, got %q", appErr.Code)
	}
}

func TestUpdateProblem_CannotUpdatePublished(t *testing.T) {
	repo := repoWith(newPublishedProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "New Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("should not update a published problem, got nil error")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeProblemIsPublished {
		t.Errorf("expected %q, got %q", ErrCodeProblemIsPublished, appErr.Code)
	}
}

func TestUpdateProblem_InvalidTimeLimit(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	tooHigh := 999999 // exceeds global max of 10000
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		TimeLimit:   &tooHigh,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected validation error for out-of-range time limit, got nil")
	}

	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %q", appErr.Code)
	}
}

func TestUpdateProblem_ActiveValidation_ReturnsConflict(t *testing.T) {
	repo := repoWith(newDraftProblem())
	validationRepo := &mockValidationRepository{
		findLatestByProblemIDFn: func(_ context.Context, _ string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}
	uc := NewUpdateProblemUseCase(repo, validationRepo, newDefaultSettings())

	newTitle := "New Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("expected %q, got %q", domainProblem.ErrCodeValidationInProgress, appErr.Code)
	}
}

func TestUpdateProblem_NotFound(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
		},
	}
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
}

func TestUpdateProblem_RepositoryError(t *testing.T) {
	repo := &mockProblemRepository{
		findBySlugFn: func(_ context.Context, _ domainProblem.Slug) (*domainProblem.Problem, error) {
			return newDraftProblem(), nil
		},
		saveFn: func(_ context.Context, _ *domainProblem.Problem) error {
			return apperror.NewInternal()
		},
	}
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	newTitle := "Title"
	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		Title:       &newTitle,
		CurrentUser: asCoach(authorID),
	})
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

func TestUpdateProblem_UpdatesEveryField(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	title := "New Title"
	statement := "New statement"
	timeLimit := 2000
	memoryLimit := 256
	accessibility := "PUBLIC"
	overrideTime := 1500

	result, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:          testSlug,
		Title:         &title,
		Statement:     &statement,
		TimeLimit:     &timeLimit,
		MemoryLimit:   &memoryLimit,
		LangOverrides: []LanguageOverrideInput{{Language: "cpp", TimeLimit: &overrideTime}},
		Tags:          []string{"dp", "graphs"},
		Accessibility: &accessibility,
		CurrentUser:   asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := result.Problem
	if got.Title != title {
		t.Errorf("title: got %q, want %q", got.Title, title)
	}
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
	if got.Accessibility != "PUBLIC" {
		t.Errorf("accessibility: got %q, want PUBLIC", got.Accessibility)
	}
}

func TestUpdateProblem_WithoutFieldsKeepsTheProblemAsIs(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	result, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:        testSlug,
		CurrentUser: asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Problem.Title != "Test Problem" {
		t.Errorf("title changed to %q", result.Problem.Title)
	}
	if result.Problem.Accessibility != "PRIVATE" {
		t.Errorf("accessibility changed to %q", result.Problem.Accessibility)
	}
}

func TestUpdateProblem_RejectsInvalidFields(t *testing.T) {
	zero := 0
	tooHigh := 999999
	blank := "   "
	secret := "SECRET"
	tooLong := strings.Repeat("a", 150001)

	tests := []struct {
		name   string
		mutate func(in *UpdateProblemInput)
		want   []string
	}{
		{"blank title", func(in *UpdateProblemInput) { in.Title = &blank }, []string{"title: Title is required"}},
		{"memory limit above the global maximum", func(in *UpdateProblemInput) { in.MemoryLimit = &tooHigh }, []string{"memoryLimit: Exceeds global maximum memory limit"}},
		{"unknown accessibility", func(in *UpdateProblemInput) { in.Accessibility = &secret }, []string{"accessibility: Invalid accessibility. Must be PRIVATE or PUBLIC"}},
		{"statement too long", func(in *UpdateProblemInput) { in.Statement = &tooLong }, []string{"statement: Statement must not exceed 150,000 characters"}},
		{"unknown tag", func(in *UpdateProblemInput) { in.Tags = []string{"nope"} }, []string{"tags: Invalid tag: nope"}},
		{"override without language", func(in *UpdateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: ""}}
		}, []string{"languageOverrides: Language must not be empty"}},
		{"override with unsupported language", func(in *UpdateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "ruby"}}
		}, []string{"languageOverrides.language: Unsupported language: ruby"}},
		{"duplicated override", func(in *UpdateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "cpp"}, {Language: "cpp"}}
		}, []string{"languageOverrides.language: Duplicate language override: cpp"}},
		{"override with a non-positive time limit", func(in *UpdateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "cpp", TimeLimit: &zero}}
		}, []string{"languageOverrides.timeLimit: Time limit must be positive"}},
		{"override above the global time limit", func(in *UpdateProblemInput) {
			in.LangOverrides = []LanguageOverrideInput{{Language: "cpp", TimeLimit: &tooHigh}}
		}, []string{"languageOverrides.timeLimit: Exceeds global maximum time limit for cpp"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := repoWith(newDraftProblem())
			uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())
			input := UpdateProblemInput{Slug: testSlug, CurrentUser: asCoach(authorID)}
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

func TestUpdateProblem_ReportsEveryFieldErrorInOrder(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())

	blank := "   "
	tooHigh := 999999
	secret := "SECRET"
	tooLong := strings.Repeat("a", 150001)

	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:          testSlug,
		Title:         &blank,
		TimeLimit:     &tooHigh,
		MemoryLimit:   &tooHigh,
		LangOverrides: []LanguageOverrideInput{{Language: "ruby"}},
		Tags:          []string{"nope"},
		Accessibility: &secret,
		Statement:     &tooLong,
		CurrentUser:   asCoach(authorID),
	})

	want := []string{
		"title: Title is required",
		"timeLimit: Exceeds global maximum time limit",
		"memoryLimit: Exceeds global maximum memory limit",
		"languageOverrides.language: Unsupported language: ruby",
		"tags: Invalid tag: nope",
		"accessibility: Invalid accessibility. Must be PRIVATE or PUBLIC",
		"statement: Statement must not exceed 150,000 characters",
	}
	if got := validationDetails(t, err); !reflect.DeepEqual(got, want) {
		t.Errorf("got  %v\nwant %v", got, want)
	}
}

func TestUpdateProblem_AnEmptyOverrideListDoesNotClearTheExistingOnes(t *testing.T) {
	repo := repoWith(newDraftProblem())
	uc := NewUpdateProblemUseCase(repo, &mockValidationRepository{}, newDefaultSettings())
	overrideTime := 1500

	_, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:          testSlug,
		LangOverrides: []LanguageOverrideInput{{Language: "cpp", TimeLimit: &overrideTime}},
		CurrentUser:   asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("setting the override failed: %v", err)
	}

	result, err := uc.Execute(context.Background(), UpdateProblemInput{
		Slug:          testSlug,
		LangOverrides: []LanguageOverrideInput{},
		CurrentUser:   asCoach(authorID),
	})
	if err != nil {
		t.Fatalf("sending an empty list failed: %v", err)
	}
	if len(result.Problem.LangOverrides) != 1 {
		t.Errorf("an empty list cleared the overrides, got %+v", result.Problem.LangOverrides)
	}
}
