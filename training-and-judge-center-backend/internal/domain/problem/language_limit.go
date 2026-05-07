package problem

import "github.com/training-judge-center/backend/pkg/apperror"

type LanguageLimit struct {
	language       string
	maxTimeLimit   int
	maxMemoryLimit int
}

func NewLanguageLimit(language string, maxTime, maxMemory int) (LanguageLimit, error) {
	if language == "" || maxTime <= 0 || maxMemory <= 0 {
		return LanguageLimit{}, apperror.NewInternal()
	}
	return LanguageLimit{
		language:       language,
		maxTimeLimit:   maxTime,
		maxMemoryLimit: maxMemory,
	}, nil
}

func (l LanguageLimit) Language() string       { return l.language }
func (l LanguageLimit) MaxTimeLimit() int       { return l.maxTimeLimit }
func (l LanguageLimit) MaxMemoryLimit() int     { return l.maxMemoryLimit }
