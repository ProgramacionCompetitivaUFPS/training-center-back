package judge

import (
	"context"
	"testing"

	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	"github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// TestValidatorRunner_UnsupportedLanguage_ReturnsInternal doesn't need any
// real tool — the dispatcher rejects the language before running anything.
func TestValidatorRunner_UnsupportedLanguage_ReturnsInternal(t *testing.T) {
	r := NewValidatorRunner()

	_, err := r.Run(context.Background(), appJudge.ValidatorRunRequest{
		Filename: "validator.rs",
		Language: submission.RestoreLanguage("rust"),
		Artifact: []byte("whatever"),
		Input:    []byte("1"),
	})
	assertAppErrorKind(t, err, apperror.KindInternal)
}
