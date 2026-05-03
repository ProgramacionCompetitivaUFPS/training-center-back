package group

import "github.com/training-judge-center/backend/pkg/apperror"

type JoinMethod string

const (
	JoinMethodDirectAdd       JoinMethod = "DIRECT_ADD"
	JoinMethodInvitation      JoinMethod = "INVITATION"
	JoinMethodRequestApproved JoinMethod = "REQUEST_APPROVED"
	JoinMethodOpenJoin        JoinMethod = "OPEN_JOIN"
)

func NewJoinMethod(s string) (JoinMethod, error) {
	switch JoinMethod(s) {
	case JoinMethodDirectAdd, JoinMethodInvitation,
		JoinMethodRequestApproved, JoinMethodOpenJoin:
		return JoinMethod(s), nil
	}
	return "", apperror.NewInternal()
}

func RestoreJoinMethod(s string) JoinMethod { return JoinMethod(s) }
func (j JoinMethod) String() string         { return string(j) }
