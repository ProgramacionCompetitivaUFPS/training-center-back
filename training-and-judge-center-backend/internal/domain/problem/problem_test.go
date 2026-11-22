package problem_test

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func newDraftProblem() *problem.Problem {
	return problem.RestoreProblem(
		testProblemID, "test-slug", "Test Title", nil,
		nil, nil, nil, "DRAFT", "PRIVATE",
		shared.RestoreUserID(testAuthorID),
		[]shared.UserID{},
		nil, nil, nil, nil, nil, nil,
		testNow, testNow,
	)
}

func newPublishedProblem() *problem.Problem {
	return problem.RestoreProblem(
		testProblemID, "test-slug", "Test Title", nil,
		nil, nil, nil, "PUBLISHED", "PUBLIC",
		shared.RestoreUserID(testAuthorID),
		[]shared.UserID{},
		nil, nil, nil, nil, nil, nil,
		testNow, testNow,
	)
}

func TestNewProblem_EmptyID_ReturnsError(t *testing.T) {
	slug := problem.RestoreSlug("test-slug")
	title := problem.RestoreTitle("Test Title")
	stmt := problem.RestoreStatement(nil)
	tags := problem.RestoreTags(nil)

	_, err := problem.NewProblem("", slug, title, stmt, nil, nil, nil, tags, shared.RestoreUserID(testAuthorID), testNow)
	if err == nil {
		t.Error("expected error for empty id, got nil")
	}
}

func TestNewProblem_SetsInitialState(t *testing.T) {
	slug := problem.RestoreSlug("test-slug")
	title := problem.RestoreTitle("Test Title")
	stmt := problem.RestoreStatement(nil)
	tags := problem.RestoreTags(nil)

	p, err := problem.NewProblem("prob-id", slug, title, stmt, nil, nil, nil, tags, shared.RestoreUserID(testAuthorID), testNow)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status().String() != "DRAFT" {
		t.Errorf("status: got %q, want DRAFT", p.Status().String())
	}
	if p.Accessibility().String() != "PRIVATE" {
		t.Errorf("accessibility: got %q, want PRIVATE", p.Accessibility().String())
	}
	if len(p.ModifierIDs()) != 1 || p.ModifierIDs()[0] != shared.RestoreUserID(testAuthorID) {
		t.Errorf("modifierIDs: got %v, want [%s] (the author)", p.ModifierIDs(), testAuthorID)
	}
	if len(p.Solutions()) != 0 {
		t.Errorf("solutions: got %d, want 0", len(p.Solutions()))
	}
	if !p.CreatedAt().Equal(testNow.UTC()) {
		t.Errorf("createdAt: got %v, want %v", p.CreatedAt(), testNow.UTC())
	}
	if !p.UpdatedAt().Equal(testNow.UTC()) {
		t.Errorf("updatedAt: got %v, want %v", p.UpdatedAt(), testNow.UTC())
	}
}

