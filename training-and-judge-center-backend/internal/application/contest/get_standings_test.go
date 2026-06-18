package contest

import (
	"context"
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

// ── helpers ──────────────────────────────────────────────────────────────────

func tpStandings(h, m int) *time.Time {
	t := time.Date(2026, 1, 1, h, m, 0, 0, time.UTC)
	return &t
}

const testStaleness = 30 * time.Second

func newGetStandingsUseCase(
	contest *domainContest.Contest,
	regs []*domainContest.ContestRegistration,
	subs []ContestSubmissionData,
	cache *mockStandingsCache,
	group *GroupInfo,
	memberProvider *mockGroupMemberProvider,
) *GetStandingsUseCase {
	contestRepo := &mockContestRepository{
		findByIDFn: func(_ context.Context, _ string) (*domainContest.Contest, error) {
			return contest, nil
		},
	}
	regRepo := &mockRegistrationRepository{
		listByContestFn: func(_ context.Context, _ string, _, _ int) ([]*domainContest.ContestRegistration, int, error) {
			return regs, len(regs), nil
		},
	}
	subProvider := &mockStandingsSubmissionProvider{
		listByContestFn: func(_ context.Context, _ string) ([]ContestSubmissionData, error) {
			return subs, nil
		},
	}
	groupProvider := &mockGroupProvider{
		findByIDFn: func(_ context.Context, _ string) (*GroupInfo, error) {
			return group, nil
		},
	}
	return NewGetStandingsUseCase(
		contestRepo, regRepo, subProvider,
		groupProvider, memberProvider,
		cache,
		testStaleness,
	)
}

func activeContest() *domainContest.Contest {
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Active Contest"),
		nil,
		time.Now().Add(-2*time.Hour),  // started 2h ago
		time.Now().Add(2*time.Hour),   // ends in 2h
		domainContest.RestorePenalty(20),
		0, // no freeze
		false, false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func finishedContest() *domainContest.Contest {
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Finished Contest"),
		nil,
		time.Now().Add(-4*time.Hour),
		time.Now().Add(-1*time.Hour),
		domainContest.RestorePenalty(20),
		0, false, false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func frozenContest() *domainContest.Contest {
	// freeze = last 60 minutes; contest ends in 30m → freeze started 30m ago
	return domainContest.RestoreContest(
		testContestID,
		domainContest.RestoreContestName("Frozen Contest"),
		nil,
		time.Now().Add(-2*time.Hour),
		time.Now().Add(30*time.Minute), // ends in 30m
		domainContest.RestorePenalty(20),
		60, // freeze last 60 min → freeze started 30m ago
		false, false,
		shared.RestoreGroupID(testGroupID),
		shared.RestoreUserID(callerID),
		domainContest.RestoreParticipationMode("INDIVIDUAL"), domainContest.RestoreTeamSize(2, 5),
		[]domainContest.ContestProblem{},
		testNow, nil,
	)
}

func reg(userID string) *domainContest.ContestRegistration {
	return domainContest.RestoreContestRegistration(
		"reg-"+userID, testContestID, userID, testNow,
	)
}

func sub(userID, problemID, status string, minutesAgo int) ContestSubmissionData {
	return ContestSubmissionData{
		UserID:      userID,
		ProblemID:   problemID,
		Status:      status,
		SubmittedAt: time.Now().Add(-time.Duration(minutesAgo) * time.Minute),
	}
}

func visibleGroup() *GroupInfo {
	return &GroupInfo{ID: testGroupID, Name: "Test Group", IsVisible: true}
}

func hiddenGroup() *GroupInfo {
	return &GroupInfo{ID: testGroupID, Name: "Hidden Group", IsVisible: false}
}

func freshCache(participants []domainContest.ParticipantStanding) *mockStandingsCache {
	cached := &CachedStandings{
		Participants: participants,
		LastUpdated:  time.Now(),
	}
	return &mockStandingsCache{
		getFn: func(_ context.Context, _ string) (*CachedStandings, error) {
			return cached, nil
		},
	}
}

func emptyCache() *mockStandingsCache {
	return &mockStandingsCache{}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestGetStandingsUseCase_CacheHit(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		{ContestantID: callerID, Problems: map[string]domainContest.ProblemAttempt{
			"A": {AcceptedAt: tpStandings(10, 30)},
		}},
	}
	cache := freshCache(participants)
	uc := newGetStandingsUseCase(activeContest(), nil, nil, cache, visibleGroup(), isMemberNotLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 1 {
		t.Errorf("total=%d, want 1", out.Total)
	}
	if out.Entries[0].ContestantID != callerID {
		t.Errorf("contestantID=%q, want %q", out.Entries[0].ContestantID, callerID)
	}
	if out.Entries[0].ProblemsSolved != 1 {
		t.Errorf("problemsSolved=%d, want 1", out.Entries[0].ProblemsSolved)
	}
}

func TestGetStandingsUseCase_CacheMissRebuild(t *testing.T) {
	regs := []*domainContest.ContestRegistration{reg(callerID), reg(otherID)}
	subs := []ContestSubmissionData{
		sub(callerID, "A", "ACCEPTED", 90),
		sub(otherID, "A", "WRONG_ANSWER", 60),
	}
	var setCalled bool
	cache := &mockStandingsCache{
		setFn: func(_ context.Context, _ string, _ *CachedStandings) error {
			setCalled = true
			return nil
		},
	}
	uc := newGetStandingsUseCase(activeContest(), regs, subs, cache, visibleGroup(), isMemberNotLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        1,
		Limit:       50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 2 {
		t.Errorf("total=%d, want 2", out.Total)
	}
	// callerID solved A, otherID did not → callerID ranked first
	if out.Entries[0].ContestantID != callerID {
		t.Errorf("rank1=%q, want %q", out.Entries[0].ContestantID, callerID)
	}
	if out.Entries[1].ProblemsSolved != 0 {
		t.Errorf("otherID solved=%d, want 0", out.Entries[1].ProblemsSolved)
	}
	if !setCalled {
		t.Error("expected cache Set to be called on rebuild")
	}
}

func TestGetStandingsUseCase_FreezeAppliedForRegularUser(t *testing.T) {
	// frozenContest: freeze started 30 min ago, so submission 20 min ago is after freeze
	acceptedAfterFreeze := time.Now().Add(-20 * time.Minute)
	participants := []domainContest.ParticipantStanding{
		{ContestantID: callerID, Problems: map[string]domainContest.ProblemAttempt{
			"A": {AcceptedAt: &acceptedAfterFreeze},
		}},
	}
	uc := newGetStandingsUseCase(frozenContest(), nil, nil, freshCache(participants), visibleGroup(), isMemberNotLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page: 1, Limit: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// problem accepted after freeze → should not count
	if out.Entries[0].ProblemsSolved != 0 {
		t.Errorf("solved=%d, want 0 (should be frozen)", out.Entries[0].ProblemsSolved)
	}
	if !out.Meta.IsFrozen {
		t.Error("meta.IsFrozen should be true")
	}
}

func TestGetStandingsUseCase_RealtimeBypassesFreezeForLead(t *testing.T) {
	acceptedAfterFreeze := time.Now().Add(-20 * time.Minute)
	participants := []domainContest.ParticipantStanding{
		{ContestantID: callerID, Problems: map[string]domainContest.ProblemAttempt{
			"A": {AcceptedAt: &acceptedAfterFreeze},
		}},
	}
	uc := newGetStandingsUseCase(frozenContest(), nil, nil, freshCache(participants), visibleGroup(), isLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Realtime:    true,
		Page: 1, Limit: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// realtime=true for lead → freeze bypassed, problem counts
	if out.Entries[0].ProblemsSolved != 1 {
		t.Errorf("solved=%d, want 1 (realtime bypass)", out.Entries[0].ProblemsSolved)
	}
}

func TestGetStandingsUseCase_RealtimeForbiddenForContestant(t *testing.T) {
	participants := []domainContest.ParticipantStanding{
		{ContestantID: callerID, Problems: map[string]domainContest.ProblemAttempt{}},
	}
	uc := newGetStandingsUseCase(frozenContest(), nil, nil, freshCache(participants), visibleGroup(), isMemberNotLead())

	_, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Realtime:    true,
		Page: 1, Limit: 50,
	})
	if err == nil {
		t.Fatal("expected forbidden error, got nil")
	}
}

func TestGetStandingsUseCase_NotFoundForNonMemberOfHiddenGroup(t *testing.T) {
	uc := newGetStandingsUseCase(activeContest(), nil, nil, emptyCache(), hiddenGroup(), notLead())

	_, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page: 1, Limit: 50,
	})
	if err == nil {
		t.Fatal("expected not found error, got nil")
	}
}

func TestGetStandingsUseCase_Pagination(t *testing.T) {
	participants := make([]domainContest.ParticipantStanding, 5)
	for i := range participants {
		participants[i] = domainContest.ParticipantStanding{
			ContestantID: "user-" + string(rune('A'+i)),
			Problems:     map[string]domainContest.ProblemAttempt{},
		}
	}
	uc := newGetStandingsUseCase(activeContest(), nil, nil, freshCache(participants), visibleGroup(), isLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asCoach(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page:        2,
		Limit:       2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 5 {
		t.Errorf("total=%d, want 5", out.Total)
	}
	if len(out.Entries) != 2 {
		t.Errorf("page entries=%d, want 2", len(out.Entries))
	}
}

func TestGetStandingsUseCase_CompilationErrorNotCountedAsWrongAttempt(t *testing.T) {
	regs := []*domainContest.ContestRegistration{reg(callerID)}
	subs := []ContestSubmissionData{
		sub(callerID, "A", "COMPILATION_ERROR", 90),
		sub(callerID, "A", "ACCEPTED", 60),
	}
	uc := newGetStandingsUseCase(activeContest(), regs, subs, emptyCache(), visibleGroup(), isMemberNotLead())

	out, err := uc.Execute(context.Background(), GetStandingsInput{
		CurrentUser: asContestant(callerID),
		GroupID:     testGroupID,
		ContestID:   testContestID,
		Page: 1, Limit: 50,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	entry := out.Entries[0]
	if entry.ProblemsSolved != 1 {
		t.Errorf("solved=%d, want 1", entry.ProblemsSolved)
	}
	// CE doesn't count as wrong attempt → penalty = only accepted time
	prob := entry.Problems["A"]
	if prob.Attempts != 0 {
		t.Errorf("wrongAttempts=%d, want 0 (CE excluded)", prob.Attempts)
	}
}
