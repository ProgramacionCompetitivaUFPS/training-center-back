package group

import "github.com/training-judge-center/backend/pkg/apperror"

const (
	joinMethodDirectAdd       = "DIRECT_ADD"
	joinMethodInvitation      = "INVITATION"
	joinMethodRequestApproved = "REQUEST_APPROVED"
	joinMethodOpenJoin        = "OPEN_JOIN"
)

type JoinMethod struct{ value string }

var (
	JoinMethodDirectAdd       = JoinMethod{value: joinMethodDirectAdd}
	JoinMethodInvitation      = JoinMethod{value: joinMethodInvitation}
	JoinMethodRequestApproved = JoinMethod{value: joinMethodRequestApproved}
	JoinMethodOpenJoin        = JoinMethod{value: joinMethodOpenJoin}
)

func NewJoinMethod(s string) (JoinMethod, error) {
	switch s {
	case joinMethodDirectAdd, joinMethodInvitation, joinMethodRequestApproved, joinMethodOpenJoin:
		return JoinMethod{value: s}, nil
	}
	return JoinMethod{}, apperror.NewValidation([]apperror.FieldError{
		{Field: "joinMethod", Message: "invalid join method: " + s},
	})
}

func RestoreJoinMethod(s string) JoinMethod { return JoinMethod{value: s} }
func (j JoinMethod) String() string         { return j.value }
