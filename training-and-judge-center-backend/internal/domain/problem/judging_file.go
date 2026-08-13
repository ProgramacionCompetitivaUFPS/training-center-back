package problem

import (
	"time"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type JudgingFile struct {
	filename    string
	fileKey     string
	language    string
	compiledKey *string
	compiledAt  *time.Time
}

func newJudgingFile(filename, fileKey, language, unsupportedMsg string) (JudgingFile, error) {
	if language == "" {
		return JudgingFile{}, apperror.NewValidation([]apperror.FieldError{
			{Field: "file", Message: unsupportedMsg},
		})
	}
	return JudgingFile{filename: filename, fileKey: fileKey, language: language}, nil
}

func NewSolutionFile(filename, fileKey, language string) (JudgingFile, error) {
	return newJudgingFile(filename, fileKey, language, "Unsupported solution file type")
}

func NewVerifierFile(filename, fileKey, language string) (JudgingFile, error) {
	return newJudgingFile(filename, fileKey, language, "Unsupported checker/validator file type")
}

func (j JudgingFile) Filename() string       { return j.filename }
func (j JudgingFile) FileKey() string        { return j.fileKey }
func (j JudgingFile) Language() string       { return j.language }
func (j JudgingFile) CompiledKey() *string   { return j.compiledKey }
func (j JudgingFile) CompiledAt() *time.Time { return j.compiledAt }

func RestoreJudgingFile(filename, fileKey, language string, compiledKey *string, compiledAt *time.Time) JudgingFile {
	return JudgingFile{
		filename:    filename,
		fileKey:     fileKey,
		language:    language,
		compiledKey: compiledKey,
		compiledAt:  compiledAt,
	}
}
