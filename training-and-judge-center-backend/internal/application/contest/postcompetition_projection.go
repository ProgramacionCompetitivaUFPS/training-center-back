package contest

import (
	"time"

	domainContest "github.com/training-judge-center/backend/internal/domain/contest"
)

func recordAttempt(a domainContest.ProblemAttempt, sub ContestSubmissionData) domainContest.ProblemAttempt {
	if a.AcceptedAt != nil {
		return a
	}
	switch sub.Status {
	case "ACCEPTED":
		t := sub.SubmittedAt
		a.AcceptedAt = &t
	// Literal strings rather than the domain type, so a verdict added there
	// does not reach this list on its own — see the test that pins them.
	case "WRONG_ANSWER", "TIME_LIMIT_EXCEEDED", "MEMORY_LIMIT_EXCEEDED",
		"OUTPUT_LIMIT_EXCEEDED", "RUNTIME_ERROR":
		a.WrongAttemptTimes = append(a.WrongAttemptTimes, sub.SubmittedAt)
	}
	return a
}

func attemptChanged(before, after domainContest.ProblemAttempt) bool {
	return after.AcceptedAt != before.AcceptedAt || len(after.WrongAttemptTimes) != len(before.WrongAttemptTimes)
}

// buildPostcompetition projects the submissions made after endTime. Unknown
// submitters are individual rows keyed by their own user ID and are returned
// in extraUserIDs so their profiles get loaded.
func buildPostcompetition(
	subs []ContestSubmissionData,
	endTime time.Time,
	inContest map[string]bool,
	userToContestant map[string]string,
	participantTypeOf map[string]string,
) ([]domainContest.ParticipantStanding, []string) {
	rows := make(map[string]*domainContest.ParticipantStanding)
	var extraUserIDs []string

	for _, sub := range subs {
		if !sub.SubmittedAt.After(endTime) || !inContest[sub.ProblemID] {
			continue
		}
		contestantID, known := userToContestant[sub.UserID]
		if !known {
			contestantID = sub.UserID
		}

		p := rows[contestantID]
		var before domainContest.ProblemAttempt
		if p != nil {
			before = p.Problems[sub.ProblemID]
		}
		after := recordAttempt(before, sub)
		if !attemptChanged(before, after) {
			continue
		}

		if p == nil {
			participantType := "INDIVIDUAL"
			if known {
				participantType = participantTypeOf[contestantID]
			} else {
				extraUserIDs = append(extraUserIDs, contestantID)
			}
			p = &domainContest.ParticipantStanding{
				ContestantID:    contestantID,
				ParticipantType: participantType,
				Problems:        make(map[string]domainContest.ProblemAttempt),
			}
			rows[contestantID] = p
		}
		p.Problems[sub.ProblemID] = after
	}

	participants := make([]domainContest.ParticipantStanding, 0, len(rows))
	for _, p := range rows {
		participants = append(participants, *p)
	}
	return participants, extraUserIDs
}
