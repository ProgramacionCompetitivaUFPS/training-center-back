package problem

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// @Summary      Get problem
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} getProblemResponse
// @Failure      401 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug} [get]
func (h *Handler) GetProblem(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteJSON(w, http.StatusUnauthorized, apperror.AppError{Code: apperror.ErrCodeUnauthorized, Message: "Invalid or missing authentication token"})
		return
	}

	slug := r.PathValue("slug")
	currentUser := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}

	out, err := h.getProblemUC.Execute(r.Context(), appProblem.GetProblemInput{
		Slug:        slug,
		CurrentUser: currentUser,
	})
	if err != nil {
		handler.WriteError(w, err)
		return
	}

	p := out.Problem
	overrides := make([]langOverrideResp, 0, len(p.LangOverrides()))
	for _, lo := range p.LangOverrides() {
		overrides = append(overrides, langOverrideResp{
			Language:    lo.Language(),
			TimeLimit:   lo.TimeLimit(),
			MemoryLimit: lo.MemoryLimit(),
		})
	}

	var tl *int
	if p.TimeLimit() != nil {
		v := p.TimeLimit().Milliseconds()
		tl = &v
	}
	var ml *int
	if p.MemoryLimit() != nil {
		v := p.MemoryLimit().Megabytes()
		ml = &v
	}

	var judgingUpdatedAt *string
	if p.JudgingUpdatedAt() != nil {
		s := p.JudgingUpdatedAt().Format("2006-01-02T15:04:05Z")
		judgingUpdatedAt = &s
	}

	resp := getProblemResponse{
		Slug:                    p.Slug().String(),
		Title:                   p.Title().String(),
		Statement:               p.Statement().Value(),
		TimeLimit:               tl,
		MemoryLimit:             ml,
		LangOverrides:           overrides,
		Tags:                    p.Tags().Values(),
		Status:                  p.Status().String(),
		Accessibility:           p.Accessibility().String(),
		Author:                  authorResp{Nickname: out.Author.Nickname, Name: out.Author.Name},
		CreatedAt:               p.CreatedAt().Format("2006-01-02T15:04:05Z"),
		UpdatedAt:               p.UpdatedAt().Format("2006-01-02T15:04:05Z"),
		ProblemJudgingUpdatedAt: judgingUpdatedAt,
	}

	if out.Modifiers != nil {
		modifiers := make([]modifierResp, 0, len(out.Modifiers))
		for _, m := range out.Modifiers {
			modifiers = append(modifiers, modifierResp{Nickname: m.Nickname, Name: m.Name})
		}
		resp.Modifiers = modifiers

		if out.Files != nil {
			solutions := make([]solutionResp, 0, len(out.Files.Solutions))
			for _, sol := range out.Files.Solutions {
				solutions = append(solutions, solutionResp{Filename: sol.Filename, Language: sol.Language})
			}
			resp.Files = &filesResp{
				TestCases: out.Files.TestCases,
				Solutions: solutions,
				Checker:   out.Files.Checker,
				Validator: out.Files.Validator,
			}
		}
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}
