package user

import "fmt"

type RequestStatus string

const (
	StatusPending RequestStatus = "PENDING"
	StatusUsed    RequestStatus = "USED"
	StatusExpired RequestStatus = "EXPIRED"
)

func NewRequestStatus(s string) (RequestStatus, error) {
	switch RequestStatus(s) {
	case StatusPending, StatusUsed, StatusExpired:
		return RequestStatus(s), nil
	}
	return "", fmt.Errorf("invalid request status: %q", s)
}

func RestoreRequestStatus(s string) RequestStatus {
	return RequestStatus(s)
}
