package problem

import (
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type createProblemRequest struct {
	Slug          string                `json:"slug"`
	Title         string                `json:"title"`
	Statement     *string               `json:"statement"`
	TimeLimit     *int                  `json:"timeLimit"`
	MemoryLimit   *int                  `json:"memoryLimit"`
	LangOverrides []langOverrideRequest `json:"languageOverrides"`
	Tags          []string              `json:"tags"`
}

type langOverrideRequest struct {
	Language    string `json:"language"`
	TimeLimit   *int   `json:"timeLimit"`
	MemoryLimit *int   `json:"memoryLimit"`
}

type updateProblemRequest struct {
	Title         *string               `json:"title"`
	Statement     *string               `json:"statement"`
	TimeLimit     *int                  `json:"timeLimit"`
	MemoryLimit   *int                  `json:"memoryLimit"`
	LangOverrides []langOverrideRequest `json:"languageOverrides"`
	Tags          []string              `json:"tags"`
	Accessibility *string               `json:"accessibility"`
}

type problemResponse struct {
	Slug          string             `json:"slug"`
	Title         string             `json:"title"`
	Statement     *string            `json:"statement"`
	TimeLimit     *int               `json:"timeLimit"`
	MemoryLimit   *int               `json:"memoryLimit"`
	LangOverrides []langOverrideResp `json:"languageOverrides"`
	Tags          []string           `json:"tags"`
	Status        string             `json:"status"`
	Accessibility string             `json:"accessibility"`
	Author        authorResp         `json:"author"`
	Modifiers     []interface{}      `json:"modifiers"`
	Files         filesResp          `json:"files"`
	CreatedAt     string             `json:"createdAt"`
	UpdatedAt     string             `json:"updatedAt"`
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
	Validator bool          `json:"validator"`
}

func buildResponse(p *domainProblem.Problem, display *user.Display) problemResponse {
	author := authorResp{Nickname: "unknown", Name: ""}
	if display != nil {
		author = authorResp{Nickname: display.Nickname, Name: display.Name}
	}

	overrides := make([]langOverrideResp, 0, len(p.LangOverrides))
	for _, lo := range p.LangOverrides {
		overrides = append(overrides, langOverrideResp{
			Language:    lo.Language(),
			TimeLimit:   lo.TimeLimit(),
			MemoryLimit: lo.MemoryLimit(),
		})
	}

	var tl *int
	if p.TimeLimit != nil {
		v := p.TimeLimit.Value()
		tl = &v
	}

	var ml *int
	if p.MemoryLimit != nil {
		v := p.MemoryLimit.Value()
		ml = &v
	}

	tags := p.Tags.Values()

	solutions := make([]solutionResp, 0, len(p.Solutions))
	for _, sol := range p.Solutions {
		solutions = append(solutions, solutionResp{
			Filename: sol.Filename(),
			Language: sol.Language(),
		})
	}

	return problemResponse{
		Slug:          p.Slug.String(),
		Title:         p.Title.String(),
		Statement:     p.Statement,
		TimeLimit:     tl,
		MemoryLimit:   ml,
		LangOverrides: overrides,
		Tags:          tags,
		Status:        p.Status.String(),
		Accessibility: p.Accessibility.String(),
		Author:        author,
		Modifiers:     []interface{}{},
		Files: filesResp{
			TestCases: p.TestCasesKey != nil,
			Solutions: solutions,
			Checker:   p.Checker != nil,
			Validator: p.Validator != nil,
		},
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
