package contest

import (
	"testing"
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

var postEnd = time.Date(2026, 1, 1, 13, 0, 0, 0, time.UTC)

func postSub(userID, problemID, status string, at time.Time) ContestSubmissionData {
	return ContestSubmissionData{UserID: userID, ProblemID: problemID, Status: status, SubmittedAt: at}
}

func runPostcompetition(subs []ContestSubmissionData, userToContestant, types map[string]string) ([]domainContest.ParticipantStanding, []string) {
	return buildPostcompetition(subs, postEnd, map[string]bool{"p1": true, "p2": true}, userToContestant, types)
}

func findPostRow(t *testing.T, participants []domainContest.ParticipantStanding, id string) domainContest.ParticipantStanding {
	t.Helper()
	for _, p := range participants {
		if p.ContestantID == id {
			return p
		}
	}
	t.Fatalf("no postcompetition row for %q", id)
	return domainContest.ParticipantStanding{}
}

func TestRecordAttempt_CountsOnlyCountingVerdicts(t *testing.T) {
	at := *tp(13, 1)
	cases := []struct {
		status     string
		wantWrong  int
		wantSolved bool
	}{
		{"ACCEPTED", 0, true},
		{"WRONG_ANSWER", 1, false},
		{"TIME_LIMIT_EXCEEDED", 1, false},
		{"MEMORY_LIMIT_EXCEEDED", 1, false},
		{"OUTPUT_LIMIT_EXCEEDED", 1, false},
		{"RUNTIME_ERROR", 1, false},
		{"COMPILATION_ERROR", 0, false},
		{"SYSTEM_ERROR", 0, false},
	}
	for _, c := range cases {
		got := recordAttempt(domainContest.ProblemAttempt{}, postSub("u1", "p1", c.status, at))
		if len(got.WrongAttemptTimes) != c.wantWrong {
			t.Errorf("%s: wrong attempts = %d, want %d", c.status, len(got.WrongAttemptTimes), c.wantWrong)
		}
		if (got.AcceptedAt != nil) != c.wantSolved {
			t.Errorf("%s: solved = %v, want %v", c.status, got.AcceptedAt != nil, c.wantSolved)
		}
	}
}

func TestRecordAttempt_IgnoresSubmissionsAfterAccept(t *testing.T) {
	accepted := tp(13, 1)
	before := domainContest.ProblemAttempt{AcceptedAt: accepted}
	got := recordAttempt(before, postSub("u1", "p1", "WRONG_ANSWER", *tp(13, 2)))
	if len(got.WrongAttemptTimes) != 0 || got.AcceptedAt != accepted {
		t.Errorf("attempt after accept changed: %+v", got)
	}
}

func TestPostcompetition_RegisteredUserWithOnlyWrongAnswers_HasRowWithoutAccept(t *testing.T) {
	subs := []ContestSubmissionData{
		postSub("u1", "p1", "WRONG_ANSWER", *tp(13, 10)),
		postSub("u1", "p1", "WRONG_ANSWER", *tp(13, 20)),
	}
	participants, extra := runPostcompetition(subs, map[string]string{"u1": "u1"}, map[string]string{"u1": "INDIVIDUAL"})

	row := findPostRow(t, participants, "u1")
	attempt := row.Problems["p1"]
	if attempt.AcceptedAt != nil {
		t.Errorf("AcceptedAt = %v, want nil", attempt.AcceptedAt)
	}
	if len(attempt.WrongAttemptTimes) != 2 {
		t.Errorf("wrong attempts = %d, want 2", len(attempt.WrongAttemptTimes))
	}
	if len(extra) != 0 {
		t.Errorf("extraUserIDs = %v, want none", extra)
	}
}

func TestPostcompetition_UnregisteredUser_GetsIndividualRowAndProfileID(t *testing.T) {
	subs := []ContestSubmissionData{postSub("u9", "p1", "ACCEPTED", *tp(13, 5))}
	participants, extra := runPostcompetition(subs, map[string]string{}, map[string]string{})

	row := findPostRow(t, participants, "u9")
	if row.ParticipantType != "INDIVIDUAL" {
		t.Errorf("type = %q, want INDIVIDUAL", row.ParticipantType)
	}
	if row.Problems["p1"].AcceptedAt == nil {
		t.Error("expected p1 accepted")
	}
	if len(extra) != 1 || extra[0] != "u9" {
		t.Errorf("extraUserIDs = %v, want [u9]", extra)
	}
}

func TestPostcompetition_TeamMemberSubmission_CountsForTeam(t *testing.T) {
	subs := []ContestSubmissionData{postSub("m1", "p1", "ACCEPTED", *tp(13, 5))}
	participants, extra := runPostcompetition(subs,
		map[string]string{"m1": "team1"},
		map[string]string{"team1": "TEAM"})

	row := findPostRow(t, participants, "team1")
	if row.ParticipantType != "TEAM" {
		t.Errorf("type = %q, want TEAM", row.ParticipantType)
	}
	if len(participants) != 1 {
		t.Errorf("rows = %d, want 1", len(participants))
	}
	if len(extra) != 0 {
		t.Errorf("extraUserIDs = %v, want none", extra)
	}
}

func TestPostcompetition_ProblemOutsideContest_Ignored(t *testing.T) {
	subs := []ContestSubmissionData{postSub("u1", "p2", "ACCEPTED", *tp(13, 5))}
	participants, _ := buildPostcompetition(subs, postEnd, map[string]bool{"p1": true}, map[string]string{"u1": "u1"}, map[string]string{"u1": "INDIVIDUAL"})
	if len(participants) != 0 {
		t.Errorf("rows = %d, want 0", len(participants))
	}
}

func TestPostcompetition_CompilationAndSystemErrors_CreateNoRow(t *testing.T) {
	subs := []ContestSubmissionData{
		postSub("u1", "p1", "COMPILATION_ERROR", *tp(13, 5)),
		postSub("u1", "p1", "SYSTEM_ERROR", *tp(13, 6)),
	}
	participants, extra := runPostcompetition(subs, map[string]string{"u1": "u1"}, map[string]string{"u1": "INDIVIDUAL"})
	if len(participants) != 0 {
		t.Errorf("rows = %d, want 0", len(participants))
	}
	if len(extra) != 0 {
		t.Errorf("extraUserIDs = %v, want none", extra)
	}
}

func TestPostcompetition_SubmissionAtEndTime_IsOfficialNotPost(t *testing.T) {
	subs := []ContestSubmissionData{postSub("u9", "p1", "ACCEPTED", postEnd)}
	participants, extra := runPostcompetition(subs, map[string]string{}, map[string]string{})
	if len(participants) != 0 || len(extra) != 0 {
		t.Errorf("rows = %d extra = %v, want none", len(participants), extra)
	}
}

func TestPostcompetition_SubmissionBeforeEnd_Ignored(t *testing.T) {
	subs := []ContestSubmissionData{postSub("u9", "p1", "ACCEPTED", *tp(12, 0))}
	participants, extra := runPostcompetition(subs, map[string]string{}, map[string]string{})
	if len(participants) != 0 || len(extra) != 0 {
		t.Errorf("rows = %d extra = %v, want none", len(participants), extra)
	}
}

func TestPostcompetition_SubmissionsAfterAccept_NotCounted(t *testing.T) {
	subs := []ContestSubmissionData{
		postSub("u1", "p1", "ACCEPTED", *tp(13, 5)),
		postSub("u1", "p1", "WRONG_ANSWER", *tp(13, 10)),
	}
	participants, _ := runPostcompetition(subs, map[string]string{"u1": "u1"}, map[string]string{"u1": "INDIVIDUAL"})
	attempt := findPostRow(t, participants, "u1").Problems["p1"]
	if attempt.AcceptedAt == nil || !attempt.AcceptedAt.Equal(*tp(13, 5)) {
		t.Errorf("AcceptedAt = %v, want 13:05", attempt.AcceptedAt)
	}
	if len(attempt.WrongAttemptTimes) != 0 {
		t.Errorf("wrong attempts = %d, want 0", len(attempt.WrongAttemptTimes))
	}
}

func TestPostcompetition_WrongThenAccept_CountsWrongAttemptsBeforeAccept(t *testing.T) {
	subs := []ContestSubmissionData{
		postSub("u1", "p1", "WRONG_ANSWER", *tp(13, 1)),
		postSub("u1", "p1", "WRONG_ANSWER", *tp(13, 2)),
		postSub("u1", "p1", "ACCEPTED", *tp(13, 3)),
	}
	participants, _ := runPostcompetition(subs, map[string]string{"u1": "u1"}, map[string]string{"u1": "INDIVIDUAL"})
	attempt := findPostRow(t, participants, "u1").Problems["p1"]
	if len(attempt.WrongAttemptTimes) != 2 {
		t.Errorf("wrong attempts = %d, want 2", len(attempt.WrongAttemptTimes))
	}
	if attempt.AcceptedAt == nil || !attempt.AcceptedAt.Equal(*tp(13, 3)) {
		t.Errorf("AcceptedAt = %v, want 13:03", attempt.AcceptedAt)
	}
}

func TestPostcompetition_AcceptOnFirstTry_HasNoWrongAttempts(t *testing.T) {
	subs := []ContestSubmissionData{postSub("u1", "p1", "ACCEPTED", *tp(13, 1))}
	participants, _ := runPostcompetition(subs, map[string]string{"u1": "u1"}, map[string]string{"u1": "INDIVIDUAL"})
	attempt := findPostRow(t, participants, "u1").Problems["p1"]
	if len(attempt.WrongAttemptTimes) != 0 || attempt.AcceptedAt == nil {
		t.Errorf("attempt = %+v, want accepted with 0 wrong", attempt)
	}
}
