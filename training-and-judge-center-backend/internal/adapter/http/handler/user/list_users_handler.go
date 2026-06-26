package user

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type listUserItem struct {
	ID            string  `json:"id"`
	Email         *string `json:"email"`
	Name          string  `json:"name"`
	Nickname      string  `json:"nickname"`
	Country       string  `json:"country"`
	City          string  `json:"city"`
	Institution   string  `json:"institution"`
	Role          string  `json:"role"`
	Status        string  `json:"status"`
	CreatedAt     string  `json:"createdAt"`
	UpdatedAt     *string `json:"updatedAt"`
	DeactivatedAt *string `json:"deactivatedAt"`
}

type paginationMeta struct {
	Page        int  `json:"page"`
	Limit       int  `json:"limit"`
	Total       int  `json:"total"`
	TotalPages  int  `json:"totalPages"`
	HasNextPage bool `json:"hasNextPage"`
	HasPrevPage bool `json:"hasPrevPage"`
}

type listUsersResponse struct {
	Users      []listUserItem `json:"users"`
	Pagination paginationMeta `json:"pagination"`
}

// @Summary      List users (admin)
// @Tags         admin
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page number"
// @Param        limit query int false "Items per page"
// @Param        role query string false "Filter by role"
// @Param        status query string false "Filter by status"
// @Success      200 {object} listUsersResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Router       /admin/users [get]
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var roles []string
	if raw := q.Get("role"); raw != "" {
		roles = strings.Split(raw, ",")
	}

	page := appuser.DefaultPage
	if raw := q.Get("page"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
				{Field: "page", Message: "page must be a positive integer"},
			}))
			return
		}
		page = n
	}

	limit := appuser.DefaultLimit
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			handler.WriteError(r.Context(), w, apperror.NewValidation([]apperror.FieldError{
				{Field: "limit", Message: "limit must be a positive integer"},
			}))
			return
		}
		limit = n
	}

	input := appuser.ListUsersInput{
		Roles:       roles,
		Status:      q.Get("status"),
		Country:     q.Get("country"),
		City:        q.Get("city"),
		Institution: q.Get("institution"),
		SearchField: q.Get("searchField"),
		SearchTerm:  q.Get("searchTerm"),
		Sort:        q.Get("sort"),
		Order:       q.Get("order"),
		Page:        page,
		Limit:       limit,
	}

	out, err := h.listUsers.Execute(r.Context(), input)
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	items := make([]listUserItem, 0, len(out.Users))
	for _, u := range out.Users {
		item := listUserItem{
			ID:          u.ID,
			Name:        u.Name,
			Nickname:    u.Nickname,
			Country:     u.Country,
			City:        u.City,
			Institution: u.Institution,
			Role:        u.Role,
			Status:      u.Status,
			CreatedAt:   u.CreatedAt.UTC().Format(time.RFC3339),
		}
		if u.Email != nil {
			item.Email = u.Email
		}
		if u.UpdatedAt != nil {
			s := u.UpdatedAt.UTC().Format(time.RFC3339)
			item.UpdatedAt = &s
		}
		if u.DeactivatedAt != nil {
			s := u.DeactivatedAt.UTC().Format(time.RFC3339)
			item.DeactivatedAt = &s
		}
		items = append(items, item)
	}

	totalPages := 0
	if out.Limit > 0 && out.TotalCount > 0 {
		totalPages = (out.TotalCount + out.Limit - 1) / out.Limit
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, listUsersResponse{
		Users: items,
		Pagination: paginationMeta{
			Page:        out.Page,
			Limit:       out.Limit,
			Total:       out.TotalCount,
			TotalPages:  totalPages,
			HasNextPage: out.Page < totalPages,
			HasPrevPage: out.Page > 1 && totalPages > 0,
		},
	})
}
