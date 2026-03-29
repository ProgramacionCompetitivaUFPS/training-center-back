package config

import (
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/domain/problem"
)

type VirtualObjectProvider struct {
	languageExtensions    map[string]string
	uploadMaxConcurrency  int
	maxFileCountSample    int
	maxFileSizeTestCaseMB int
	maxFileCountTestCase  int
	maxFileSizeDefaultMB  int
	globalMaxTimeLimit   int
	globalMaxMemoryLimit int
	languages            map[string]problem.LanguageLimit
	supportedLanguages   map[string]struct{}
	allowedTags          map[string]struct{}
	tagsList             []string
}

func NewVirtualObjectProvider(cfg *config.VirtualObject) *VirtualObjectProvider {
	if cfg == nil {
		return &VirtualObjectProvider{
			languages:          make(map[string]problem.LanguageLimit),
			supportedLanguages: make(map[string]struct{}),
			allowedTags:        make(map[string]struct{}),
			tagsList:           []string{},
		}
	}

	langs := make(map[string]problem.LanguageLimit, len(cfg.LanguageOverrides))
	for _, l := range cfg.LanguageOverrides {
		langs[l.Language] = problem.LanguageLimit{
			Language:       l.Language,
			MaxTimeLimit:   l.MaxTimeLimit,
			MaxMemoryLimit: l.MaxMemoryLimit,
		}
	}

	tagsMap := make(map[string]struct{}, len(cfg.Tags))
	for _, t := range cfg.Tags {
		tagsMap[t] = struct{}{}
	}

	supportedMap := make(map[string]struct{}, len(cfg.SupportedLanguages))
	for _, l := range cfg.SupportedLanguages {
		supportedMap[l] = struct{}{}
	}

	return &VirtualObjectProvider{
		languageExtensions:    cfg.LanguageExtensions,
		uploadMaxConcurrency:  cfg.UploadMaxConcurrency,
		maxFileCountSample:    cfg.MaxFileCountSample,
		maxFileSizeTestCaseMB: cfg.MaxFileSizeTestCaseMB,
		maxFileCountTestCase:  cfg.MaxFileCountTestCase,
		maxFileSizeDefaultMB:  cfg.MaxFileSizeDefaultMB,
		globalMaxTimeLimit:   cfg.MaxTimeLimitGlobal,
		globalMaxMemoryLimit: cfg.MaxMemoryLimitGlobal,
		languages:            langs,
		supportedLanguages:   supportedMap,
		allowedTags:          tagsMap,
		tagsList:             cfg.Tags,
	}
}

func (p *VirtualObjectProvider) IsLanguageSupported(language string) bool {
	_, ok := p.supportedLanguages[language]
	return ok
}

func (p *VirtualObjectProvider) GetLanguageByExtension(ext string) (string, bool) {
	lang, ok := p.languageExtensions[ext]
	return lang, ok
}

func (p *VirtualObjectProvider) GetUploadMaxConcurrency() int {
	if p.uploadMaxConcurrency <= 0 {
		return 10 // Default fallback
	}
	return p.uploadMaxConcurrency
}

func (p *VirtualObjectProvider) GetMaxFileCountSample() int {
	if p.maxFileCountSample <= 0 {
		return 10 // Default fallback
	}
	return p.maxFileCountSample
}

func (p *VirtualObjectProvider) GetMaxFileSizeDefaultMB() int {
	if p.maxFileSizeDefaultMB <= 0 {
		return 10 // Default fallback
	}
	return p.maxFileSizeDefaultMB
}

func (p *VirtualObjectProvider) GetMaxFileSizeTestCaseMB() int {
	if p.maxFileSizeTestCaseMB <= 0 {
		return 200 // Default fallback
	}
	return p.maxFileSizeTestCaseMB
}

func (p *VirtualObjectProvider) GetMaxFileCountTestCase() int {
	if p.maxFileCountTestCase <= 0 {
		return 10000 // Default fallback
	}
	return p.maxFileCountTestCase
}

func (p *VirtualObjectProvider) GetGlobalLimits() (int, int) {
	return p.globalMaxTimeLimit, p.globalMaxMemoryLimit
}

func (p *VirtualObjectProvider) GetLanguageLimit(language string) *problem.LanguageLimit {
	if limit, ok := p.languages[language]; ok {
		return &limit
	}
	return nil
}

func (p *VirtualObjectProvider) IsValidTag(tag string) bool {
	_, ok := p.allowedTags[tag]
	return ok
}

func (p *VirtualObjectProvider) GetAllowedTags() map[string]struct{} {
	return p.allowedTags
}
