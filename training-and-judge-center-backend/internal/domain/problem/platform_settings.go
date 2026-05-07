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
	if globalMaxTimeLimit <= 0 || globalMaxMemoryLimit <= 0 ||
		uploadMaxConcurrency <= 0 || maxFileCountSample <= 0 ||
		maxFileSizeTestCaseMB <= 0 || maxFileCountTestCase <= 0 ||
		maxFileSizeDefaultMB <= 0 {
		return PlatformSettings{}, apperror.NewInternal()
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

func (s PlatformSettings) AllowedTags() map[string]struct{} { return s.allowedTags }

func (s PlatformSettings) UploadMaxConcurrency() int  { return s.uploadMaxConcurrency }
func (s PlatformSettings) MaxFileCountSample() int    { return s.maxFileCountSample }
func (s PlatformSettings) MaxFileSizeDefaultMB() int  { return s.maxFileSizeDefaultMB }
func (s PlatformSettings) MaxFileSizeTestCaseMB() int { return s.maxFileSizeTestCaseMB }
func (s PlatformSettings) MaxFileCountTestCase() int  { return s.maxFileCountTestCase }

func (s PlatformSettings) IsValidTag(tag string) bool {
	_, ok := s.allowedTags[tag]
	return ok
}
