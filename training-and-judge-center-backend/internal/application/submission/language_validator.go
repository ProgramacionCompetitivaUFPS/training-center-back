package submission

import (
	"path/filepath"
	"strings"

	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// languageConfig defines the valid compiler and file extensions for each language.
var languageConfig = map[string]struct {
	compiler   string
	extensions []string
	ext        string // canonical extension for storage path
}{
	"cpp20":     {compiler: "g++", extensions: []string{".cpp", ".cc", ".cxx"}, ext: ".cpp"},
	"java17":    {compiler: "javac", extensions: []string{".java"}, ext: ".java"},
	"python310": {compiler: "py", extensions: []string{".py"}, ext: ".py"},
}

// validateLanguage checks that language, compiler, and filename extension are
// mutually compatible. Returns the typed Language value object on success.
func validateLanguage(language, compiler, fileName string) (domainSubmission.Language, error) {
	langVO, err := domainSubmission.NewLanguage(language)
	if err != nil {
		return domainSubmission.Language{}, err
	}
	cfg, ok := languageConfig[language]
	if !ok {
		return domainSubmission.Language{}, apperror.NewBadRequest(domainSubmission.ErrCodeCompilerMismatch, "unsupported language")
	}
	if compiler != cfg.compiler {
		return domainSubmission.Language{}, apperror.NewBadRequest(domainSubmission.ErrCodeCompilerMismatch, "compiler does not match the selected language")
	}
	fileExt := strings.ToLower(filepath.Ext(fileName))
	for _, ext := range cfg.extensions {
		if fileExt == ext {
			return langVO, nil
		}
	}
	return domainSubmission.Language{}, apperror.NewBadRequest(domainSubmission.ErrCodeCompilerMismatch, "file extension does not match the selected language")
}
