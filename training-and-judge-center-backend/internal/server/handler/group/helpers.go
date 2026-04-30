package group

import (
	"net/http"
	"net/url"
	"strconv"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	domainGroup "github.com/training-judge-center/backend/internal/domain/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
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

func parsePaginationParams(q url.Values, w http.ResponseWriter) (page, limit int, ok bool) {
	var err error
	page, err = parseIntParam(q.Get("page"), 1)
	if err != nil {
		writeBadPagination(w, "page", "page must be a positive integer")
		return 0, 0, false
	}
	limit, err = parseIntParam(q.Get("limit"), appGroup.DefaultPageLimit)
	if err != nil {
		writeBadPagination(w, "limit", "limit must be an integer")
		return 0, 0, false
	}
	return page, limit, true
}

func writeBadPagination(w http.ResponseWriter, field, msg string) {
	handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
		Code:    apperror.ErrCodeValidationError,
		Message: "Invalid request parameters",
		Details: []apperror.FieldError{{Field: field, Message: msg}},
	})
}

func memberRoleToStringPtr(r *domainGroup.MemberRole) *string {
	if r == nil {
		return nil
	}
	s := string(*r)
	return &s
}

func memberRoleValueToStringPtr(r domainGroup.MemberRole) *string {
	if r == "" {
		return nil
	}
	s := string(r)
	return &s
}

func userIDPtrToStringPtr(uid *shared.UserID) *string {
	if uid == nil {
		return nil
	}
	v := uid.Value()
	return &v
}
