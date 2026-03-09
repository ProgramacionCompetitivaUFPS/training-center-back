package config

import (
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/internal/domain/problem"
)

type VirtualObjectProvider struct {
	globalMaxTimeLimit   int
	globalMaxMemoryLimit int
	languages            map[string]problem.LanguageLimit
	allowedTags          map[string]struct{}
	tagsList             []string
}

func NewVirtualObjectProvider(cfg *config.VirtualObject) *VirtualObjectProvider {
	if cfg == nil {
		return &VirtualObjectProvider{
			languages:   make(map[string]problem.LanguageLimit),
			allowedTags: make(map[string]struct{}),
			tagsList:    []string{},
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

	return &VirtualObjectProvider{
		globalMaxTimeLimit:   cfg.MaxTimeLimitGlobal,
		globalMaxMemoryLimit: cfg.MaxMemoryLimitGlobal,
		languages:            langs,
		allowedTags:          tagsMap,
		tagsList:             cfg.Tags,
	}
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

func (p *VirtualObjectProvider) GetValidTags() []string {
	return p.tagsList
}
