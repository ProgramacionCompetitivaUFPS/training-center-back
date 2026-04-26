package material

import (
	"net/http"
	"strconv"
	"strings"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	currentUser, ok := h.requireCurrentUser(w, r)
	if !ok {
		return
	}

	groupID := r.PathValue("groupId")
	if groupID == "" {
		handler.WriteError(w, apperror.NewBadRequest(apperror.ErrCodeBadRequest, "groupId is required"))
		return
	}

	q := r.URL.Query()

	page, limit, ok := parsePagination(w, q.Get("page"), q.Get("limit"))
	if !ok {
		return
	}

	var pinned *bool
	if raw := q.Get("pinned"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			handler.WriteError(w, apperror.NewValidation([]apperror.FieldError{
				{Field: "pinned", Message: "pinned must be true or false"},
			}))
			return
		}
		pinned = &v
	}

	var tags []string
	if raw := q.Get("tags"); raw != "" {
		for _, t := range strings.Split(raw, ",") {
			if t = strings.TrimSpace(t); t != "" {
				tags = append(tags, t)
			}
		}
	}

	out, err := h.listUC.Execute(r.Context(), appMaterial.ListMaterialsInput{
		CurrentUser: *currentUser,
		GroupID:     groupID,
		Pinned:      pinned,
		Tags:        tags,
		Page:        page,
		Limit:       limit,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	items := make([]materialDetailResponse, 0, len(out.Materials))
	for _, m := range out.Materials {
		items = append(items, buildDetailResponse(m))
	}

	handler.WriteJSON(w, http.StatusOK, listMaterialsResponse{
		Materials: items,
		Pagination: paginationResp{
			TotalCount:   out.Pagination.TotalCount,
			CurrentPage:  out.Pagination.CurrentPage,
			TotalPages:   out.Pagination.TotalPages,
			ItemsPerPage: out.Pagination.ItemsPerPage,
		},
	})
}

func parsePagination(w http.ResponseWriter, rawPage, rawLimit string) (page, limit int, ok bool) {
	page = appMaterial.DefaultPage
	limit = appMaterial.DefaultLimit

	if rawPage != "" {
		v, err := strconv.Atoi(rawPage)
		if err != nil || v < 1 {
			handler.WriteError(w, apperror.NewValidation([]apperror.FieldError{
				{Field: "page", Message: "page must be a positive integer"},
			}))
			return 0, 0, false
		}
		page = v
	}

	if rawLimit != "" {
		v, err := strconv.Atoi(rawLimit)
		if err != nil || v < 1 || v > appMaterial.MaxLimit {
			handler.WriteError(w, apperror.NewValidation([]apperror.FieldError{
				{Field: "limit", Message: "limit must be between 1 and 100"},
			}))
			return 0, 0, false
		}
		limit = v
	}

	return page, limit, true
}
