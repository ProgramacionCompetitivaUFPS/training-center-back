package contest

import (
	"context"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func newListSubmissionsUseCase(
	contest *domainContest.Contest,
	group *GroupInfo,
	memberProvider *mockGroupMemberProvider,
	participantProvider *mockContestParticipantProvider,
	subsProvider *mockContestSubmissionsProvider,
) *ListContestSubmissionsUseCase {
	return NewListContestSubmissionsUseCase(
		repoWith(contest),
		&mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) { return group, nil }},
		memberProvider,
		participantProvider,
		subsProvider,
	)
}

func registeredParticipant() *mockContestParticipantProvider {
	return &mockContestParticipantProvider{
		isRegisteredFn: func(_ context.Context, _, _ string) (bool, error) { return true, nil },
	}
}

func notRegisteredParticipant() *mockContestParticipantProvider {
	return &mockContestParticipantProvider{
		isRegisteredFn: func(_ context.Context, _, _ string) (bool, error) { return false, nil },
	}
}

func frozenActiveContest() *domainContest.Contest {
	// freeze = last 60 minutes; contest ends in 30m → freeze started 30m ago
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Frozen Active Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(30*time.Minute),
		domainContest.RestorePenalty(20),
		60,
		false, false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func richSub(id, userID, status string, minutesAgo int) RichSubmissionData {
	return RichSubmissionData{
		ID:           id,
		ProblemID:    "prob-1",
		ProblemSlug:  "sum",
		ProblemTitle: "Sum of Two Numbers",
		ProblemOrder: 1,
		UserID:       userID,
		Nickname:     "nick_" + userID,
		Language:     "cpp20",
		Status:       status,
		SubmittedAt:  time.Now().Add(-time.Duration(minutesAgo) * time.Minute),
	}
}

func richSubWithTime(id, userID, status string, at time.Time) RichSubmissionData {
	return RichSubmissionData{
		ID:           id,
		ProblemID:    "prob-1",
		ProblemSlug:  "sum",
		ProblemTitle: "Sum of Two Numbers",
		ProblemOrder: 1,
		UserID:       userID,
		Nickname:     "nick_" + userID,
		Language:     "cpp20",
		Status:       status,
		SubmittedAt:  at,
	}
}

func subsProvider(subs []RichSubmissionData) *mockContestSubmissionsProvider {
	return &mockContestSubmissionsProvider{
		listByContestFn: func(_ context.Context, _ string, _ ContestSubmissionFilters) ([]RichSubmissionData, error) {
			return subs, nil
		},
	}
}

func baseInput() ListContestSubmissionsInput {
	return ListContestSubmissionsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       50,
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestListContestSubmissions_NotRegistered_Returns403(t *testing.T) {
	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), notRegisteredParticipant(), subsProvider(nil))

	_, err := uc.Execute(context.Background(), baseInput())
	if err == nil {
		t.Fatal("expected 403 error, got nil")
	}
}

func TestListContestSubmissions_HiddenGroupNonMember_Returns404(t *testing.T) {
	uc := newListSubmissionsUseCase(activeContest(), hiddenGroup(), notLead(), registeredParticipant(), subsProvider(nil))

	_, err := uc.Execute(context.Background(), baseInput())
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}

