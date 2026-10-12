package problem

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

// Checkers and validators accept only these; solutions accept every platform language.
var allowedVerifierExtensions = []string{".py", ".cpp"}

func validateVerifierExtension(fileType, fileName string) error {
	ext := strings.ToLower(filepath.Ext(fileName))
	for _, allowed := range allowedVerifierExtensions {
		if ext == allowed {
			return nil
		}
	}
	return apperror.NewValidation([]apperror.FieldError{
		{Field: "file", Message: fmt.Sprintf("Unsupported %s file type (allowed: %s)", fileType, strings.Join(allowedVerifierExtensions, ", "))},
	})
}
