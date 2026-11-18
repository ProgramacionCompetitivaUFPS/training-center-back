package contest

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appcontest "github.com/training-judge-center/backend/internal/application/contest"
	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func newHandlerWithUpdate(uc *appcontest.UpdateContestUseCase) *Handler {
	return &Handler{updateContest: uc}
}

// stubUpdateRepo returns a valid contest from FindByID; all writes are no-ops.
type stubUpdateRepo struct{}

func (s *stubUpdateRepo) Create(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *stubUpdateRepo) Update(_ context.Context, _ *domainContest.Contest) error { return nil }
func (s *stubUpdateRepo) FindByID(_ context.Context, _ string) (*domainContest.Contest, error) {
	return domainContest.RestoreContest(
		"c1",
		domainContest.RestoreContestName("Old Name"),
		nil,
		time.Now().Add(24*time.Hour),
		time.Now().Add(29*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, false,
		shared.RestoreGroupID("g1"),
		shared.RestoreUserID("u1"),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		time.Now().Add(-time.Hour),
		nil,
	), nil
}
func (s *stubUpdateRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubUpdateRepo) List(_ context.Context, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}
func (s *stubUpdateRepo) ListByGroupIDs(_ context.Context, _ []string, _ domainContest.ListFilters) ([]*domainContest.Contest, int, error) {
	return nil, 0, nil
}

func defaultUpdateUC() *appcontest.UpdateContestUseCase {
	return appcontest.NewUpdateContestUseCase(
		&stubUpdateRepo{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockProblemProvider{},
		&mockOwnerProvider{},
		&mockTeamParticipRepo{},
		&mockTransactionManager{},
	)
}

func TestUpdateContest_Unauthenticated_Returns401(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
	r := httptest.NewRequest(http.MethodPut, "/groups/g1/contests/c1", nil)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	http.HandlerFunc(h.Update).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestUpdateContest_MissingPathParams_Returns400(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
	r := authedRequest(http.MethodPut, "/groups//contests/", nil)
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestUpdateContest_InvalidBody_Returns400(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())
	r := authedRequest(http.MethodPut, "/groups/g1/contests/c1", []byte("not-json"))
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error: %v", err)
	}
	if resp.Code != apperror.ErrCodeValidationError {
		t.Errorf("expected VALIDATION_ERROR, got %s", resp.Code)
	}
}

func TestUpdateContest_ValidRequest_Returns200(t *testing.T) {
	h := newHandlerWithUpdate(defaultUpdateUC())

	newName := "Updated Name"
	body, _ := json.Marshal(map[string]interface{}{"name": newName})
	r := authedRequest(http.MethodPut, "/groups/g1/contests/c1", body)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp contestResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}
	if resp.Name != newName {
		t.Errorf("expected name %q, got %q", newName, resp.Name)
	}
}

func TestUpdateContest_ContestNotFound_Returns404(t *testing.T) {
	uc := appcontest.NewUpdateContestUseCase(
		&mockContestRepo{}, // FindByID returns 404 by default
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockProblemProvider{},
		&mockOwnerProvider{},
		&mockTeamParticipRepo{},
		&mockTransactionManager{},
	)
	h := newHandlerWithUpdate(uc)

	body, _ := json.Marshal(map[string]interface{}{"name": "x"})
	r := authedRequest(http.MethodPut, "/groups/g1/contests/missing", body)
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "missing")
	w := httptest.NewRecorder()

	wrapAuth(http.HandlerFunc(h.Update)).ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

type coloredUpdateRepo struct{ stubUpdateRepo }

func (s *coloredUpdateRepo) FindByID(ctx context.Context, id string) (*domainContest.Contest, error) {
	c, err := s.stubUpdateRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	red := "#aabbcc"
	c.SetProblems([]domainContest.ContestProblem{domainContest.RestoreContestProblem("cp1", "p1", 1, &red)}, time.Now())
	return c, nil
}

func updateWithColoredProblem() *appcontest.UpdateContestUseCase {
	return appcontest.NewUpdateContestUseCase(
		&coloredUpdateRepo{},
		&mockGroupProvider{},
		&mockMemberProvider{isLead: true, isMember: true},
		&mockProblemProvider{},
		&mockOwnerProvider{},
		&mockTeamParticipRepo{},
		&mockTransactionManager{},
	)
}

func putProblems(t *testing.T, uc *appcontest.UpdateContestUseCase, problemsJSON string) *httptest.ResponseRecorder {
	t.Helper()
	r := authedRequest(http.MethodPut, "/groups/g1/contests/c1", []byte(`{"problems":`+problemsJSON+`}`))
	r.SetPathValue("groupId", "g1")
	r.SetPathValue("contestId", "c1")
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(newHandlerWithUpdate(uc).Update)).ServeHTTP(w, r)
	return w
}

func colorKeyOf(t *testing.T, body []byte) (value interface{}, present bool) {
	t.Helper()
	var raw struct {
		Problems []map[string]interface{} `json:"problems"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || len(raw.Problems) == 0 {
		t.Fatalf("could not decode problems from %s: %v", body, err)
	}
	value, present = raw.Problems[0]["color"]
	return value, present
}

func TestUpdateContest_ColorInBodyIsAnsweredLowercased(t *testing.T) {
	w := putProblems(t, updateWithColoredProblem(), `[{"slug":"a","order":1,"color":"#112233"},{"slug":"b","order":2}]`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got, _ := colorKeyOf(t, w.Body.Bytes()); got != "#112233" {
		t.Errorf("color = %v, want #112233", got)
	}
}

func TestUpdateContest_OmittedColorKeepsTheCurrentOne(t *testing.T) {
	w := putProblems(t, updateWithColoredProblem(), `[{"slug":"a","order":1}]`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if got, _ := colorKeyOf(t, w.Body.Bytes()); got != "#aabbcc" {
		t.Errorf("color = %v, want #aabbcc", got)
	}
}

func TestUpdateContest_NullColorClearsTheCurrentOneAndIsAnsweredAsNull(t *testing.T) {
	w := putProblems(t, updateWithColoredProblem(), `[{"slug":"a","order":1,"color":null}]`)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	got, present := colorKeyOf(t, w.Body.Bytes())
	if !present || got != nil {
		t.Errorf("color = %v (present=%v), want an explicit null", got, present)
	}
}

func TestUpdateContest_InvalidColor_Returns400WithItsCode(t *testing.T) {
	w := putProblems(t, updateWithColoredProblem(), `[{"slug":"a","order":1,"color":"red"}]`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
	var resp apperror.AppError
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("could not decode error: %v", err)
	}
	if resp.Code != domainContest.ErrCodeInvalidProblemColor {
		t.Errorf("error = %q, want %q", resp.Code, domainContest.ErrCodeInvalidProblemColor)
	}
}

func TestUpdateContest_NonStringColor_Returns400(t *testing.T) {
	w := putProblems(t, updateWithColoredProblem(), `[{"slug":"a","order":1,"color":123}]`)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
