package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) ListMyGroups(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	page, err := parseIntParam(r.URL.Query().Get("page"), 1)
	if err != nil {
		writeBadPagination(w, "page", "page must be a positive integer")
		return
	}
	limit, err := parseIntParam(r.URL.Query().Get("limit"), appGroup.DefaultPageLimit)
	if err != nil {
		writeBadPagination(w, "limit", "limit must be an integer")
		return
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	in := appGroup.ListMyGroupsInput{
		CurrentUser: currentUser,
		Role:        queryStringPtr(r.URL.Query().Get("role")),
		Search:      r.URL.Query().Get("search"),
		SortBy:      r.URL.Query().Get("sortBy"),
		Order:       r.URL.Query().Get("order"),
		Page:        page,
		Limit:       limit,
	}

	out, err := h.listMyUC.Execute(r.Context(), in)
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	items := make([]myGroupItemResp, 0, len(out.Groups))
	for _, mg := range out.Groups {
		g := mg.Group
		items = append(items, myGroupItemResp{
			ID:          g.ID(),
			Name:        g.Name().String(),
			Description: g.Description(),
			Visibility:  g.Visibility().String(),
			JoinPolicy:  g.JoinPolicy().String(),
			IsGlobal:    g.IsDefault(),
			MyRole:      string(mg.MyRole),
			JoinedAt:    mg.JoinedAt,
			MemberCount: mg.MemberCount,
			CreatedAt:   g.CreatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	handler.WriteJSON(w, http.StatusOK, listMyGroupsResponse{
		Groups:     items,
		Pagination: buildPagination(out.TotalCount, out.Page, out.TotalPages, out.Limit),
	})
}
