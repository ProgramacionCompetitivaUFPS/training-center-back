package group

import (
	"strconv"

	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
)

func parseIntParam(raw string, defaultVal int) (int, error) {
	if raw == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(raw)
}

func stringPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func memberRoleToStringPtr(r *domainGroup.MemberRole) *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}