func TestListContestSubmissions_ActiveContest_NoExecutionDetails(t *testing.T) {
	judgedAt := time.Now().Add(-30 * time.Minute)
	timeMs := 100
	memKb := 2048
	sub := richSub("s1", callerID, "ACCEPTED", 45)
	sub.JudgedAt = &judgedAt
	sub.TimeMs = &timeMs
	sub.MemoryKb = &memKb

	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider([]RichSubmissionData{sub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Fatalf("len(submissions)=%d, want 1", len(out.Submissions))
	}
	s := out.Submissions[0]
	if s.JudgedAt != nil {
		t.Error("judgedAt should be nil during ACTIVE for participants")
	}
	if s.TimeMs != nil {
		t.Error("timeMs should be nil during ACTIVE for participants")
	}
	if s.Phase != nil {
		t.Error("phase should be nil during ACTIVE for participants")
	}
	if s.Status != "ACCEPTED" {
		t.Errorf("status=%q, want ACCEPTED", s.Status)
	}
}

func TestListContestSubmissions_FinishedContest_ReturnsFullDetails(t *testing.T) {
	judgedAt := time.Now().Add(-2 * time.Hour)
	timeMs := 100
	memKb := 2048
	// finishedContest() has endTime = Now-1h; submit 2h ago → competition phase
	sub := richSubWithTime("s1", callerID, "ACCEPTED", time.Now().Add(-2*time.Hour))
	sub.JudgedAt = &judgedAt
	sub.TimeMs = &timeMs
	sub.MemoryKb = &memKb

	uc := newListSubmissionsUseCase(finishedContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider([]RichSubmissionData{sub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	s := out.Submissions[0]
	if s.JudgedAt == nil {
		t.Error("judgedAt should be set after FINISHED")
	}
	if s.TimeMs == nil {
		t.Error("timeMs should be set after FINISHED")
	}
	if s.Phase == nil || *s.Phase != "competition" {
		t.Errorf("phase=%v, want 'competition'", s.Phase)
	}
}

func TestListContestSubmissions_FreezePeriod_OwnSubmissionsAlwaysVisible(t *testing.T) {
	// frozen contest: freeze started 30 min ago
	// own submission 20 min ago (after freeze) → must still appear
	postFreeze := time.Now().Add(-20 * time.Minute)
	ownSub := richSubWithTime("s1", callerID, "ACCEPTED", postFreeze)

	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider([]RichSubmissionData{ownSub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Errorf("len(submissions)=%d, want 1 (own post-freeze submission must appear)", len(out.Submissions))
	}
}

func TestListContestSubmissions_FreezePeriod_OthersPostFreezeHidden(t *testing.T) {
	// frozen contest: freeze started 30 min ago
	// other's submission 20 min ago (after freeze) → must NOT appear
	postFreeze := time.Now().Add(-20 * time.Minute)
	otherSub := richSubWithTime("s2", otherID, "ACCEPTED", postFreeze)
	preFreeze := time.Now().Add(-45 * time.Minute)
	otherPreSub := richSubWithTime("s1", otherID, "WRONG_ANSWER", preFreeze)

	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider([]RichSubmissionData{otherSub, otherPreSub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Errorf("len(submissions)=%d, want 1 (only pre-freeze other visible)", len(out.Submissions))
	}
	if out.Submissions[0].ID != "s1" {
		t.Errorf("visible submission id=%q, want s1 (pre-freeze)", out.Submissions[0].ID)
	}
}

func TestListContestSubmissions_Lead_SeesAllDuringFreeze(t *testing.T) {
	postFreeze := time.Now().Add(-20 * time.Minute)
	otherSub := richSubWithTime("s1", otherID, "ACCEPTED", postFreeze)

	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isLead(), registeredParticipant(), subsProvider([]RichSubmissionData{otherSub}))

	input := baseInput()
	input.CurrentUser = asCoach(callerID)

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Errorf("len(submissions)=%d, want 1 (lead sees all)", len(out.Submissions))
	}
}

func TestListContestSubmissions_PhaseFilter_Competition(t *testing.T) {
	// finishedContest() has endTime = Now-1h
	// competition: 2h ago (before endTime); postCompetition: 30min ago (after endTime)
	postCompetition := time.Now().Add(-30 * time.Minute)
	competition := time.Now().Add(-2 * time.Hour)

	subs := []RichSubmissionData{
		richSubWithTime("s1", callerID, "ACCEPTED", competition),
		richSubWithTime("s2", callerID, "WRONG_ANSWER", postCompetition),
	}

	uc := newListSubmissionsUseCase(finishedContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider(subs))

	input := baseInput()
	input.Phase = "competition"

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Fatalf("len(submissions)=%d, want 1 (only competition phase)", len(out.Submissions))
	}
	if out.Submissions[0].ID != "s1" {
		t.Errorf("id=%q, want s1", out.Submissions[0].ID)
	}
}

func TestListContestSubmissions_Pagination(t *testing.T) {
	subs := make([]RichSubmissionData, 5)
	for i := range subs {
		subs[i] = richSub("s"+string(rune('1'+i)), callerID, "ACCEPTED", 10*(i+1))
	}

	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider(subs))

	input := baseInput()
	input.Page = 2
	input.Limit = 2

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 5 {
		t.Errorf("total=%d, want 5", out.Total)
	}
	if len(out.Submissions) != 2 {
		t.Errorf("page entries=%d, want 2", len(out.Submissions))
	}
}

func TestListContestSubmissions_Meta_ContestContext(t *testing.T) {
	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isMemberNotLead(), registeredParticipant(), subsProvider(nil))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Meta.InFreeze {
		t.Error("meta.InFreeze should be true for frozen contest")
	}
	if out.Meta.FreezeTime == nil {
		t.Error("meta.FreezeTime should be set")
	}
	if out.Meta.Status != "ACTIVE" {
		t.Errorf("meta.Status=%q, want ACTIVE", out.Meta.Status)
	}
}
