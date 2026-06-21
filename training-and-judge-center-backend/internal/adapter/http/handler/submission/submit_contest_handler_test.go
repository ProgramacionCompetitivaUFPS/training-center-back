package submission

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
	domainshared "github.com/training-judge-center/backend/internal/domain/shared"
	domainsubmission "github.com/training-judge-center/backend/internal/domain/submission"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	testGroupID   = "g1"
	testContestID = "c1"
)

func TestSubmitContest_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithSubmitContest(
		&mockContestSubmissionProvider{},
		&mockStandingIDResolver{},
		&mockProblemProvider{},
		&mockSubmissionRepo{},
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/contests/c1/problems/two-sum/submissions", nil)
	r.SetPathValue("groupId", testGroupID)
	r.SetPathValue("contestId", testContestID)
	r.SetPathValue("problemSlug", "two-sum")
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSubmitContest_MissingMultipartReturns400(t *testing.T) {
	h := newHandlerWithSubmitContest(
		&mockContestSubmissionProvider{},
		&mockStandingIDResolver{},
		&mockProblemProvider{},
		&mockSubmissionRepo{},
	)
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/groups/g1/contests/c1/problems/two-sum/submissions", nil)
	r.Header.Set("Authorization", "Bearer tok")
	r.SetPathValue("groupId", testGroupID)
	r.SetPathValue("contestId", testContestID)
	r.SetPathValue("problemSlug", "two-sum")
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestSubmitContest_InvalidLanguageReturns400(t *testing.T) {
	h := newHandlerWithSubmitContest(
		&mockContestSubmissionProvider{},
		&mockStandingIDResolver{},
		&mockProblemProvider{},
		&mockSubmissionRepo{},
	)
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cobol85", "cob", "solution.cbl", []byte("IDENTIFICATION DIVISION."))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected %s, got %s", apperror.ErrCodeValidationError, body.Code)
	}
}

func TestSubmitContest_ContestNotFoundReturns404(t *testing.T) {
	contestProvider := &mockContestSubmissionProvider{
		fn: func(_, _ string) (*appsubmission.ContestSubmissionInfo, error) {
			return nil, apperror.NewNotFound("CONTEST_NOT_FOUND", "contest not found")
		},
	}
	h := newHandlerWithSubmitContest(contestProvider, &mockStandingIDResolver{}, &mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d\nbody: %s", w.Code, w.Body.String())
	}
}

func TestSubmitContest_NotRegisteredReturns403(t *testing.T) {
	standingResolver := &mockStandingIDResolver{
		fn: func(_, _ string) (string, bool, error) { return "", false, nil },
	}
	h := newHandlerWithSubmitContest(&mockContestSubmissionProvider{}, standingResolver, &mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeNotRegistered {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeNotRegistered, body.Code)
	}
}

func TestSubmitContest_ContestNotStartedReturns400(t *testing.T) {
	contestProvider := &mockContestSubmissionProvider{
		fn: func(groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error) {
			return &appsubmission.ContestSubmissionInfo{
				ID:                contestID,
				Name:              "Future Contest",
				GroupID:           groupID,
				StartTime:         time.Now().Add(1 * time.Hour),
				EndTime:           time.Now().Add(3 * time.Hour),
				EnablePostContest: false,
				ProblemIDs:        []string{"problem-001"},
			}, nil
		},
	}
	h := newHandlerWithSubmitContest(contestProvider, &mockStandingIDResolver{}, &mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeContestNotStarted {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeContestNotStarted, body.Code)
	}
}

func TestSubmitContest_ContestFinishedNoPostContestReturns400(t *testing.T) {
	contestProvider := &mockContestSubmissionProvider{
		fn: func(groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error) {
			return &appsubmission.ContestSubmissionInfo{
				ID:                contestID,
				Name:              "Past Contest",
				GroupID:           groupID,
				StartTime:         time.Now().Add(-3 * time.Hour),
				EndTime:           time.Now().Add(-1 * time.Hour),
				EnablePostContest: false,
				ProblemIDs:        []string{"problem-001"},
			}, nil
		},
	}
	h := newHandlerWithSubmitContest(contestProvider, &mockStandingIDResolver{}, &mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeContestFinished {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeContestFinished, body.Code)
	}
}

