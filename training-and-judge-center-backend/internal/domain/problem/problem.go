package problem

import (
	"time"
)

type Problem struct {
	ID            string
	Slug          Slug
	Title         Title
	Statement     *string
	TimeLimit     *int
	MemoryLimit   *int
	LangOverrides []LanguageOverride
	Tags          Tags
	Status        Status
	Accessibility Accessibility
	AuthorID      string
	ModifierIDs   []string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func NewProblem(
	id string,
	slug Slug,
	title Title,
	statement *string,
	timeLimit *int,
	memoryLimit *int,
	langOverrides []LanguageOverride,
	tags Tags,
	authorID string,
) *Problem {
	now := time.Now().UTC()
	return &Problem{
		ID:            id,
		Slug:          slug,
		Title:         title,
		Statement:     statement,
		TimeLimit:     timeLimit,
		MemoryLimit:   memoryLimit,
		LangOverrides: langOverrides,
		Tags:          tags,
		Status:        NewStatusDraft(),
		Accessibility: NewAccessibilityPrivate(),
		AuthorID:      authorID,
		ModifierIDs:   []string{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}
