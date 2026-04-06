package user

type RequestStatus string

const (
	StatusPending RequestStatus = "PENDING"
	StatusUsed    RequestStatus = "USED"
	StatusExpired RequestStatus = "EXPIRED"
)
