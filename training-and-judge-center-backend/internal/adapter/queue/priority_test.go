package queue

import (
	"testing"

	appproblem "github.com/training-judge-center/backend/internal/application/problem"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// The problem package cannot import the submission one, so this is what keeps a validation and a
// contest submission in the same tier: served in the order they arrive, neither jumping the other.
func TestPriorities_AValidationIsInTheSameTierAsAContestSubmission(t *testing.T) {
	if appproblem.QueuePriorityPublishValidation != appsubmission.QueuePriorityContest {
		t.Errorf("a validation has priority %d and a contest submission %d: they must be equal",
			appproblem.QueuePriorityPublishValidation, appsubmission.QueuePriorityContest)
	}
}

func TestPriorities_AValidationStillComesBeforeEverythingLowerThanAContest(t *testing.T) {
	for name, lower := range map[string]int{
		"post-contest": appsubmission.QueuePriorityPostContest,
		"practice":     appsubmission.QueuePriorityPractice,
		"rejudge":      appsubmission.QueuePriorityRejudge,
	} {
		if appproblem.QueuePriorityPublishValidation <= lower {
			t.Errorf("a validation (%d) must come before %s (%d)", appproblem.QueuePriorityPublishValidation, name, lower)
		}
	}
}

// The live queue was declared with x-max-priority 5. RabbitMQ refuses to declare it again with another
// value, so lowering this would break startup, and no tier in use may go above it.
func TestPriorities_TheQueueKeepsTheMaximumItWasDeclaredWith(t *testing.T) {
	if maxPriority != 5 {
		t.Errorf("maxPriority = %d: the queue already exists with 5 and cannot be redeclared with another value", maxPriority)
	}
	for name, p := range map[string]int{
		"validation": appproblem.QueuePriorityPublishValidation,
		"contest":    appsubmission.QueuePriorityContest,
	} {
		if p > maxPriority {
			t.Errorf("%s has priority %d, above the queue's maximum %d", name, p, maxPriority)
		}
	}
}
