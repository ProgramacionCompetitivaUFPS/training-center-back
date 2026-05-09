package problem

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/problem"
)

type LangOverrideDTO struct {
	Language    string
	TimeLimit   *int
	MemoryLimit *int
}

type SolutionDTO struct {
	Filename string
	Language string
}

type ProblemDTO struct {
	Slug             string
	Title            string
	Statement        *string
	TimeLimit        *int
	MemoryLimit      *int
	LangOverrides    []LangOverrideDTO
	Tags             []string
	Status           string
	Accessibility    string
	AuthorID         string
	Solutions        []SolutionDTO
	TestCasesLoaded  bool
	CheckerLoaded    bool
	ValidatorLoaded  bool
	JudgingUpdatedAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func problemToDTO(p *problem.Problem) ProblemDTO {
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

	overrides := make([]LangOverrideDTO, 0, len(p.LangOverrides()))
	for _, lo := range p.LangOverrides() {
		overrides = append(overrides, LangOverrideDTO{
			Language:    lo.Language(),
			TimeLimit:   lo.TimeLimit(),
			MemoryLimit: lo.MemoryLimit(),
		})
	}

	solutions := make([]SolutionDTO, 0, len(p.Solutions()))
	for _, sol := range p.Solutions() {
		solutions = append(solutions, SolutionDTO{
			Filename: sol.Filename(),
			Language: sol.Language(),
		})
	}

	return ProblemDTO{
		Slug:             p.Slug().String(),
		Title:            p.Title().String(),
		Statement:        p.Statement().Value(),
		TimeLimit:        tl,
		MemoryLimit:      ml,
		LangOverrides:    overrides,
		Tags:             p.Tags().Values(),
		Status:           p.Status().String(),
		Accessibility:    p.Accessibility().String(),
		AuthorID:         p.AuthorID().Value(),
		Solutions:        solutions,
		TestCasesLoaded:  p.TestCasesKey() != nil,
		CheckerLoaded:    p.Checker() != nil,
		ValidatorLoaded:  p.Validator() != nil,
		JudgingUpdatedAt: p.JudgingUpdatedAt(),
		CreatedAt:        p.CreatedAt(),
		UpdatedAt:        p.UpdatedAt(),
	}
}
