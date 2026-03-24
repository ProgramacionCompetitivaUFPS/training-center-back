package problem

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Problem struct {
	ID               string
	Slug             Slug
	Title            Title
	Statement        *string
	TimeLimit        *TimeLimit
	MemoryLimit      *MemoryLimit
	LangOverrides    []LanguageOverride
	Tags             Tags
	Status           Status
	Accessibility    Accessibility
	AuthorID         string
	ModifierIDs      []string
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
	statement *string,
	timeLimit *TimeLimit,
	memoryLimit *MemoryLimit,
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
		Solutions:     []JudgingFile{},
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func (p *Problem) UpdateMetadata(
	title *Title,
	statement **string,
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

func (p *Problem) CanBeEditedBy(userID string, role user.Role) bool {
	if p.AuthorID == userID || role == user.RoleAdmin {
		return true
	}
	for _, id := range p.ModifierIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (p *Problem) UpdateAccessibility(acc Accessibility) {
	p.Accessibility = acc
	p.UpdatedAt = time.Now().UTC()
}

func (p *Problem) AddModifier(userID string) error {
	for _, id := range p.ModifierIDs {
		if id == userID {
			return apperror.NewConflict(ErrCodeModifierAlreadyExists, "User is already a modifier of this problem")
		}
	}
	p.ModifierIDs = append(p.ModifierIDs, userID)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) RemoveModifier(userID string) error {
	found := false
	var newModifiers []string
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

// AddSolution adds a new solution or replaces one with the same filename.
// Returns the replaced JudgingFile if one existed, nil otherwise.
// The caller should use this to clean up the old storage key when keys differ.
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
	authorID string,
	modifierIDs []string,
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
		tl = &TimeLimit{value: *timeLimit}
	}

	var ml *MemoryLimit
	if memoryLimit != nil {
		ml = &MemoryLimit{value: *memoryLimit}
	}

	var tagsVO Tags
	if tags != nil {
		tagsVO = Tags{values: tags}
	} else {
		tagsVO = Tags{values: []string{}}
	}

	if modifierIDs == nil {
		modifierIDs = []string{}
	}

	if langOverrides == nil {
		langOverrides = []LanguageOverride{}
	}

	if solutions == nil {
		solutions = []JudgingFile{}
	}

	return &Problem{
		ID:               id,
		Slug:             Slug{value: slug},
		Title:            Title{value: title},
		Statement:        statement,
		TimeLimit:        tl,
		MemoryLimit:      ml,
		LangOverrides:    langOverrides,
		Tags:             tagsVO,
		Status:           Status{value: status},
		Accessibility:    Accessibility{value: accessibility},
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
