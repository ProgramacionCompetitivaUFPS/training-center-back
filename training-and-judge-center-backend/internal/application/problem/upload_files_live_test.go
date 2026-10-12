package problem

import (
	"context"
	"errors"
	"strings"
	"testing"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type liveUploadFixture struct {
	uc            *UploadProblemFilesUseCase
	storage       *mockFileStorage
	queue         *mockValidationQueue
	tickets       []*domainProblem.ProblemValidation
	uploaded      []string
	problemSaveCt int
}

func newLiveUploadFixture(p *domainProblem.Problem, inActiveContest bool) *liveUploadFixture {
	f := &liveUploadFixture{queue: &mockValidationQueue{}}
	repo := repoWith(p)
	repo.saveFn = func(context.Context, *domainProblem.Problem) error {
		f.problemSaveCt++
		return nil
	}
	f.storage = &mockFileStorage{
		uploadFileFn: func(_ context.Context, path string, _ []byte) error {
			f.uploaded = append(f.uploaded, path)
			return nil
		},
	}
	validationRepo := &mockValidationRepository{
		saveFn: func(_ context.Context, v *domainProblem.ProblemValidation) error {
			f.tickets = append(f.tickets, v)
			return nil
		},
	}
	zipParser := &mockZipParser{parseTestCasesZipFn: func(context.Context, []byte) ([]ParsedFile, error) {
		return []ParsedFile{{Path: "data/secret/1.in", Content: []byte("1")}}, nil
	}}
	contests := &mockAffectedContests{contests: []ContestRef{{ID: "c1", Name: "Round 1"}}}
	f.uc = NewUploadProblemFilesUseCase(repo, validationRepo, f.storage, zipParser, newCppPythonJavaSettings(),
		&mockActiveContestChecker{inActive: inActiveContest}, contests, f.queue)
	return f
}

func liveInput(fileType, fileName string) UploadProblemFilesInput {
	return UploadProblemFilesInput{
		Slug:        testSlug,
		FileType:    fileType,
		FileName:    fileName,
		FileData:    []byte("data"),
		ConfirmSlug: testSlug,
		CurrentUser: asCoach(authorID),
	}
}

func appErrorCode(t *testing.T, err error) string {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %v", err)
	}
	return appErr.Code
}

