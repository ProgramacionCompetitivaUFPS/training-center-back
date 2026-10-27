package team

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	appTeam "github.com/training-judge-center/backend/internal/application/team"
	domainShared "github.com/training-judge-center/backend/internal/domain/shared"
	domainTeam "github.com/training-judge-center/backend/internal/domain/team"
)

func newHandlerWithLeaveImpact(memberRepo domainTeam.MemberRepository, atRisk []appTeam.RegistrationAtRisk) *Handler {
	return &Handler{getLeaveImpact: appTeam.NewGetLeaveImpactUseCase(memberRepo, &mockScheduledParticipationCleanerHandler{atRisk: atRisk})}
}

func TestGetLeaveImpact_UnauthenticatedReturns401(t *testing.T) {
	h := newHandlerWithLeaveImpact(&mockMemberRepo{}, nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/teams/t1/leave-impact", nil)
	wrapAuth(http.HandlerFunc(h.GetLeaveImpact)).ServeHTTP(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetLeaveImpact_NotMemberReturns404(t *testing.T) {
	h := newHandlerWithLeaveImpact(&mockMemberRepo{}, nil)
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetLeaveImpact)).ServeHTTP(w, authedGetRequest("/teams/t1/leave-impact"))
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestGetLeaveImpact_ReturnsTheRegistrationsAtRisk(t *testing.T) {
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	h := newHandlerWithLeaveImpact(memberRepo, []appTeam.RegistrationAtRisk{{ContestID: "c1", ContestName: "Taller ICPC", GroupID: "g1"}})
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetLeaveImpact)).ServeHTTP(w, authedGetRequest("/teams/t1/leave-impact"))
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		RegistrationsToCancel []struct {
			ContestID   string `json:"contestId"`
			ContestName string `json:"contestName"`
			GroupID     string `json:"groupId"`
		} `json:"registrationsToCancel"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(body.RegistrationsToCancel) != 1 || body.RegistrationsToCancel[0].ContestName != "Taller ICPC" || body.RegistrationsToCancel[0].GroupID != "g1" {
		t.Errorf("unexpected body: %s", w.Body.String())
	}
}

func TestGetLeaveImpact_NothingAtRiskReturnsAnEmptyList(t *testing.T) {
	memberRepo := &mockMemberRepo{
		findByTeamAndUserFn: func(_ string, userID domainShared.UserID) (*domainTeam.TeamMember, error) {
			return makeHandlerMember("m1", "t1", userID.Value(), time.Now()), nil
		},
	}
	h := newHandlerWithLeaveImpact(memberRepo, nil)
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetLeaveImpact)).ServeHTTP(w, authedGetRequest("/teams/t1/leave-impact"))
	if w.Code != http.StatusOK || w.Body.String() == "" {
		t.Fatalf("expected 200 with a body, got %d: %s", w.Code, w.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(w.Body.Bytes(), &raw); err != nil || string(raw["registrationsToCancel"]) != "[]" {
		t.Errorf("expected registrationsToCancel to be [], got %s", w.Body.String())
	}
}
