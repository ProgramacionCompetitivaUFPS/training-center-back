package problem

import (
	"time"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type langOverrideRequest struct {
	Language    string `json:"language"`
	TimeLimit   *int   `json:"timeLimit"`
	MemoryLimit *int   `json:"memoryLimit"`
}

type langOverrideResp struct {
	Language    string `json:"language"`
	TimeLimit   *int   `json:"timeLimit,omitempty"`
	MemoryLimit *int   `json:"memoryLimit,omitempty"`
}

type authorResp struct {
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type solutionResp struct {
	Filename string `json:"filename"`
	Language string `json:"language"`
}

type filesResp struct {
	TestCases bool           `json:"testCases"`
	Solutions []solutionResp `json:"solutions"`
	Checker   bool           `json:"checker"`
	Validator bool           `json:"validator"`
}

type modifierResp struct {
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type getProblemResponse struct {
	Slug                    string             `json:"slug"`
	Title                   string             `json:"title"`
	Statement               *string            `json:"statement"`
	TimeLimit               *int               `json:"timeLimit"`
	MemoryLimit             *int               `json:"memoryLimit"`
	LangOverrides           []langOverrideResp `json:"languageOverrides"`
	Tags                    []string           `json:"tags"`
	Status                  string             `json:"status"`
	Accessibility           string             `json:"accessibility"`
	Author                  authorResp         `json:"author"`
	Modifiers               []modifierResp     `json:"modifiers,omitempty"`
	Files                   *filesResp         `json:"files,omitempty"`
	CreatedAt               string             `json:"createdAt"`
	UpdatedAt               string             `json:"updatedAt"`
	ProblemJudgingUpdatedAt *string            `json:"problemJudgingUpdatedAt"`
}

type problemListItemResp struct {
	Slug          string     `json:"slug"`
	Title         string     `json:"title"`
	Tags          []string   `json:"tags"`
	Status        string     `json:"status"`
	Accessibility string     `json:"accessibility"`
	Author        authorResp `json:"author"`
	CreatedAt     string     `json:"createdAt"`
	UpdatedAt     string     `json:"updatedAt"`
}

type addModifierRequest struct {
	UserID string `json:"userId"`
}

type listModifiersResponse struct {
	Modifiers []modifierResp `json:"modifiers"`
}

type paginationResp struct {
	Page        int  `json:"page"`
	Limit       int  `json:"limit"`
	Total       int  `json:"total"`
	TotalPages  int  `json:"totalPages"`
	HasNextPage bool `json:"hasNextPage"`
	HasPrevPage bool `json:"hasPrevPage"`
}

type listProblemsResponse struct {
	Problems   []problemListItemResp `json:"problems"`
	Pagination paginationResp        `json:"pagination"`
}

func convertLangOverrides(overrides []langOverrideRequest) []appProblem.LanguageOverrideInput {
	result := make([]appProblem.LanguageOverrideInput, 0, len(overrides))
	for _, lo := range overrides {
		result = append(result, appProblem.LanguageOverrideInput{
			Language:    lo.Language,
			TimeLimit:   lo.TimeLimit,
			MemoryLimit: lo.MemoryLimit,
		})
	}
	return result
}

func buildResponse(p appProblem.ProblemDTO, display *appProblem.UserDisplay) getProblemResponse {
	author := authorResp{Nickname: "unknown", Name: ""}
	if display != nil {
		author = authorResp{Nickname: display.Nickname, Name: display.Name}
	}

	overrides := make([]langOverrideResp, 0, len(p.LangOverrides))
	for _, lo := range p.LangOverrides {
		overrides = append(overrides, langOverrideResp{
			Language:    lo.Language,
			TimeLimit:   lo.TimeLimit,
			MemoryLimit: lo.MemoryLimit,
		})
	}

	var judgingUpdatedAt *string
	if p.JudgingUpdatedAt != nil {
		s := p.JudgingUpdatedAt.UTC().Format(time.RFC3339)
		judgingUpdatedAt = &s
	}

	solutions := make([]solutionResp, 0, len(p.Solutions))
	for _, sol := range p.Solutions {
		solutions = append(solutions, solutionResp{
			Filename: sol.Filename,
			Language: sol.Language,
		})
	}

	return getProblemResponse{
		Slug:          p.Slug,
		Title:         p.Title,
		Statement:     p.Statement,
		TimeLimit:     p.TimeLimit,
		MemoryLimit:   p.MemoryLimit,
		LangOverrides: overrides,
		Tags:          p.Tags,
		Status:        p.Status,
		Accessibility: p.Accessibility,
		Author:        author,
		Files: &filesResp{
			TestCases: p.TestCasesLoaded,
			Solutions: solutions,
			Checker:   p.CheckerLoaded,
			Validator: p.ValidatorLoaded,
		},
		ProblemJudgingUpdatedAt: judgingUpdatedAt,
		CreatedAt:               p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:               p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}
