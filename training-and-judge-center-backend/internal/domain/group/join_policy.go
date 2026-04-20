package group

import "github.com/training-judge-center/backend/pkg/apperror"

type JoinPolicy string

const (
	JoinPolicyInvite  JoinPolicy = "INVITE"
	JoinPolicyRequest JoinPolicy = "REQUEST"
	JoinPolicyOpen    JoinPolicy = "OPEN"
)

func NewJoinPolicy(s string) (JoinPolicy, error) {
	switch JoinPolicy(s) {
	case JoinPolicyInvite, JoinPolicyRequest, JoinPolicyOpen:
		return JoinPolicy(s), nil
	}
	return "", apperror.NewBadRequest(ErrCodeInvalidJoinPolicy, "invalid join policy: "+s)
}

func RestoreJoinPolicy(s string) JoinPolicy { return JoinPolicy(s) }
func (p JoinPolicy) String() string         { return string(p) }
