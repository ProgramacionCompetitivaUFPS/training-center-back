package problem

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Problem struct {
	id               string
	slug             Slug
	title            Title
	statement        Statement
	timeLimit        *TimeLimit
	memoryLimit      *MemoryLimit
	langOverrides    []LanguageOverride
	tags             Tags
	status           Status
	accessibility    Accessibility
	authorID         shared.UserID
	modifierIDs      []shared.UserID
	testCasesKey     *string
	solutions        []JudgingFile
	checker          *JudgingFile
	validator        *JudgingFile
	judgingUpdatedAt *time.Time
	createdAt        time.Time
	updatedAt        time.Time
}

func (p *Problem) ID() string                        { return p.id }
func (p *Problem) Slug() Slug                        { return p.slug }
func (p *Problem) Title() Title                      { return p.title }
func (p *Problem) Statement() Statement              { return p.statement }
func (p *Problem) TimeLimit() *TimeLimit             { return p.timeLimit }
func (p *Problem) MemoryLimit() *MemoryLimit         { return p.memoryLimit }
func (p *Problem) LangOverrides() []LanguageOverride { return p.langOverrides }
func (p *Problem) Tags() Tags                        { return p.tags }
func (p *Problem) Status() Status                    { return p.status }
func (p *Problem) Accessibility() Accessibility      { return p.accessibility }
func (p *Problem) AuthorID() shared.UserID           { return p.authorID }
func (p *Problem) ModifierIDs() []shared.UserID      { return p.modifierIDs }
func (p *Problem) TestCasesKey() *string             { return p.testCasesKey }
func (p *Problem) Solutions() []JudgingFile          { return p.solutions }
func (p *Problem) Checker() *JudgingFile             { return p.checker }
func (p *Problem) Validator() *JudgingFile           { return p.validator }
func (p *Problem) JudgingUpdatedAt() *time.Time      { return p.judgingUpdatedAt }
func (p *Problem) CreatedAt() time.Time              { return p.createdAt }
func (p *Problem) UpdatedAt() time.Time              { return p.updatedAt }

func NewProblem(
	id string,
	slug Slug,
	title Title,
	statement Statement,
	timeLimit *TimeLimit,
	memoryLimit *MemoryLimit,
	langOverrides []LanguageOverride,
	tags Tags,
	authorID shared.UserID,
) *Problem {
	now := time.Now().UTC()
	return &Problem{
		id:            id,
		slug:          slug,
		title:         title,
		statement:     statement,
		timeLimit:     timeLimit,
		memoryLimit:   memoryLimit,
		langOverrides: langOverrides,
		tags:          tags,
		status:        NewStatusDraft(),
		accessibility: NewAccessibilityPrivate(),
		authorID:      authorID,
		modifierIDs:   []shared.UserID{},
		solutions:     []JudgingFile{},
		createdAt:     now,
		updatedAt:     now,
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
		p.title = *title
	}
	if statement != nil {
		p.statement = *statement
	}
	if timeLimit != nil {
		p.timeLimit = *timeLimit
	}
	if memoryLimit != nil {
		p.memoryLimit = *memoryLimit
	}
	if langOverrides != nil {
		p.langOverrides = langOverrides
	}
	if tags != nil {
		p.tags = *tags
	}
	p.updatedAt = time.Now().UTC()
}

func (p *Problem) CanBeEditedBy(userID shared.UserID, isAdmin bool) bool {
	if p.authorID == userID || isAdmin {
		return true
	}
	for _, id := range p.modifierIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (p *Problem) Unpublish() error {
	if !p.status.IsPublished() {
		return apperror.NewConflict(ErrCodeAlreadyDraft, "Problem is already unpublished")
	}
	p.status = NewStatusDraft()
	p.updatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) UpdateAccessibility(acc Accessibility) {
	p.accessibility = acc
	p.updatedAt = time.Now().UTC()
}

func (p *Problem) AddModifier(userID shared.UserID) error {
	for _, id := range p.modifierIDs {
		if id == userID {
			return apperror.NewConflict(ErrCodeModifierAlreadyExists, "User is already a modifier of this problem")
		}
	}
	p.modifierIDs = append(p.modifierIDs, userID)
	p.updatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) RemoveModifier(userID shared.UserID) error {
	found := false
	newModifiers := make([]shared.UserID, 0, len(p.modifierIDs))
	for _, id := range p.modifierIDs {
		if id == userID {
			found = true
			continue
		}
		newModifiers = append(newModifiers, id)
	}
	if !found {
		return apperror.NewNotFound(ErrCodeModifierNotFound, "User is not a modifier of this problem")
	}
	p.modifierIDs = newModifiers
	p.updatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) touchJudgingUpdatedAt(now time.Time) {
	p.judgingUpdatedAt = &now
	p.updatedAt = now
}

func (p *Problem) SetTestCases(fileKey string) {
	p.testCasesKey = &fileKey
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) RemoveTestCases() {
	p.testCasesKey = nil
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) AddSolution(solution JudgingFile) *JudgingFile {
	for i, sol := range p.solutions {
		if sol.Filename() == solution.Filename() {
			old := sol
			p.solutions[i] = solution
			p.updatedAt = time.Now().UTC()
			return &old
		}
	}
	p.solutions = append(p.solutions, solution)
	p.updatedAt = time.Now().UTC()
	return nil
}

func (p *Problem) RemoveSolution(filename string) {
	newSolutions := make([]JudgingFile, 0, len(p.solutions))
	for _, sol := range p.solutions {
		if sol.Filename() != filename {
			newSolutions = append(newSolutions, sol)
		}
	}
	p.solutions = newSolutions
	p.updatedAt = time.Now().UTC()
}

func (p *Problem) SetChecker(checker JudgingFile) {
	p.checker = &checker
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) RemoveChecker() {
	p.checker = nil
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) SetValidator(validator JudgingFile) {
	p.validator = &validator
	p.touchJudgingUpdatedAt(time.Now().UTC())
}

func (p *Problem) RemoveValidator() {
	p.validator = nil
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
	authorID shared.UserID,
	modifierIDs []shared.UserID,
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
		modifierIDs = []shared.UserID{}
	}
	if langOverrides == nil {
		langOverrides = []LanguageOverride{}
	}
	if solutions == nil {
		solutions = []JudgingFile{}
	}

	return &Problem{
		id:               id,
		slug:             RestoreSlug(slug),
		title:            RestoreTitle(title),
		statement:        RestoreStatement(statement),
		timeLimit:        tl,
		memoryLimit:      ml,
		langOverrides:    langOverrides,
		tags:             RestoreTags(tags),
		status:           RestoreStatus(status),
		accessibility:    RestoreAccessibility(accessibility),
		authorID:         authorID,
		modifierIDs:      modifierIDs,
		testCasesKey:     testCasesKey,
		solutions:        solutions,
		checker:          checker,
		validator:        validator,
		judgingUpdatedAt: judgingUpdatedAt,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
	}
}
