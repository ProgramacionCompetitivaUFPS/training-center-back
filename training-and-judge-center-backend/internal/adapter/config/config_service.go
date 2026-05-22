package config

import (
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func NewPlatformSettings(cfg *config.VirtualObject) (problem.PlatformSettings, error) {
	if cfg == nil {
		return problem.PlatformSettings{}, apperror.NewInternal()
	}

	langs := make(map[string]problem.LanguageLimit, len(cfg.LanguageOverrides))
	for _, l := range cfg.LanguageOverrides {
		ll, err := problem.NewLanguageLimit(l.Language, l.MaxTimeLimit, l.MaxMemoryLimit)
		if err != nil {
			return problem.PlatformSettings{}, err
		}
		langs[l.Language] = ll
	}

	tagsMap := make(map[string]struct{}, len(cfg.Tags))
	for _, t := range cfg.Tags {
		tagsMap[t] = struct{}{}
	}

	supportedMap := make(map[string]struct{}, len(cfg.SupportedLanguages))
	for _, l := range cfg.SupportedLanguages {
		supportedMap[l] = struct{}{}
	}

	return problem.NewPlatformSettings(
		cfg.MaxTimeLimitGlobal,
		cfg.MaxMemoryLimitGlobal,
		langs,
		supportedMap,
		cfg.LanguageExtensions,
		tagsMap,
		defaultIfZero(cfg.UploadMaxConcurrency, 10),
		defaultIfZero(cfg.MaxFileCountSample, 10),
		defaultIfZero(cfg.MaxFileSizeTestCaseMB, 200),
		defaultIfZero(cfg.MaxFileCountTestCase, 10000),
		defaultIfZero(cfg.MaxFileSizeDefaultMB, 10),
	)
}

func defaultIfZero(v, d int) int {
	if v == 0 {
		return d
	}
	return v
}