func TestUploadPublished_TestCases_StagesAndQueuesWithoutTouchingTheProblem(t *testing.T) {
	live := newPublishedProblemWithTestCases()
	liveKey := *live.TestCasesKey()
	f := newLiveUploadFixture(live, true)

	out, err := f.uc.Execute(context.Background(), liveInput(FileTypeTestCases, "tc.zip"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.ValidationID == "" || len(f.queue.published) != 1 || f.queue.published[0].ValidationID != out.ValidationID {
		t.Errorf("expected one queued validation %q, got %+v", out.ValidationID, f.queue.published)
	}
	if f.problemSaveCt != 0 {
		t.Error("the problem must not be saved until the update is validated")
	}
	if len(f.tickets) != 1 || !f.tickets[0].IsJudgingUpdate() || f.tickets[0].PendingChange().TestCasesKey() == nil {
		t.Fatalf("expected one judging-update ticket carrying the staged test cases, got %+v", f.tickets)
	}
	if *f.tickets[0].PendingChange().TestCasesKey() == liveKey {
		t.Error("the staged test cases must not reuse the live prefix")
	}
	if len(out.AffectedContests) != 1 || out.AffectedContests[0].ID != "c1" {
		t.Errorf("AffectedContests: got %+v", out.AffectedContests)
	}
}

func TestUploadPublished_Checker_StagesApartFromTheLiveOne(t *testing.T) {
	f := newLiveUploadFixture(newPublishedProblem(), true)

	out, err := f.uc.Execute(context.Background(), liveInput(FileTypeChecker, "MyChecker.cpp"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantKey := "problems/" + testSlug + "/checker/" + out.ValidationID + "/Checker.cpp"
	if len(f.uploaded) != 1 || f.uploaded[0] != wantKey {
		t.Errorf("uploaded: got %v, want [%s]", f.uploaded, wantKey)
	}
	checker := f.tickets[0].PendingChange().Checker()
	if checker == nil || checker.FileKey() != wantKey || checker.CompiledKey() != nil {
		t.Errorf("pending checker: got %+v", checker)
	}
	if out.Problem.CheckerLoaded {
		t.Error("the response must describe the live problem, which has no checker yet")
	}
	if f.problemSaveCt != 0 {
		t.Error("the problem must not be saved until the update is validated")
	}
}

func TestUploadPublished_Rejections(t *testing.T) {
	asStranger := func(in UploadProblemFilesInput) UploadProblemFilesInput {
		in.CurrentUser = asCoach(strangerID)
		return in
	}
	asModifier := func(in UploadProblemFilesInput) UploadProblemFilesInput {
		in.CurrentUser = asCoach(modifierID)
		return in
	}
	withConfirm := func(slug string) func(UploadProblemFilesInput) UploadProblemFilesInput {
		return func(in UploadProblemFilesInput) UploadProblemFilesInput {
			in.ConfirmSlug = slug
			return in
		}
	}

	tests := []struct {
		name     string
		inActive bool
		fileType string
		file     string
		tweak    func(UploadProblemFilesInput) UploadProblemFilesInput
		wantCode string
	}{
		{"not in an active contest", false, FileTypeTestCases, "tc.zip", nil, ErrCodeProblemIsPublished},
		{"solution", true, FileTypeSolution, "sol.cpp", nil, ErrCodeProblemIsPublished},
		{"validator", true, FileTypeValidator, "v.cpp", nil, ErrCodeProblemIsPublished},
		{"stranger", true, FileTypeTestCases, "tc.zip", asStranger, ErrCodeInsufficientPermissions},
		{"modifier", true, FileTypeTestCases, "tc.zip", asModifier, ErrCodeInsufficientPermissions},
		{"missing confirmation", true, FileTypeTestCases, "tc.zip", withConfirm(""), domainProblem.ErrCodeSlugMismatch},
		{"wrong confirmation", true, FileTypeChecker, "Checker.cpp", withConfirm("other-slug"), domainProblem.ErrCodeSlugMismatch},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLiveUploadFixture(newPublishedProblemWithModifier(), tt.inActive)
			in := liveInput(tt.fileType, tt.file)
			if tt.tweak != nil {
				in = tt.tweak(in)
			}

			_, err := f.uc.Execute(context.Background(), in)

			if code := appErrorCode(t, err); code != tt.wantCode {
				t.Errorf("code: got %q, want %q", code, tt.wantCode)
			}
			if len(f.uploaded) != 0 || len(f.tickets) != 0 || len(f.queue.published) != 0 {
				t.Error("a rejected upload must stage, save and queue nothing")
			}
		})
	}
}

func TestUploadPublished_Admin_IsAllowed(t *testing.T) {
	f := newLiveUploadFixture(newPublishedProblem(), true)
	in := liveInput(FileTypeChecker, "Checker.cpp")
	in.CurrentUser = asAdmin(strangerID)

	if _, err := f.uc.Execute(context.Background(), in); err != nil {
		t.Fatalf("an admin should be allowed, got: %v", err)
	}
}

func TestUploadPublished_ValidationInProgress_IsRejected(t *testing.T) {
	f := newLiveUploadFixture(newPublishedProblem(), true)
	f.uc.validationRepo = &mockValidationRepository{
		findLatestByProblemIDFn: func(context.Context, string) (*domainProblem.ProblemValidation, bool, error) {
			return runningValidationFixture(), true, nil
		},
	}

	_, err := f.uc.Execute(context.Background(), liveInput(FileTypeChecker, "Checker.cpp"))

	if code := appErrorCode(t, err); code != domainProblem.ErrCodeValidationInProgress {
		t.Errorf("code: got %q, want %q", code, domainProblem.ErrCodeValidationInProgress)
	}
}

func TestUploadPublished_QueueFails_DropsTheStagedFileAndClosesTheTicket(t *testing.T) {
	f := newLiveUploadFixture(newPublishedProblem(), true)
	f.queue.publishFn = func(context.Context, ValidationQueueMessage) error { return apperror.NewInternal() }
	var deleted []string
	f.storage.deleteFileFn = func(_ context.Context, path string) error {
		deleted = append(deleted, path)
		return nil
	}

	_, err := f.uc.Execute(context.Background(), liveInput(FileTypeChecker, "Checker.cpp"))

	if err == nil {
		t.Fatal("expected the queue error")
	}
	if len(deleted) != 1 || !strings.HasSuffix(deleted[0], "/Checker.cpp") {
		t.Errorf("deleted: got %v, want the staged checker", deleted)
	}
	if last := f.tickets[len(f.tickets)-1]; last.Status().String() != "SYSTEM_ERROR" {
		t.Errorf("ticket status: got %v, want SYSTEM_ERROR so it releases the problem", last.Status())
	}
}
