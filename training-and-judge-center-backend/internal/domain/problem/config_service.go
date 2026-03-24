package problem

type LanguageLimit struct {
	Language       string
	MaxTimeLimit   int
	MaxMemoryLimit int
}

type PlatformSettingsService interface {
	GetLanguageByExtension(ext string) (string, bool)
	GetUploadMaxConcurrency() int
	GetMaxFileCountSample() int
	GetMaxFileSizeDefaultMB() int
	GetMaxFileSizeTestCaseMB() int
	GetMaxFileCountTestCase() int
	GetGlobalLimits() (maxTimeLimit, maxMemoryLimit int)
	GetLanguageLimit(language string) *LanguageLimit
	IsValidTag(tag string) bool
	GetAllowedTags() map[string]struct{}
}
