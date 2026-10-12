package problem

import (
	"context"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

const stagedTestCasesKey = "problems/" + testSlug + "/testcases/staged"

func newJudgingUpdateTicket(testCases *string, checker *domainProblem.JudgingFile) *domainProblem.ProblemValidation {
	change, err := domainProblem.NewPendingJudgingChange(testCases, checker)
	if err != nil {
		panic(err)
	}
	v, err := domainProblem.NewJudgingUpdateValidation("validation-1", testProbID, shared.RestoreUserID(authorID), change, testNow)
	if err != nil {
		panic(err)
	}
	return v
}

func stagedChecker() *domainProblem.JudgingFile {
	f := domainProblem.RestoreJudgingFile("Checker.cpp", "problems/"+testSlug+"/checker/validation-1/Checker.cpp", "cpp20", nil, nil)
	return &f
}

type judgingUpdateFixture struct {
	uc        *ValidateProblemUseCase
	ticket    *domainProblem.ProblemValidation
	preparer  *mockJudgingPreparer
	validator *mockSolutionValidator
	staged    *mockStagedFiles
	published *mockProblemPublisher
	problem   *domainProblem.Problem
	saved     []*domainProblem.Problem
}

func newJudgingUpdateFixture(ticket *domainProblem.ProblemValidation) *judgingUpdateFixture {
	f := &judgingUpdateFixture{
		ticket:    ticket,
		preparer:  &mockJudgingPreparer{},
		validator: &mockSolutionValidator{},
		staged:    &mockStagedFiles{},
		published: &mockProblemPublisher{},
		problem:   newPublishedProblemWithTestCases(),
	}
	repo := repoWith(f.problem)
	repo.saveFn = func(_ context.Context, p *domainProblem.Problem) error {
		f.saved = append(f.saved, p)
		return nil
	}
	validationRepo := &mockValidationRepository{
		findByIDFn: func(context.Context, string) (*domainProblem.ProblemValidation, error) { return ticket, nil },
	}
	f.uc = NewValidateProblemUseCase(validationRepo, f.preparer, &mockJudgingArtifactWriter{}, f.validator, f.published, &mockTransactionManager{}, repo, f.staged)
	return f
}

func (f *judgingUpdateFixture) run(t *testing.T) {
	t.Helper()
	if err := f.uc.Execute(context.Background(), ValidateProblemInput{ValidationID: "validation-1", Slug: testSlug}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProblem_JudgingUpdate_Passes_AppliesTheStagedFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, stagedChecker()))

	f.run(t)

	if f.ticket.Status().String() != "PASSED" {
		t.Fatalf("ticket status: got %v, want PASSED", f.ticket.Status())
	}
	if len(f.saved) != 1 {
		t.Fatalf("expected the problem saved once, got %d", len(f.saved))
	}
	p := f.saved[0]
	if p.TestCasesKey() == nil || *p.TestCasesKey() != stagedTestCasesKey {
		t.Errorf("TestCasesKey: got %v, want %s", p.TestCasesKey(), stagedTestCasesKey)
	}
	wantCompiled := "problems/" + testSlug + "/checker/validation-1/compiled"
	if p.Checker() == nil || p.Checker().CompiledKey() == nil || *p.Checker().CompiledKey() != wantCompiled {
		t.Errorf("checker compiled key: got %v, want %s", p.Checker(), wantCompiled)
	}
	if p.JudgingUpdatedAt() == nil {
		t.Error("JudgingUpdatedAt must move so the affected submissions can be rejudged")
	}
	if len(f.staged.deletedPrefixes) != 0 {
		t.Errorf("nothing staged should be dropped on success, got %v", f.staged.deletedPrefixes)
	}
	if f.published.calls != 0 {
		t.Error("a judging update must not run the publish step")
	}
}

