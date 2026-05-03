package group

import domainGroup "github.com/training-judge-center/backend/internal/domain/group"

type JoinRequestDetail struct {
	Request *domainGroup.JoinRequest
	Display *UserDisplay
}
