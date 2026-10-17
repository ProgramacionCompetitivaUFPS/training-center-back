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
	subsProvider *mockContestSubmissionsProvider,
) *ListContestSubmissionsUseCase {
	return NewListContestSubmissionsUseCase(
		repoWith(contest),
		&mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) { return group, nil }},
		memberProvider,
		subsProvider,
		&mockCallerStandingProvider{},
	)
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
		false, false, false,
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
		StandingID:   userID, // individual by default
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
		StandingID:   userID, // individual by default
		Nickname:     "nick_" + userID,
		Language:     "cpp20",
		Status:       status,
		SubmittedAt:  at,
	}
}

func teamSub(id, userID, teamID, teamName, status string, at time.Time, members []string) RichSubmissionData {
	return RichSubmissionData{
		ID:                  id,
		ProblemID:           "prob-1",
		ProblemSlug:         "sum",
		ProblemTitle:        "Sum of Two Numbers",
		ProblemOrder:        1,
		UserID:              userID,
		StandingID:          teamID,
		Nickname:            "nick_" + userID,
		TeamID:              &teamID,
		TeamName:            &teamName,
		TeamMemberNicknames: members,
		Language:            "cpp20",
		Status:              status,
		SubmittedAt:         at,
	}
}

func subsProvider(subs []RichSubmissionData) *mockContestSubmissionsProvider {
	return &mockContestSubmissionsProvider{
		listByContestFn: func(_ context.Context, _ string, _ ContestSubmissionFilters) ([]RichSubmissionData, int, error) {
			return subs, len(subs), nil
		},
	}
}

