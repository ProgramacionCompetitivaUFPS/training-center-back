package group

type JoinMethod string

const (
	JoinMethodDirectAdd       JoinMethod = "DIRECT_ADD"
	JoinMethodInvitation      JoinMethod = "INVITATION"
	JoinMethodRequestApproved JoinMethod = "REQUEST_APPROVED"
	JoinMethodOpenJoin        JoinMethod = "OPEN_JOIN"
)

func RestoreJoinMethod(s string) JoinMethod { return JoinMethod(s) }
