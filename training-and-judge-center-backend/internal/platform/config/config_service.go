package config

import (
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/domain/problem"
)

func NewPlatformSettings(cfg *config.VirtualObject) *problem.PlatformSettings {
	if cfg == nil {
		return &problem.PlatformSettings{
			LanguageLimits:     make(map[string]*problem.LanguageLimit),
			SupportedLanguages: make(map[string]struct{}),
			LanguageExtensions: make(map[string]string),
			AllowedTags:        make(map[string]struct{}),
		}
	}

	langs := make(map[string]*problem.LanguageLimit, len(cfg.LanguageOverrides))
	for _, l := range cfg.LanguageOverrides {
		l := l
		langs[l.Language] = &problem.LanguageLimit{
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

	return &problem.PlatformSettings{
		GlobalMaxTimeLimit:    cfg.MaxTimeLimitGlobal,
		GlobalMaxMemoryLimit:  cfg.MaxMemoryLimitGlobal,
		LanguageLimits:        langs,
		SupportedLanguages:    supportedMap,
		LanguageExtensions:    cfg.LanguageExtensions,
		AllowedTags:           tagsMap,
		UploadMaxConcurrency:  defaultIfZero(cfg.UploadMaxConcurrency, 10),
		MaxFileCountSample:    defaultIfZero(cfg.MaxFileCountSample, 10),
		MaxFileSizeTestCaseMB: defaultIfZero(cfg.MaxFileSizeTestCaseMB, 200),
		MaxFileCountTestCase:  defaultIfZero(cfg.MaxFileCountTestCase, 10000),
		MaxFileSizeDefaultMB:  defaultIfZero(cfg.MaxFileSizeDefaultMB, 10),
	}
}

func defaultIfZero(v, d int) int {
	if v <= 0 {
		return d
	}
	return v
}