// capturingSubsProvider records the filters the use case sends; filtering and
// pagination happen in SQL, so these tests assert on the request, not the rows.
func capturingSubsProvider(got *ContestSubmissionFilters) *mockContestSubmissionsProvider {
	return &mockContestSubmissionsProvider{
		listByContestFn: func(_ context.Context, _ string, f ContestSubmissionFilters) ([]RichSubmissionData, int, error) {
			*got = f
			return nil, 0, nil
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

func TestListContestSubmissions_VisibleGroupSpectator_CanList(t *testing.T) {
	uc := newListSubmissionsUseCase(finishedContest(), visibleGroup(), notLead(), subsProvider([]RichSubmissionData{
		richSub("s1", otherID, "ACCEPTED", 5),
	}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Errorf("len(submissions)=%d, want 1", len(out.Submissions))
	}
}

func TestListContestSubmissions_CanViewDetail(t *testing.T) {
	publicSub := richSub("public", otherID, "ACCEPTED", 5)
	publicSub.Visibility = "PUBLIC"
	privateSub := richSub("private", otherID, "ACCEPTED", 6)
	privateSub.Visibility = "PRIVATE"
	ownSub := richSub("own", callerID, "ACCEPTED", 7)
	ownSub.Visibility = "PRIVATE"
	uc := newListSubmissionsUseCase(finishedContest(), visibleGroup(), notLead(), subsProvider([]RichSubmissionData{
		publicSub, privateSub, ownSub,
	}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]bool{"public": true, "private": false, "own": true}
	for _, s := range out.Submissions {
		if s.CanViewDetail != want[s.ID] {
			t.Errorf("%s: CanViewDetail=%v, want %v", s.ID, s.CanViewDetail, want[s.ID])
		}
	}
}

func TestListContestSubmissions_Lead_CanViewEveryDetail(t *testing.T) {
	privateSub := richSub("private", otherID, "ACCEPTED", 5)
	privateSub.Visibility = "PRIVATE"
	uc := newListSubmissionsUseCase(finishedContest(), visibleGroup(), isLead(), subsProvider([]RichSubmissionData{privateSub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !out.Submissions[0].CanViewDetail {
		t.Error("lead should be able to open any submission")
	}
}

func TestListContestSubmissions_HiddenGroupNonMember_Returns404(t *testing.T) {
	uc := newListSubmissionsUseCase(activeContest(), hiddenGroup(), notLead(), subsProvider(nil))

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

	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), subsProvider([]RichSubmissionData{sub}))

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

	uc := newListSubmissionsUseCase(finishedContest(), visibleGroup(), isMemberNotLead(), subsProvider([]RichSubmissionData{sub}))

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

func TestListContestSubmissions_FreezePeriod_ParticipantGetsFreezeFilter(t *testing.T) {
	// frozen contest: freeze started 30 min ago → others' post-freeze rows hidden, own standing kept
	var got ContestSubmissionFilters
	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isMemberNotLead(), capturingSubsProvider(&got))

	if _, err := uc.Execute(context.Background(), baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FreezeAt == nil {
		t.Fatal("FreezeAt should be set for a participant during freeze")
	}
	wantFreeze := frozenActiveContest().EndTime().Add(-60 * time.Minute)
	if d := got.FreezeAt.Sub(wantFreeze); d > time.Second || d < -time.Second {
		t.Errorf("FreezeAt=%v, want ≈%v", got.FreezeAt, wantFreeze)
	}
	if got.ExceptStandingID != callerID {
		t.Errorf("ExceptStandingID=%q, want %q", got.ExceptStandingID, callerID)
	}
}

func TestListContestSubmissions_NotFrozen_NoFreezeFilter(t *testing.T) {
	var got ContestSubmissionFilters
	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), capturingSubsProvider(&got))

	if _, err := uc.Execute(context.Background(), baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FreezeAt != nil {
		t.Errorf("FreezeAt=%v, want nil outside the freeze window", got.FreezeAt)
	}
}

func TestListContestSubmissions_Lead_SeesAllDuringFreeze(t *testing.T) {
	var got ContestSubmissionFilters
	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isLead(), capturingSubsProvider(&got))

	input := baseInput()
	input.CurrentUser = asCoach(callerID)

	if _, err := uc.Execute(context.Background(), input); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.FreezeAt != nil {
		t.Errorf("FreezeAt=%v, want nil (lead sees all)", got.FreezeAt)
	}
}

func TestListContestSubmissions_PhaseFilter(t *testing.T) {
	// One instance, reused below: finishedContest() reads time.Now() internally, so calling
	// it again per subtest drifted endTime from the boundary the use case actually derived,
	// making the equality check below flaky by a few hundred microseconds.
	contest := finishedContest()
	endTime := contest.EndTime()
	tests := []struct {
		phase              string
		wantAtMost, wantAf bool
	}{
		{"competition", true, false},
		{"postcompetition", false, true},
		{"", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			var got ContestSubmissionFilters
			uc := newListSubmissionsUseCase(contest, visibleGroup(), isMemberNotLead(), capturingSubsProvider(&got))

			input := baseInput()
			input.Phase = tt.phase
			if _, err := uc.Execute(context.Background(), input); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if (got.SubmittedAtMost != nil) != tt.wantAtMost {
				t.Errorf("SubmittedAtMost=%v, want set=%v", got.SubmittedAtMost, tt.wantAtMost)
			}
			if (got.SubmittedAfter != nil) != tt.wantAf {
				t.Errorf("SubmittedAfter=%v, want set=%v", got.SubmittedAfter, tt.wantAf)
			}
			for _, ts := range []*time.Time{got.SubmittedAtMost, got.SubmittedAfter} {
				if ts != nil && !ts.Equal(endTime) {
					t.Errorf("phase boundary=%v, want endTime %v", ts, endTime)
				}
			}
		})
	}
}

func TestListContestSubmissions_Pagination(t *testing.T) {
	var got ContestSubmissionFilters
	page := []RichSubmissionData{
		richSub("s3", callerID, "ACCEPTED", 30),
		richSub("s4", callerID, "ACCEPTED", 40),
	}
	provider := &mockContestSubmissionsProvider{
		listByContestFn: func(_ context.Context, _ string, f ContestSubmissionFilters) ([]RichSubmissionData, int, error) {
			got = f
			return page, 5, nil
		},
	}
	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), provider)

	input := baseInput()
	input.Page = 2
	input.Limit = 2

	out, err := uc.Execute(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Limit != 2 || got.Offset != 2 {
		t.Errorf("limit/offset=%d/%d, want 2/2", got.Limit, got.Offset)
	}
	if out.Total != 5 {
		t.Errorf("total=%d, want 5", out.Total)
	}
	if len(out.Submissions) != 2 {
		t.Errorf("page entries=%d, want 2", len(out.Submissions))
	}
}

func TestListContestSubmissions_Meta_ContestContext(t *testing.T) {
	uc := newListSubmissionsUseCase(frozenActiveContest(), visibleGroup(), isMemberNotLead(), subsProvider(nil))

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

func TestListContestSubmissions_TeamSubmission_ShowsTeamInfo(t *testing.T) {
	teamID := "team-001"
	teamName := "Team Alpha"
	members := []string{"alice", "bob", "carol"}
	sub := teamSub("s1", callerID, teamID, teamName, "ACCEPTED", time.Now().Add(-30*time.Minute), members)

	uc := newListSubmissionsUseCase(activeContest(), visibleGroup(), isMemberNotLead(), subsProvider([]RichSubmissionData{sub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Fatalf("len(submissions)=%d, want 1", len(out.Submissions))
	}
	by := out.Submissions[0].SubmittedBy
	if by.Type != "TEAM" {
		t.Errorf("type=%q, want TEAM", by.Type)
	}
	if by.TeamID != teamID {
		t.Errorf("teamId=%q, want %q", by.TeamID, teamID)
	}
	if by.TeamName != teamName {
		t.Errorf("teamName=%q, want %q", by.TeamName, teamName)
	}
	// showTeamMembers=false by default → members hidden
	if len(by.Members) != 0 {
		t.Errorf("members should be empty when showTeamMembers=false, got %v", by.Members)
	}
}

func TestListContestSubmissions_TeamSubmission_ShowsMembersWhenEnabled(t *testing.T) {
	teamID := "team-001"
	teamName := "Team Alpha"
	members := []string{"alice", "bob", "carol"}
	sub := teamSub("s1", callerID, teamID, teamName, "ACCEPTED", time.Now().Add(-30*time.Minute), members)

	// Contest with showTeamMembers=true
	contest := domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Team Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(2*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false, true, // showTeamMembers=true
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("TEAM"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)

	uc := newListSubmissionsUseCase(contest, visibleGroup(), isMemberNotLead(), subsProvider([]RichSubmissionData{sub}))

	out, err := uc.Execute(context.Background(), baseInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(out.Submissions) != 1 {
		t.Fatalf("len(submissions)=%d, want 1", len(out.Submissions))
	}
	by := out.Submissions[0].SubmittedBy
	if len(by.Members) != 3 {
		t.Errorf("members len=%d, want 3 (showTeamMembers=true)", len(by.Members))
	}
}

func TestListContestSubmissions_FreezePeriod_TeamStandingExcepted(t *testing.T) {
	teamID := "team-001"
	// CallerStandingProvider returns teamID as the caller's standing
	standingProvider := &mockCallerStandingProvider{
		fn: func(_, _ string) (string, bool, error) { return teamID, true, nil },
	}

	var got ContestSubmissionFilters
	uc := NewListContestSubmissionsUseCase(
		repoWith(frozenActiveContest()),
		&mockGroupProvider{findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) { return visibleGroup(), nil }},
		isMemberNotLead(),
		capturingSubsProvider(&got),
		standingProvider,
	)

	if _, err := uc.Execute(context.Background(), baseInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ExceptStandingID != teamID {
		t.Errorf("ExceptStandingID=%q, want %q (own team's post-freeze submissions must appear)", got.ExceptStandingID, teamID)
	}
}