func TestAddModifier_AddsUser(t *testing.T) {
	p := newDraftProblem()
	later := testNow.Add(time.Hour)

	err := p.AddModifier(shared.RestoreUserID(testModifierID), later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.ModifierIDs()) != 1 {
		t.Errorf("modifierIDs: got %d, want 1", len(p.ModifierIDs()))
	}
	if p.ModifierIDs()[0] != shared.RestoreUserID(testModifierID) {
		t.Errorf("modifierIDs[0]: got %q, want %q", p.ModifierIDs()[0], testModifierID)
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestAddModifier_DuplicateUser_ReturnsConflict(t *testing.T) {
	p := newDraftProblem()
	_ = p.AddModifier(shared.RestoreUserID(testModifierID), testNow)

	err := p.AddModifier(shared.RestoreUserID(testModifierID), testNow.Add(time.Hour))
	if err == nil {
		t.Error("expected error for duplicate modifier, got nil")
	}
}

func TestAddModifier_ExceedsMaxModifiers_ReturnsBadRequest(t *testing.T) {
	modifiers := make([]shared.UserID, 20)
	for i := range modifiers {
		modifiers[i] = shared.RestoreUserID(fmt.Sprintf("mod-%02d", i))
	}
	p := problem.RestoreProblem(
		testProblemID, "test-slug", "Test Title", nil,
		nil, nil, nil, "DRAFT", "PRIVATE",
		shared.RestoreUserID(testAuthorID),
		modifiers, nil, nil, nil, nil, nil, nil,
		testNow, testNow,
	)

	err := p.AddModifier(shared.RestoreUserID(testStrangerID), testNow.Add(time.Hour))
	if err == nil {
		t.Error("expected error when exceeding MaxModifiers, got nil")
	}
}

func TestRemoveModifier_RemovesUser(t *testing.T) {
	p := newDraftProblem()
	_ = p.AddModifier(shared.RestoreUserID(testModifierID), testNow)
	later := testNow.Add(time.Hour)

	err := p.RemoveModifier(shared.RestoreUserID(testModifierID), later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.ModifierIDs()) != 0 {
		t.Errorf("modifierIDs: got %d, want 0", len(p.ModifierIDs()))
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestRemoveModifier_NotFound_ReturnsNotFound(t *testing.T) {
	p := newDraftProblem()

	err := p.RemoveModifier(shared.RestoreUserID(testStrangerID), testNow.Add(time.Hour))
	if err == nil {
		t.Error("expected error for absent modifier, got nil")
	}
}

func TestCanBeEditedBy(t *testing.T) {
	modifiers := []shared.UserID{shared.RestoreUserID(testModifierID)}
	p := problem.RestoreProblem(
		testProblemID, "test-slug", "Test Title", nil,
		nil, nil, nil, "DRAFT", "PRIVATE",
		shared.RestoreUserID(testAuthorID),
		modifiers, nil, nil, nil, nil, nil, nil,
		testNow, testNow,
	)

	tests := []struct {
		name    string
		userID  shared.UserID
		isAdmin bool
		want    bool
	}{
		{"author can edit", shared.RestoreUserID(testAuthorID), false, true},
		{"admin can edit", shared.RestoreUserID(testStrangerID), true, true},
		{"modifier can edit", shared.RestoreUserID(testModifierID), false, true},
		{"stranger cannot edit", shared.RestoreUserID(testStrangerID), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.CanBeEditedBy(tt.userID, tt.isAdmin)
			if got != tt.want {
				t.Errorf("CanBeEditedBy: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPublish_AlreadyPublished_ReturnsError(t *testing.T) {
	p := newPublishedProblem()

	err := p.Publish(testNow.Add(time.Hour))
	if err == nil {
		t.Error("expected error for publishing an already published problem, got nil")
	}
}

func TestPublish_Draft_TransitionsToPublished(t *testing.T) {
	p := newDraftProblem()
	later := testNow.Add(time.Hour)

	err := p.Publish(later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status().String() != "PUBLISHED" {
		t.Errorf("status: got %q, want PUBLISHED", p.Status().String())
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestUnpublish_AlreadyDraft_ReturnsError(t *testing.T) {
	p := newDraftProblem()

	err := p.Unpublish(testNow.Add(time.Hour))
	if err == nil {
		t.Error("expected error for unpublishing a draft, got nil")
	}
}

func TestUnpublish_Published_TransitionsToDraft(t *testing.T) {
	p := newPublishedProblem()
	later := testNow.Add(time.Hour)

	err := p.Unpublish(later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Status().String() != "DRAFT" {
		t.Errorf("status: got %q, want DRAFT", p.Status().String())
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestSetTestCases_SetsKeyAndUpdatesTimestamps(t *testing.T) {
	p := newDraftProblem()
	later := testNow.Add(time.Hour)

	p.SetTestCases("cases-key", later)

	if p.TestCasesKey() == nil || *p.TestCasesKey() != "cases-key" {
		t.Errorf("TestCasesKey: got %v, want cases-key", p.TestCasesKey())
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestRemoveTestCases_ClearsKey(t *testing.T) {
	p := newDraftProblem()
	p.SetTestCases("cases-key", testNow)

	p.RemoveTestCases(testNow.Add(time.Hour))

	if p.TestCasesKey() != nil {
		t.Errorf("TestCasesKey: got %q, want nil", *p.TestCasesKey())
	}
}

func TestAddSolution_NewFile_Appends(t *testing.T) {
	p := newDraftProblem()
	sol := problem.RestoreJudgingFile("sol.cpp", "key-1", "cpp20", nil, nil)
	later := testNow.Add(time.Hour)

	replaced := p.AddSolution(sol, later)

	if replaced != nil {
		t.Errorf("AddSolution: expected nil for new file, got non-nil")
	}
	if len(p.Solutions()) != 1 {
		t.Errorf("solutions: got %d, want 1", len(p.Solutions()))
	}
	if p.JudgingUpdatedAt() != nil {
		t.Errorf("JudgingUpdatedAt: got %v, want nil", p.JudgingUpdatedAt())
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestAddSolution_DuplicateFilename_Replaces(t *testing.T) {
	p := newDraftProblem()
	original := problem.RestoreJudgingFile("sol.cpp", "key-1", "cpp20", nil, nil)
	updated := problem.RestoreJudgingFile("sol.cpp", "key-2", "cpp20", nil, nil)
	later := testNow.Add(time.Hour)

	p.AddSolution(original, testNow)
	replaced := p.AddSolution(updated, later)

	if replaced == nil {
		t.Fatal("AddSolution: expected replaced file, got nil")
	}
	if replaced.FileKey() != "key-1" {
		t.Errorf("replaced key: got %q, want key-1", replaced.FileKey())
	}
	if len(p.Solutions()) != 1 {
		t.Errorf("solutions count: got %d, want 1", len(p.Solutions()))
	}
	if p.Solutions()[0].FileKey() != "key-2" {
		t.Errorf("updated key: got %q, want key-2", p.Solutions()[0].FileKey())
	}
	if p.JudgingUpdatedAt() != nil {
		t.Errorf("JudgingUpdatedAt: got %v, want nil", p.JudgingUpdatedAt())
	}
}

func TestRemoveSolution_RemovesMatchingFile(t *testing.T) {
	p := newDraftProblem()
	sol := problem.RestoreJudgingFile("sol.cpp", "key-1", "cpp20", nil, nil)
	p.AddSolution(sol, testNow)
	later := testNow.Add(time.Hour)

	err := p.RemoveSolution("sol.cpp", later)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(p.Solutions()) != 0 {
		t.Errorf("solutions: got %d, want 0", len(p.Solutions()))
	}
	if p.JudgingUpdatedAt() != nil {
		t.Errorf("JudgingUpdatedAt: got %v, want nil", p.JudgingUpdatedAt())
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestRemoveSolution_NotFound_ReturnsNotFound(t *testing.T) {
	p := newDraftProblem()

	err := p.RemoveSolution("nonexistent.cpp", testNow.Add(time.Hour))
	if err == nil {
		t.Error("expected error for absent solution, got nil")
	}
}

func TestUpdateMetadata_NilFields_DoNotChange(t *testing.T) {
	p := newDraftProblem()
	originalTitle := p.Title()

	p.UpdateMetadata(nil, nil, nil, nil, nil, nil, testNow.Add(time.Hour))

	if p.Title() != originalTitle {
		t.Error("title changed unexpectedly when nil was passed")
	}
	if !p.UpdatedAt().After(testNow) {
		t.Errorf("updatedAt was not refreshed: got %v", p.UpdatedAt())
	}
}

func TestUpdateMetadata_SetsProvidedFields(t *testing.T) {
	p := newDraftProblem()
	newTitle := problem.RestoreTitle("Updated Title")

	p.UpdateMetadata(&newTitle, nil, nil, nil, nil, nil, testNow.Add(time.Hour))

	if p.Title().String() != "Updated Title" {
		t.Errorf("title: got %q, want Updated Title", p.Title().String())
	}
}

func TestUpdateAccessibility_SetsAccessibility(t *testing.T) {
	p := newDraftProblem()
	pub := problem.NewAccessibilityPublic()

	p.UpdateAccessibility(pub, testNow.Add(time.Hour))

	if p.Accessibility().String() != "PUBLIC" {
		t.Errorf("accessibility: got %q, want PUBLIC", p.Accessibility().String())
	}
}

func TestSetChecker_SetsCheckerAndUpdatesTimestamps(t *testing.T) {
	p := newDraftProblem()
	checker := problem.RestoreJudgingFile("checker.cpp", "key-c", "cpp20", nil, nil)
	later := testNow.Add(time.Hour)

	p.SetChecker(checker, later)

	if p.Checker() == nil {
		t.Fatal("Checker: got nil, want non-nil")
	}
	if p.Checker().FileKey() != "key-c" {
		t.Errorf("Checker.FileKey(): got %q, want key-c", p.Checker().FileKey())
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestRemoveChecker_ClearsCheckerAndUpdatesTimestamps(t *testing.T) {
	p := newDraftProblem()
	p.SetChecker(problem.RestoreJudgingFile("checker.cpp", "key-c", "cpp20", nil, nil), testNow)
	later := testNow.Add(time.Hour)

	p.RemoveChecker(later)

	if p.Checker() != nil {
		t.Error("Checker: expected nil after removal, got non-nil")
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestSetValidator_SetsValidatorAndUpdatesTimestamps(t *testing.T) {
	p := newDraftProblem()
	validator := problem.RestoreJudgingFile("validator.cpp", "key-v", "cpp20", nil, nil)
	later := testNow.Add(time.Hour)

	p.SetValidator(validator, later)

	if p.Validator() == nil {
		t.Fatal("Validator: got nil, want non-nil")
	}
	if p.Validator().FileKey() != "key-v" {
		t.Errorf("Validator.FileKey(): got %q, want key-v", p.Validator().FileKey())
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestRemoveValidator_ClearsValidatorAndUpdatesTimestamps(t *testing.T) {
	p := newDraftProblem()
	p.SetValidator(problem.RestoreJudgingFile("validator.cpp", "key-v", "cpp20", nil, nil), testNow)
	later := testNow.Add(time.Hour)

	p.RemoveValidator(later)

	if p.Validator() != nil {
		t.Error("Validator: expected nil after removal, got non-nil")
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
	}
	if !p.UpdatedAt().Equal(later.UTC()) {
		t.Errorf("updatedAt not refreshed: got %v, want %v", p.UpdatedAt(), later.UTC())
	}
}

func TestJudgingComponents_UpdateJudgingUpdatedAt(t *testing.T) {
	later := testNow.Add(time.Hour)
	file := problem.RestoreJudgingFile("c.cpp", "key", "cpp20", nil, nil)

	tests := []struct {
		name string
		do   func(p *problem.Problem)
	}{
		{"set test cases", func(p *problem.Problem) { p.SetTestCases("cases", later) }},
		{"remove test cases", func(p *problem.Problem) { p.RemoveTestCases(later) }},
		{"set checker", func(p *problem.Problem) { p.SetChecker(file, later) }},
		{"remove checker", func(p *problem.Problem) { p.RemoveChecker(later) }},
		{"set validator", func(p *problem.Problem) { p.SetValidator(file, later) }},
		{"remove validator", func(p *problem.Problem) { p.RemoveValidator(later) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := newDraftProblem()
			tt.do(p)
			if p.JudgingUpdatedAt() == nil || !p.JudgingUpdatedAt().Equal(later.UTC()) {
				t.Errorf("JudgingUpdatedAt: got %v, want %v", p.JudgingUpdatedAt(), later.UTC())
			}
		})
	}
}

func fullProblem() *problem.Problem {
	checker := problem.RestoreJudgingFile("checker.cpp", "problems/p/checker.cpp", "cpp20", nil, nil)
	return problem.RestoreProblem(
		testProblemID, "sum", "Sum", strPtr("Add two numbers"),
		intPtr(1000), intPtr(256), []string{"dp"}, "DRAFT", "PRIVATE",
		shared.RestoreUserID(testAuthorID),
		[]shared.UserID{shared.RestoreUserID(testAuthorID)},
		[]problem.LanguageOverride{problem.RestoreLanguageOverride("cpp20", intPtr(500), nil)},
		strPtr("problems/p/tests.zip"), nil, &checker, nil, nil,
		testNow, testNow,
	)
}

func TestNewProblem_StoresTheGivenFields(t *testing.T) {
	timeLimit := problem.RestoreTimeLimit(1000)
	memoryLimit := problem.RestoreMemoryLimit(256)
	overrides := []problem.LanguageOverride{problem.RestoreLanguageOverride("cpp20", intPtr(500), nil)}
	localNow := time.Date(2024, 1, 15, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	p, err := problem.NewProblem(
		"prob-id", problem.RestoreSlug("sum"), problem.RestoreTitle("Sum"),
		problem.RestoreStatement(strPtr("Add two numbers")), &timeLimit, &memoryLimit, overrides,
		problem.RestoreTags([]string{"dp"}), shared.RestoreUserID(testAuthorID), localNow,
	)

	if err != nil {
		t.Fatalf("NewProblem: %v", err)
	}
	if p.ID() != "prob-id" || p.Slug().String() != "sum" || p.Title().String() != "Sum" {
		t.Errorf("id/slug/title = %s/%s/%s, want prob-id/sum/Sum", p.ID(), p.Slug(), p.Title())
	}
	if p.Statement().Value() == nil || *p.Statement().Value() != "Add two numbers" {
		t.Errorf("Statement() = %v, want Add two numbers", p.Statement().Value())
	}
	if p.TimeLimit() == nil || p.TimeLimit().Milliseconds() != 1000 || p.MemoryLimit() == nil || p.MemoryLimit().Megabytes() != 256 {
		t.Errorf("limits = %v/%v, want 1000 ms and 256 MB", p.TimeLimit(), p.MemoryLimit())
	}
	if len(p.LangOverrides()) != 1 || p.LangOverrides()[0].Language() != "cpp20" {
		t.Errorf("LangOverrides() = %+v, want the cpp20 override", p.LangOverrides())
	}
	if !reflect.DeepEqual(p.Tags().Values(), []string{"dp"}) {
		t.Errorf("Tags() = %v, want [dp]", p.Tags().Values())
	}
	if p.AuthorID().Value() != testAuthorID || len(p.ModifierIDs()) != 1 || p.ModifierIDs()[0].Value() != testAuthorID {
		t.Errorf("author/modifiers = %s/%v, want the author as the only modifier", p.AuthorID().Value(), p.ModifierIDs())
	}
	if want := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC); !p.CreatedAt().Equal(want) || p.CreatedAt().Location() != time.UTC || !p.UpdatedAt().Equal(want) {
		t.Errorf("createdAt/updatedAt = %v/%v, want %v in UTC", p.CreatedAt(), p.UpdatedAt(), want)
	}
}

func TestRestoreProblem_KeepsStoredValuesAndNeverLeavesNilCollections(t *testing.T) {
	p := problem.RestoreProblem(
		testProblemID, "sum", "Sum", nil,
		intPtr(1000), intPtr(256), nil, "PUBLISHED", "PUBLIC",
		shared.RestoreUserID(testAuthorID),
		nil, nil, nil, nil, nil, nil, nil,
		testNow, testNow,
	)

	if p.TimeLimit() == nil || p.TimeLimit().Milliseconds() != 1000 || p.MemoryLimit() == nil || p.MemoryLimit().Megabytes() != 256 {
		t.Errorf("limits = %v/%v, want 1000 ms and 256 MB", p.TimeLimit(), p.MemoryLimit())
	}
	if p.Status().String() != "PUBLISHED" || p.Accessibility().String() != "PUBLIC" {
		t.Errorf("status/accessibility = %s/%s, want PUBLISHED/PUBLIC", p.Status(), p.Accessibility())
	}
	if p.ModifierIDs() == nil || p.LangOverrides() == nil || p.Solutions() == nil {
		t.Errorf("collections must never be nil: modifiers %v, overrides %v, solutions %v", p.ModifierIDs(), p.LangOverrides(), p.Solutions())
	}
}

func TestUpdateMetadata_AppliesEveryProvidedField(t *testing.T) {
	p := fullProblem()
	title := problem.RestoreTitle("New title")
	statement := problem.RestoreStatement(strPtr("New statement"))
	timeLimit := problem.RestoreTimeLimit(2000)
	memoryLimit := problem.RestoreMemoryLimit(512)
	overrides := []problem.LanguageOverride{problem.RestoreLanguageOverride("python310", intPtr(900), intPtr(128))}
	tags := problem.RestoreTags([]string{"graphs", "dp"})
	now := time.Date(2024, 1, 15, 14, 0, 0, 0, time.FixedZone("UTC+2", 2*3600))

	p.UpdateMetadata(&title, &statement, &timeLimit, &memoryLimit, overrides, &tags, now)

	if p.Title().String() != "New title" {
		t.Errorf("Title() = %q, want New title", p.Title())
	}
	if p.Statement().Value() == nil || *p.Statement().Value() != "New statement" {
		t.Errorf("Statement() = %v, want New statement", p.Statement().Value())
	}
	if p.TimeLimit().Milliseconds() != 2000 || p.MemoryLimit().Megabytes() != 512 {
		t.Errorf("limits = %d/%d, want 2000/512", p.TimeLimit().Milliseconds(), p.MemoryLimit().Megabytes())
	}
	if len(p.LangOverrides()) != 1 || p.LangOverrides()[0].Language() != "python310" {
		t.Errorf("LangOverrides() = %+v, want the python310 override", p.LangOverrides())
	}
	if !reflect.DeepEqual(p.Tags().Values(), []string{"graphs", "dp"}) {
		t.Errorf("Tags() = %v, want [graphs dp]", p.Tags().Values())
	}
	if want := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC); !p.UpdatedAt().Equal(want) || p.UpdatedAt().Location() != time.UTC {
		t.Errorf("UpdatedAt() = %v, want %v in UTC", p.UpdatedAt(), want)
	}
}

func TestUpdateMetadata_NilFields_KeepEverythingButTheUpdateInstant(t *testing.T) {
	p := fullProblem()
	later := testNow.Add(time.Hour)

	p.UpdateMetadata(nil, nil, nil, nil, nil, nil, later)

	if p.Title().String() != "Sum" || *p.Statement().Value() != "Add two numbers" {
		t.Errorf("title/statement = %q/%q, want Sum/Add two numbers", p.Title(), *p.Statement().Value())
	}
	if p.TimeLimit().Milliseconds() != 1000 || p.MemoryLimit().Megabytes() != 256 {
		t.Errorf("limits = %d/%d, want 1000/256", p.TimeLimit().Milliseconds(), p.MemoryLimit().Megabytes())
	}
	if len(p.LangOverrides()) != 1 || !reflect.DeepEqual(p.Tags().Values(), []string{"dp"}) {
		t.Errorf("overrides/tags = %+v/%v, want them untouched", p.LangOverrides(), p.Tags().Values())
	}
	if !p.UpdatedAt().Equal(later) {
		t.Errorf("UpdatedAt() = %v, want %v", p.UpdatedAt(), later)
	}
}

func TestUpdateMetadata_EmptyOverrides_ClearThem(t *testing.T) {
	p := fullProblem()

	p.UpdateMetadata(nil, nil, nil, nil, []problem.LanguageOverride{}, nil, testNow.Add(time.Hour))

	if len(p.LangOverrides()) != 0 {
		t.Errorf("LangOverrides() = %+v, want none", p.LangOverrides())
	}
}

func TestUpdateMetadata_StoresItsOwnCopyOfTheLimits(t *testing.T) {
	p := fullProblem()
	timeLimit := problem.RestoreTimeLimit(2000)
	memoryLimit := problem.RestoreMemoryLimit(512)

	p.UpdateMetadata(nil, nil, &timeLimit, &memoryLimit, nil, nil, testNow.Add(time.Hour))
	timeLimit = problem.RestoreTimeLimit(9999)
	memoryLimit = problem.RestoreMemoryLimit(9999)

	if p.TimeLimit().Milliseconds() != 2000 || p.MemoryLimit().Megabytes() != 512 {
		t.Errorf("limits = %d/%d, want 2000/512: the caller's variables must not alias the stored ones", p.TimeLimit().Milliseconds(), p.MemoryLimit().Megabytes())
	}
}

func TestRemoveModifier_KeepsTheOthersInOrder(t *testing.T) {
	p := problem.RestoreProblem(
		testProblemID, "test-slug", "Test Title", nil,
		nil, nil, nil, "DRAFT", "PRIVATE",
		shared.RestoreUserID(testAuthorID),
		[]shared.UserID{shared.RestoreUserID(testModifierID), shared.RestoreUserID(testStrangerID)},
		nil, nil, nil, nil, nil, nil,
		testNow, testNow,
	)

	err := p.RemoveModifier(shared.RestoreUserID(testModifierID), testNow.Add(time.Hour))

	if err != nil {
		t.Fatalf("RemoveModifier: %v", err)
	}
	if got := p.ModifierIDs(); len(got) != 1 || got[0].Value() != testStrangerID {
		t.Errorf("ModifierIDs() = %v, want only %s", got, testStrangerID)
	}
}
