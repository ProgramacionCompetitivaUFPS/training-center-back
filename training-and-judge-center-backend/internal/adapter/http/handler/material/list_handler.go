package material

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const (
	defaultPage  = 1
	defaultLimit = 20
	maxLimit     = 100
)

// @Summary      List materials
// @Tags         materials
// @Produce      json
// @Security     BearerAuth
// @Param        groupId path string true "Group ID"
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Param        pinned query bool false "Filter pinned"
// @Param        tags query string false "Comma-separated tags"
// @Success      200 {object} listMaterialsResponse
// @Failure      401 {object} apperror.AppError
// @Router       /groups/{groupId}/materials [get]
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	if groupID == "" {
		handler.WriteError(r.Context(), w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "groupId is required"))
		return
	}

	q := r.URL.Query()

	page, limit, ok := parsePagination(r.Context(), w, q.Get("page"), q.Get("limit"))
	if !ok {
		return
	}

	var pinned *bool
	if raw := q.Get("pinned"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
				{Field: "pinned", Message: "pinned must be true or false"},
			}))
			return
		}
		pinned = &v
	}

	var tags []string
	if raw := q.Get("tags"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			if tag := strings.TrimSpace(part); tag != "" {
				tags = append(tags, tag)
			}
		}
	}

	query := strings.TrimSpace(q.Get("q"))
	author := strings.TrimSpace(q.Get("author"))
	sort := strings.TrimSpace(q.Get("sort"))

	parseDate := func(field, raw string) (*time.Time, bool) {
		if raw == "" {
			return nil, true
		}
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
				{Field: field, Message: field + " must be a date in YYYY-MM-DD format"},
			}))
			return nil, false
		}
		return &t, true
	}

	publishedFrom, ok2 := parseDate("publishedFrom", q.Get("publishedFrom"))
	if !ok2 {
		return
	}
	publishedTo, ok3 := parseDate("publishedTo", q.Get("publishedTo"))
	if !ok3 {
		return
	}

	out, err := h.listMaterials.Execute(r.Context(), appMaterial.ListMaterialsInput{
		CurrentUser:   *currentUser,
		GroupID:       groupID,
		Query:         query,
		Author:        author,
		PublishedFrom: publishedFrom,
		PublishedTo:   publishedTo,
		Pinned:        pinned,
		Tags:          tags,
		Sort:          sort,
		Page:          page,
		Limit:         limit,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]materialResponse, 0, len(out.Materials))
	for _, m := range out.Materials {
		items = append(items, buildResponse(m))
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listMaterialsResponse{
		Materials: items,
		Pagination: paginationResp{
			TotalCount:   out.Pagination.TotalCount,
			CurrentPage:  out.Pagination.CurrentPage,
			TotalPages:   out.Pagination.TotalPages,
			ItemsPerPage: out.Pagination.ItemsPerPage,
		},
	})
}

func parsePagination(ctx context.Context, w http.ResponseWriter, rawPage, rawLimit string) (page, limit int, ok bool) {
	page = defaultPage
	limit = defaultLimit

	if rawPage != "" {
		v, err := strconv.Atoi(rawPage)
		if err != nil || v < 1 {
			handler.WriteError(ctx, w, apperror.NewValidation([]apperror.FieldError{
				{Field: "page", Message: "page must be a positive integer"},
			}))
			return 0, 0, false
		}
		page = v
	}

	if rawLimit != "" {
		v, err := strconv.Atoi(rawLimit)
		if err != nil || v < 1 || v > maxLimit {
			handler.WriteError(ctx, w, apperror.NewValidation([]apperror.FieldError{
				{Field: "limit", Message: fmt.Sprintf("limit must be between 1 and %d", maxLimit)},
			}))
			return 0, 0, false
		}
		limit = v
	}

	return page, limit, true
}
