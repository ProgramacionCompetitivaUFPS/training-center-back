package group

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appGroup "github.com/training-judge-center/backend/internal/application/group"
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

func parsePaginationParams(ctx context.Context, q url.Values, w http.ResponseWriter) (page, limit int, ok bool) {
	var err error
	page, err = parseIntParam(q.Get("page"), 1)
	if err != nil {
		writeBadPagination(ctx, w, "page", "page must be a positive integer")
		return 0, 0, false
	}
	limit, err = parseIntParam(q.Get("limit"), appGroup.DefaultPageLimit)
	if err != nil {
		writeBadPagination(ctx, w, "limit", "limit must be an integer")
		return 0, 0, false
	}
	return page, limit, true
}

func writeBadPagination(ctx context.Context, w http.ResponseWriter, field, msg string) {
	handler.WriteJSON(ctx, w, http.StatusBadRequest, apperror.AppError{
		Code:    apperror.ErrCodeValidationError,
		Message: "Invalid request parameters",
		Details: []apperror.FieldError{{Field: field, Message: msg}},
	})
}
