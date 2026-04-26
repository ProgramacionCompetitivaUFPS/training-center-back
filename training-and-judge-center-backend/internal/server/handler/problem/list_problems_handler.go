package problem

import (
	"net/http"
	"strconv"
	"strings"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func (h *Handler) ListProblems(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	page, err := parseIntParam(r.URL.Query().Get("page"), 1)
	if err != nil || page < 1 {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request parameters",
			Details: []apperror.FieldError{{Field: "page", Message: "Page must be a positive integer"}},
		})
		return
	}

	limit, err := parseIntParam(r.URL.Query().Get("limit"), 20)
	if err != nil || limit < 1 || limit > 100 {
		handler.WriteJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    apperror.ErrCodeValidationError,
			Message: "Invalid request parameters",
			Details: []apperror.FieldError{{Field: "limit", Message: "Limit must be between 1 and 100"}},
		})
		return
	}

	var tags []string
	if raw := r.URL.Query().Get("tags"); raw != "" {
		tags = strings.Split(raw, ",")
	}

	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}

	in := appProblem.ListProblemsInput{
		CurrentUser:    currentUser,
		Tags:           tags,
		Accessibility:  queryStringPtr(r.URL.Query().Get("accessibility")),
		Status:         queryStringPtr(r.URL.Query().Get("status")),
		AuthorNickname: queryStringPtr(r.URL.Query().Get("author")),
		Page:           page,
		Limit:          limit,
	}

	out, ucErr := h.listProblemsUC.Execute(r.Context(), in)
	if ucErr != nil {
		handler.WriteError(w, ucErr)
		return
	}

	items := make([]problemListItemResp, 0, len(out.Problems))
	for _, s := range out.Problems {
		p := s.Problem
		items = append(items, problemListItemResp{
			Slug:          p.Slug().String(),
			Title:         p.Title().String(),
			Tags:          p.Tags().Values(),
			Status:        p.Status().String(),
			Accessibility: p.Accessibility().String(),
			Author:        authorResp{Nickname: s.Author.Nickname, Name: s.Author.Name},
			CreatedAt:     p.CreatedAt().Format("2006-01-02T15:04:05Z"),
			UpdatedAt:     p.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		})
	}

	handler.WriteJSON(w, http.StatusOK, listProblemsResponse{
		Problems: items,
		Pagination: paginationResp{
			TotalCount:   out.TotalCount,
			CurrentPage:  out.Page,
			TotalPages:   out.TotalPages,
			ItemsPerPage: out.Limit,
		},
	})
}

func parseIntParam(raw string, defaultVal int) (int, error) {
	if raw == "" {
		return defaultVal, nil
	}
	return strconv.Atoi(raw)
}

func queryStringPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