func TestValidateProblem_JudgingUpdate_ValidatesTheCandidateNotTheLiveFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, stagedChecker()))

	f.run(t)

	want := &JudgingCandidate{
		TestCasesKey:       stagedTestCasesKey,
		CheckerFileKey:     "problems/" + testSlug + "/checker/validation-1/Checker.cpp",
		CheckerLanguage:    "cpp20",
		CheckerCompiledKey: "problems/" + testSlug + "/checker/validation-1/compiled",
	}
	for name, got := range map[string][]*JudgingCandidate{"preparer": f.preparer.candidates, "validator": f.validator.candidates} {
		if len(got) != 1 || got[0] == nil || *got[0] != *want {
			t.Errorf("%s candidate: got %+v, want %+v", name, got, want)
		}
	}
}

func TestValidateProblem_JudgingUpdate_TestCasesOnly_LeavesTheCheckerAlone(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, nil))

	f.run(t)

	if got := f.preparer.candidates[0]; got.CheckerFileKey != "" {
		t.Errorf("a test-cases-only update must not carry a checker, got %+v", got)
	}
	if f.saved[0].Checker() != nil {
		t.Errorf("checker: got %+v, want it untouched", f.saved[0].Checker())
	}
}

func TestValidateProblem_JudgingUpdate_SolutionFails_KeepsTheProblemAndDropsTheStagedFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, stagedChecker()))
	f.validator.validateFn = func(context.Context, string) (*SolutionValidationResult, error) {
		return &SolutionValidationResult{
			Passed:  false,
			Failure: &SolutionValidationFailure{Kind: SolutionFailureWrongAnswer, TestCase: "secret/1"},
		}, nil
	}

	f.run(t)

	if f.ticket.Status().String() != "FAILED" {
		t.Fatalf("ticket status: got %v, want FAILED", f.ticket.Status())
	}
	if len(f.saved) != 0 {
		t.Error("the live problem must be left untouched when the update fails")
	}
	want := []string{stagedTestCasesKey, "problems/" + testSlug + "/checker/validation-1/"}
	if len(f.staged.deletedPrefixes) != 2 || f.staged.deletedPrefixes[0] != want[0] || f.staged.deletedPrefixes[1] != want[1] {
		t.Errorf("dropped prefixes: got %v, want %v", f.staged.deletedPrefixes, want)
	}
}

func TestValidateProblem_JudgingUpdate_CheckerDoesNotCompile_FailsBeforeRunningSolutions(t *testing.T) {
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(nil, stagedChecker()))
	f.preparer.prepareFn = func(context.Context, string, string) (*JudgingPreparationResult, error) {
		return &JudgingPreparationResult{
			Failure: &JudgingPreparationFailure{Kind: JudgingFailureCheckerCompileError, FileKey: "Checker.cpp", Log: "syntax error"},
		}, nil
	}
	f.validator.validateFn = func(context.Context, string) (*SolutionValidationResult, error) {
		t.Fatal("solutions must not run when the candidate checker does not compile")
		return nil, nil
	}

	f.run(t)

	if f.ticket.Status().String() != "FAILED" {
		t.Errorf("ticket status: got %v, want FAILED", f.ticket.Status())
	}
	if len(f.saved) != 0 {
		t.Error("the live problem must be left untouched")
	}
}

func TestValidateProblem_JudgingUpdate_InfraError_MarksSystemErrorAndDropsTheStagedFiles(t *testing.T) {
	key := stagedTestCasesKey
	f := newJudgingUpdateFixture(newJudgingUpdateTicket(&key, nil))
	f.preparer.prepareFn = func(context.Context, string, string) (*JudgingPreparationResult, error) {
		return nil, context.DeadlineExceeded
	}

	f.run(t)

	if f.ticket.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("ticket status: got %v, want SYSTEM_ERROR", f.ticket.Status())
	}
	if len(f.staged.deletedPrefixes) != 1 || f.staged.deletedPrefixes[0] != stagedTestCasesKey {
		t.Errorf("dropped prefixes: got %v", f.staged.deletedPrefixes)
	}
}
