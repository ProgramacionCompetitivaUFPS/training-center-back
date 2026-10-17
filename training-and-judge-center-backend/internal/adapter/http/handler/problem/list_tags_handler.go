package problem

import (
	"net/http"
	"sort"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type listProblemTagsResponse struct {
	Tags []string `json:"tags"`
}

// ListTags handles GET /problems/tags.
//
// Unlike materials' tag suggestions (a sample of recent data), this is the platform's
// closed vocabulary: CreateProblem/UpdateProblem reject any tag not in this exact set
// (domain/problem/tags.go, NewTags). The frontend must source its suggestions from here
// instead of hardcoding a list that can drift from config/virtual_object.json.
//
// @Summary      List allowed problem tags
// @Description  Returns the platform's fixed tag vocabulary (config/virtual_object.json).
// @Description  A tag outside this list is rejected by CreateProblem/UpdateProblem.
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} listProblemTagsResponse
// @Failure      401 {object} apperror.AppError
// @Router       /problems/tags [get]
func (h *Handler) ListTags(w http.ResponseWriter, r *http.Request) {
	if _, ok := middleware.GetCurrentUser(r.Context()); !ok {
		handler.WriteJSON(r.Context(), w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	allowed := h.settings.AllowedTags()
	tags := make([]string, 0, len(allowed))
	for t := range allowed {
		tags = append(tags, t)
	}
	sort.Strings(tags)

	handler.WriteJSON(r.Context(), w, http.StatusOK, listProblemTagsResponse{Tags: tags})
}
