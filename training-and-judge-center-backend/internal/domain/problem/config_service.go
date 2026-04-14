package problem

type LanguageLimit struct {
	Language       string
	MaxTimeLimit   int
	MaxMemoryLimit int
}

type PlatformSettings struct {
	GlobalMaxTimeLimit    int
	GlobalMaxMemoryLimit  int
	LanguageLimits        map[string]*LanguageLimit
	SupportedLanguages    map[string]struct{}
	LanguageExtensions    map[string]string
	AllowedTags           map[string]struct{}
	UploadMaxConcurrency  int
	MaxFileCountSample    int
	MaxFileSizeTestCaseMB int
	MaxFileCountTestCase  int
	MaxFileSizeDefaultMB  int
}

func (s *PlatformSettings) IsLanguageSupported(language string) bool {
	_, ok := s.SupportedLanguages[language]
	return ok
}

func (s *PlatformSettings) GetLanguageByExtension(ext string) (string, bool) {
	lang, ok := s.LanguageExtensions[ext]
	return lang, ok
}

func (s *PlatformSettings) GetUploadMaxConcurrency() int  { return s.UploadMaxConcurrency }
func (s *PlatformSettings) GetMaxFileCountSample() int    { return s.MaxFileCountSample }
func (s *PlatformSettings) GetMaxFileSizeDefaultMB() int  { return s.MaxFileSizeDefaultMB }
func (s *PlatformSettings) GetMaxFileSizeTestCaseMB() int { return s.MaxFileSizeTestCaseMB }
func (s *PlatformSettings) GetMaxFileCountTestCase() int  { return s.MaxFileCountTestCase }

func (s *PlatformSettings) GetGlobalLimits() (int, int) {
	return s.GlobalMaxTimeLimit, s.GlobalMaxMemoryLimit
}

func (s *PlatformSettings) GetLanguageLimit(language string) *LanguageLimit {
	return s.LanguageLimits[language]
}

func (s *PlatformSettings) IsValidTag(tag string) bool {
	_, ok := s.AllowedTags[tag]
	return ok
}

func (s *PlatformSettings) GetAllowedTags() map[string]struct{} {
	return s.AllowedTags
}
