package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type PlatformSettings struct {
	globalMaxTimeLimit    int
	globalMaxMemoryLimit  int
	languageLimits        map[string]LanguageLimit
	supportedLanguages    map[string]struct{}
	languageExtensions    map[string]string
	allowedTags           map[string]struct{}
	uploadMaxConcurrency  int
	maxFileCountSample    int
	maxFileSizeTestCaseMB int
	maxFileCountTestCase  int
	maxFileSizeDefaultMB  int
}

func NewPlatformSettings(
	globalMaxTimeLimit int,
	globalMaxMemoryLimit int,
	languageLimits map[string]LanguageLimit,
	supportedLanguages map[string]struct{},
	languageExtensions map[string]string,
	allowedTags map[string]struct{},
	uploadMaxConcurrency int,
	maxFileCountSample int,
	maxFileSizeTestCaseMB int,
	maxFileCountTestCase int,
	maxFileSizeDefaultMB int,
) (PlatformSettings, error) {
	var fieldErrs []apperror.FieldError

	if globalMaxTimeLimit <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "globalMaxTimeLimit", Message: "Global max time limit must be positive"})
	}
	if globalMaxMemoryLimit <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "globalMaxMemoryLimit", Message: "Global max memory limit must be positive"})
	}
	if uploadMaxConcurrency <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "uploadMaxConcurrency", Message: "Upload max concurrency must be positive"})
	}
	if maxFileCountSample <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxFileCountSample", Message: "Max file count for samples must be positive"})
	}
	if maxFileSizeTestCaseMB <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxFileSizeTestCaseMB", Message: "Max file size for test cases must be positive"})
	}
	if maxFileCountTestCase <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxFileCountTestCase", Message: "Max file count for test cases must be positive"})
	}
	if maxFileSizeDefaultMB <= 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxFileSizeDefaultMB", Message: "Default max file size must be positive"})
	}
	if len(supportedLanguages) == 0 {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "supportedLanguages", Message: "At least one supported language is required"})
	}

	// cross-field: default file size must not exceed test case file size
	if maxFileSizeDefaultMB > 0 && maxFileSizeTestCaseMB > 0 && maxFileSizeDefaultMB > maxFileSizeTestCaseMB {
		fieldErrs = append(fieldErrs, apperror.FieldError{Field: "maxFileSizeDefaultMB", Message: "Default file size limit cannot exceed test case file size limit"})
	}

	// cross-field: per-language limits must not exceed globals
	if globalMaxTimeLimit > 0 || globalMaxMemoryLimit > 0 {
		for lang, ll := range languageLimits {
			if globalMaxTimeLimit > 0 && ll.MaxTimeLimit() > globalMaxTimeLimit {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageLimits." + lang + ".maxTimeLimit",
					Message: "Language time limit exceeds global maximum",
				})
			}
			if globalMaxMemoryLimit > 0 && ll.MaxMemoryLimit() > globalMaxMemoryLimit {
				fieldErrs = append(fieldErrs, apperror.FieldError{
					Field:   "languageLimits." + lang + ".maxMemoryLimit",
					Message: "Language memory limit exceeds global maximum",
				})
			}
		}
	}

	if len(fieldErrs) > 0 {
		return PlatformSettings{}, apperror.NewValidation(fieldErrs)
	}
	return PlatformSettings{
		globalMaxTimeLimit:    globalMaxTimeLimit,
		globalMaxMemoryLimit:  globalMaxMemoryLimit,
		languageLimits:        languageLimits,
		supportedLanguages:    supportedLanguages,
		languageExtensions:    languageExtensions,
		allowedTags:           allowedTags,
		uploadMaxConcurrency:  uploadMaxConcurrency,
		maxFileCountSample:    maxFileCountSample,
		maxFileSizeTestCaseMB: maxFileSizeTestCaseMB,
		maxFileCountTestCase:  maxFileCountTestCase,
		maxFileSizeDefaultMB:  maxFileSizeDefaultMB,
	}, nil
}

func (s PlatformSettings) GlobalLimits() (int, int) {
	return s.globalMaxTimeLimit, s.globalMaxMemoryLimit
}

func (s PlatformSettings) IsLanguageSupported(language string) bool {
	_, ok := s.supportedLanguages[language]
	return ok
}

func (s PlatformSettings) LanguageByExtension(ext string) (string, bool) {
	lang, ok := s.languageExtensions[ext]
	return lang, ok
}

func (s PlatformSettings) LanguageLimit(language string) (LanguageLimit, bool) {
	l, ok := s.languageLimits[language]
	return l, ok
}

func (s PlatformSettings) AllowedTags() map[string]struct{} {
	cp := make(map[string]struct{}, len(s.allowedTags))
	for k, v := range s.allowedTags {
		cp[k] = v
	}
	return cp
}

func (s PlatformSettings) UploadMaxConcurrency() int  { return s.uploadMaxConcurrency }
func (s PlatformSettings) MaxFileCountSample() int    { return s.maxFileCountSample }
func (s PlatformSettings) MaxFileSizeDefaultMB() int  { return s.maxFileSizeDefaultMB }
func (s PlatformSettings) MaxFileSizeTestCaseMB() int { return s.maxFileSizeTestCaseMB }
func (s PlatformSettings) MaxFileCountTestCase() int  { return s.maxFileCountTestCase }

func (s PlatformSettings) IsValidTag(tag string) bool {
	_, ok := s.allowedTags[tag]
	return ok
}
