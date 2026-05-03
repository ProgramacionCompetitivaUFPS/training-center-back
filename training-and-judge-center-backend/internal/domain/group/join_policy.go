package group

import "github.com/training-judge-center/backend/pkg/apperror"

type JoinPolicy struct{ value string }

var (
	JoinPolicyInvite  = JoinPolicy{value: "INVITE"}
	JoinPolicyRequest = JoinPolicy{value: "REQUEST"}
	JoinPolicyOpen    = JoinPolicy{value: "OPEN"}
)

func NewJoinPolicy(s string) (JoinPolicy, error) {
	switch s {
	case "INVITE", "REQUEST", "OPEN":
		return JoinPolicy{value: s}, nil
	}
	return JoinPolicy{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "joinPolicy", Message: "invalid join policy: " + s},
	})
}

func RestoreJoinPolicy(s string) JoinPolicy { return JoinPolicy{value: s} }
func (p JoinPolicy) String() string         { return p.value }
