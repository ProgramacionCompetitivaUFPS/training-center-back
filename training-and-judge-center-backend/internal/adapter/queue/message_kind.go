package queue

import (
	"encoding/json"
	"fmt"
)

// messageKind identifies which business meaning a queue message carries.
// The set of valid values is closed to kindSubmission/kindProblemValidation —
// no other value can ever exist, not even from a malformed message on the wire.
type messageKind struct {
	value string
}

var (
	kindSubmission        = messageKind{value: "SUBMISSION"}
	kindProblemValidation = messageKind{value: "PROBLEM_VALIDATION"}
)

// allMessageKinds enumerates every valid kind — Consume checks against this
// so a forgotten handler fails loudly at startup instead of silently
// dropping messages of that kind forever. Add a new kind here too.
var allMessageKinds = []messageKind{kindSubmission, kindProblemValidation}

func (k messageKind) String() string { return k.value }

func (k messageKind) MarshalJSON() ([]byte, error) {
	return json.Marshal(k.value)
}

func (k *messageKind) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	switch raw {
	case kindSubmission.value:
		*k = kindSubmission
	case kindProblemValidation.value:
		*k = kindProblemValidation
	default:
		return fmt.Errorf("rabbitmq: unknown message kind %q", raw)
	}
	return nil
}
