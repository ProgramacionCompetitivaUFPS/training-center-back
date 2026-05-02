package user

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/internal/server/handler"
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
	TotalCount   int `json:"totalCount"`
	CurrentPage  int `json:"currentPage"`
	TotalPages   int `json:"totalPages"`
	ItemsPerPage int `json:"itemsPerPage"`
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
func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var roles []string
	if raw := q.Get("role"); raw != "" {
		roles = strings.Split(raw, ",")
	}

	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

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

	result, err := h.listUsers.Execute(r.Context(), input)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	items := make([]listUserItem, 0, len(result.Users))
	for _, u := range result.Users {
		item := listUserItem{
			ID:          u.ID,
			Name:        u.Name,
			Nickname:    u.Nickname,
			Country:     u.Country,
			City:        u.City,
			Institution: u.Institution,
			Role:        u.Role,
			Status:      u.Status,
			CreatedAt:   u.CreatedAt.Format("2006-01-02T15:04:05Z"),
		}
		if u.Email != nil {
			item.Email = u.Email
		}
		if u.UpdatedAt != nil {
			s := u.UpdatedAt.Format("2006-01-02T15:04:05Z")
			item.UpdatedAt = &s
		}
		if u.DeactivatedAt != nil {
			s := u.DeactivatedAt.Format("2006-01-02T15:04:05Z")
			item.DeactivatedAt = &s
		}
		items = append(items, item)
	}

	totalPages := 1
	if result.Limit > 0 && result.TotalCount > 0 {
		totalPages = int(math.Ceil(float64(result.TotalCount) / float64(result.Limit)))
	}

	handler.WriteJSON(w, http.StatusOK, listUsersResponse{
		Users: items,
		Pagination: paginationMeta{
			TotalCount:   result.TotalCount,
			CurrentPage:  result.Page,
			TotalPages:   totalPages,
			ItemsPerPage: result.Limit,
		},
	})
}
