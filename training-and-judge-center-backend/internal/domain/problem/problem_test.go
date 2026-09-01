package problem_test

import (
	"fmt"
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
	if len(p.ModifierIDs()) != 0 {
		t.Errorf("modifierIDs: got %d, want 0", len(p.ModifierIDs()))
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
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
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
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
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
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt: got nil, want non-nil")
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
