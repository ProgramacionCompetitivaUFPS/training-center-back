package problem

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type Problem struct {
	ID               string
	Slug             Slug
	Title            Title
	Statement        Statement
	TimeLimit        *TimeLimit
	MemoryLimit      *MemoryLimit
	LangOverrides    []LanguageOverride
	Tags             Tags
	Status           Status
	Accessibility    Accessibility
	AuthorID         UserID
	ModifierIDs      []UserID
	TestCasesKey     *string
	Solutions        []JudgingFile
	Checker          *JudgingFile
	Validator        *JudgingFile
	JudgingUpdatedAt *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func NewProblem(
	id string,
	slug Slug,
	title Title,
	statement Statement,
	timeLimit *TimeLimit,
	memoryLimit *MemoryLimit,
	langOverrides []LanguageOverride,
	tags Tags,
	authorID UserID,
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
		ModifierIDs:   []UserID{},
		Solutions:     []JudgingFile{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (p *Problem) UpdateMetadata(
	title *Title,
	statement *Statement,
	timeLimit **TimeLimit,
	memoryLimit **MemoryLimit,
	langOverrides []LanguageOverride,
	tags *Tags,
) {
	if title != nil {
		p.Title = *title
	}
	if statement != nil {
		p.Statement = *statement
	}
	if timeLimit != nil {
		p.TimeLimit = *timeLimit
	}
	if memoryLimit != nil {
		p.MemoryLimit = *memoryLimit
	}
	if langOverrides != nil {
		p.LangOverrides = langOverrides
	}
	if tags != nil {
		p.Tags = *tags
	}
	p.UpdatedAt = time.Now().UTC()
}

func (p *Problem) CanBeEditedBy(userID UserID, isAdmin bool) bool {
	if p.AuthorID == userID || isAdmin {
		return true
	}
	for _, id := range p.ModifierIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (p *Problem) Unpublish() error {
	if !p.Status.IsPublished() {
		return apperror.NewConflict(ErrCodeAlreadyDraft, "Problem is already unpublished")
	}
	p.Status = NewStatusDraft()
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) UpdateAccessibility(acc Accessibility) {
	p.Accessibility = acc
	p.UpdatedAt = time.Now().UTC()
}

func (p *Problem) AddModifier(userID UserID) error {
	for _, id := range p.ModifierIDs {
		if id == userID {
			return apperror.NewConflict(ErrCodeModifierAlreadyExists, "User is already a modifier of this problem")
		}
	}
	p.ModifierIDs = append(p.ModifierIDs, userID)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) RemoveModifier(userID UserID) error {
	found := false
	var newModifiers []UserID
	for _, id := range p.ModifierIDs {
		if id == userID {
			found = true
			continue
		}
		newModifiers = append(newModifiers, id)
	}
	if !found {
		return apperror.NewNotFound(ErrCodeModifierNotFound, "User is not a modifier of this problem")
	}
	p.ModifierIDs = newModifiers
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) touchJudgingUpdatedAt(now time.Time) {
	p.JudgingUpdatedAt = &now
	p.UpdatedAt = now
}

func (p *Problem) SetTestCases(fileKey string) {
	p.TestCasesKey = &fileKey
	p.touchJudgingUpdatedAt(time.Now().UTC())
}
func (p *Problem) RemoveTestCases() {
	p.TestCasesKey = nil
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) AddSolution(solution JudgingFile) *JudgingFile {
	for i, sol := range p.Solutions {
		if sol.Filename() == solution.Filename() {
			old := sol
			p.Solutions[i] = solution
			p.UpdatedAt = time.Now().UTC()
			return &old
		}
	}
	p.Solutions = append(p.Solutions, solution)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) RemoveSolution(filename string) {
	var newSolutions []JudgingFile
	for _, sol := range p.Solutions {
		if sol.Filename() != filename {
			newSolutions = append(newSolutions, sol)
		}
	}
	p.Solutions = newSolutions
	p.UpdatedAt = time.Now().UTC()
}

func (p *Problem) SetChecker(checker JudgingFile) {
	p.Checker = &checker
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) RemoveChecker() {
	p.Checker = nil
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) SetValidator(validator JudgingFile) {
	p.Validator = &validator
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) RemoveValidator() {
	p.Validator = nil
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func RestoreProblem(
	id string,
	slug string,
	title string,
	statement *string,
	timeLimit *int,
	memoryLimit *int,
	tags []string,
	status string,
	accessibility string,
	authorID UserID,
	modifierIDs []UserID,
	langOverrides []LanguageOverride,
	testCasesKey *string,
	solutions []JudgingFile,
	checker *JudgingFile,
	validator *JudgingFile,
	judgingUpdatedAt *time.Time,
	createdAt time.Time,
	updatedAt time.Time,
) *Problem {
	var tl *TimeLimit
	if timeLimit != nil {
		restored := RestoreTimeLimit(*timeLimit)
		tl = &restored
	}

	var ml *MemoryLimit
	if memoryLimit != nil {
		restored := RestoreMemoryLimit(*memoryLimit)
		ml = &restored
	}

	if modifierIDs == nil {
		modifierIDs = []UserID{}
	}

	if langOverrides == nil {
		langOverrides = []LanguageOverride{}
	}

	if solutions == nil {
		solutions = []JudgingFile{}
	}

	return &Problem{
		ID:               id,
		Slug:             RestoreSlug(slug),
		Title:            RestoreTitle(title),
		Statement:        RestoreStatement(statement),
		TimeLimit:        tl,
		MemoryLimit:      ml,
		LangOverrides:    langOverrides,
		Tags:             RestoreTags(tags),
		Status:           RestoreStatus(status),
		Accessibility:    RestoreAccessibility(accessibility),
		AuthorID:         authorID,
		ModifierIDs:      modifierIDs,
		TestCasesKey:     testCasesKey,
		Solutions:        solutions,
		Checker:          checker,
		Validator:        validator,
		JudgingUpdatedAt: judgingUpdatedAt,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
}
