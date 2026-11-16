package problem

const (
	ErrCodeSlugAlreadyExists      = "SLUG_ALREADY_EXISTS"
	ErrCodeModifierAlreadyExists  = "MODIFIER_ALREADY_EXISTS"
	ErrCodeModifierNotFound       = "MODIFIER_NOT_FOUND"
	ErrCodeSolutionNotFound       = "SOLUTION_NOT_FOUND"
	ErrCodeTooManyModifiers       = "TOO_MANY_MODIFIERS"
	ErrCodeAlreadyDraft           = "ALREADY_DRAFT"
	ErrCodeAlreadyPublished       = "ALREADY_PUBLISHED"
	ErrCodeSlugMismatch           = "SLUG_MISMATCH"
	ErrCodeProblemNotFound        = "PROBLEM_NOT_FOUND"
	ErrCodeProblemInActiveContest = "PROBLEM_IN_ACTIVE_CONTEST"
	ErrCodeValidationInProgress   = "VALIDATION_IN_PROGRESS"
	ErrCodeExtractedSizeTooLarge  = "EXTRACTED_SIZE_TOO_LARGE"
	ErrCodeMissingProblemYAML     = "MISSING_PROBLEM_YAML"
	ErrCodeInvalidProblemYAML     = "INVALID_PROBLEM_YAML"
	ErrCodeFileTooLarge           = "FILE_TOO_LARGE"
)