func TestSubmitContest_ProblemNotInContestReturns400(t *testing.T) {
	contestProvider := &mockContestSubmissionProvider{
		fn: func(groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error) {
			return &appsubmission.ContestSubmissionInfo{
				ID:                contestID,
				Name:              "Weekly Contest",
				GroupID:           groupID,
				StartTime:         time.Now().Add(-1 * time.Hour),
				EndTime:           time.Now().Add(1 * time.Hour),
				EnablePostContest: false,
				ProblemIDs:        []string{"other-problem-id"}, // "problem-001" NOT here
			}, nil
		},
	}
	h := newHandlerWithSubmitContest(contestProvider, &mockStandingIDResolver{}, &mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Code != domainsubmission.ErrCodeProblemNotInContest {
		t.Errorf("expected %s, got %s", domainsubmission.ErrCodeProblemNotInContest, body.Code)
	}
}

func TestSubmitContest_SuccessReturns201(t *testing.T) {
	h := newHandlerWithSubmitContest(
		&mockContestSubmissionProvider{},
		&mockStandingIDResolver{},
		&mockProblemProvider{},
		&mockSubmissionRepo{},
	)
	w := httptest.NewRecorder()
	fileData := []byte("int main(){}")
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", fileData)
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d\nbody: %s", w.Code, w.Body.String())
	}
	var body submitContestResponse
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.ID == "" {
		t.Error("expected non-empty ID")
	}
	if body.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", body.Status)
	}
	if body.Language != "cpp20" {
		t.Errorf("Language = %q, want cpp20", body.Language)
	}
	if body.Compiler != "g++" {
		t.Errorf("Compiler = %q, want g++", body.Compiler)
	}
	if body.FileHash == "" {
		t.Error("expected non-empty FileHash")
	}
	if body.FileSize != len(fileData) {
		t.Errorf("FileSize = %d, want %d", body.FileSize, len(fileData))
	}
	if body.ProblemSlug != "two-sum" {
		t.Errorf("ProblemSlug = %q, want two-sum", body.ProblemSlug)
	}
	if body.ContestID != testContestID {
		t.Errorf("ContestID = %q, want %s", body.ContestID, testContestID)
	}
	if body.ContestName != "Weekly Contest #1" {
		t.Errorf("ContestName = %q, want 'Weekly Contest #1'", body.ContestName)
	}
	if body.SubmittedAt == "" {
		t.Error("expected non-empty SubmittedAt")
	}
}

func TestSubmitContest_PostContestAllowedWhenEnabled(t *testing.T) {
	contestProvider := &mockContestSubmissionProvider{
		fn: func(groupID, contestID string) (*appsubmission.ContestSubmissionInfo, error) {
			return &appsubmission.ContestSubmissionInfo{
				ID:                contestID,
				Name:              "Finished Contest",
				GroupID:           groupID,
				StartTime:         time.Now().Add(-3 * time.Hour),
				EndTime:           time.Now().Add(-1 * time.Hour),
				EnablePostContest: true, // post-contest enabled
				ProblemIDs:        []string{"problem-001"},
			}, nil
		},
	}
	h := newHandlerWithSubmitContest(contestProvider, &mockStandingIDResolver{}, &mockProblemProvider{}, &mockSubmissionRepo{})
	w := httptest.NewRecorder()
	r := contestMultipartRequest(testGroupID, testContestID, "two-sum",
		"cpp20", "g++", "solution.cpp", []byte("int main(){}"))
	wrapWithAuth(http.HandlerFunc(h.SubmitContest), &domainuser.TokenClaims{UserID: testAuthorID, Role: domainshared.RoleContestant}).ServeHTTP(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 for post-contest, got %d\nbody: %s", w.Code, w.Body.String())
	}
}
