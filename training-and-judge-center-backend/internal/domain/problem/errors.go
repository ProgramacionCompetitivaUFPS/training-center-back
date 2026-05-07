package problem

const (
	ErrCodeSlugAlreadyExists     = "SLUG_ALREADY_EXISTS"
	ErrCodeModifierAlreadyExists = "MODIFIER_ALREADY_EXISTS"
	ErrCodeModifierNotFound      = "MODIFIER_NOT_FOUND"
	ErrCodeSolutionNotFound      = "SOLUTION_NOT_FOUND"
	ErrCodeTooManyModifiers      = "TOO_MANY_MODIFIERS"
	ErrCodeAlreadyDraft          = "ALREADY_DRAFT"
	ErrCodeAlreadyPublished      = "ALREADY_PUBLISHED"
	ErrCodeSlugMismatch          = "SLUG_MISMATCH"
)

const (
	ErrCodeInvalidLanguage                = "INVALID_LANGUAGE"
	ErrCodeInvalidTimeLimit               = "INVALID_TIME_LIMIT"
	ErrCodeInvalidMemoryLimit             = "INVALID_MEMORY_LIMIT"
	ErrCodeInvalidUploadConcurrency       = "INVALID_UPLOAD_CONCURRENCY"
	ErrCodeInvalidFileCount               = "INVALID_FILE_COUNT"
	ErrCodeInvalidFileSize                = "INVALID_FILE_SIZE"
	ErrCodeNoSupportedLanguages           = "NO_SUPPORTED_LANGUAGES"
	ErrCodeLanguageLimitExceedsGlobal     = "LANGUAGE_LIMIT_EXCEEDS_GLOBAL"
	ErrCodeFileSizeDefaultExceedsTestCase = "FILE_SIZE_DEFAULT_EXCEEDS_TEST_CASE"
)
