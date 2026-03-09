package problem

type LanguageLimit struct {
	Language       string
	MaxTimeLimit   int
	MaxMemoryLimit int
}

type PlatformSettingsService interface {
	GetGlobalLimits() (maxTimeLimit, maxMemoryLimit int)
	GetLanguageLimit(language string) *LanguageLimit
	IsValidTag(tag string) bool
	GetValidTags() []string
}
