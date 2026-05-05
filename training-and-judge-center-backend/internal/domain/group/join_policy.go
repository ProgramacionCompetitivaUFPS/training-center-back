package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	joinPolicyInvite  = "INVITE"
	joinPolicyRequest = "REQUEST"
	joinPolicyOpen    = "OPEN"
)

type JoinPolicy struct{ value string }

var (
	JoinPolicyInvite  = JoinPolicy{value: joinPolicyInvite}
	JoinPolicyRequest = JoinPolicy{value: joinPolicyRequest}
	JoinPolicyOpen    = JoinPolicy{value: joinPolicyOpen}
)

func NewJoinPolicy(s string) (JoinPolicy, error) {
	switch s {
	case joinPolicyInvite, joinPolicyRequest, joinPolicyOpen:
		return JoinPolicy{value: s}, nil
	}
	return JoinPolicy{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "joinPolicy", Message: "invalid join policy: " + s},
	})
}

func RestoreJoinPolicy(s string) JoinPolicy { return JoinPolicy{value: s} }
func (p JoinPolicy) String() string         { return p.value }
