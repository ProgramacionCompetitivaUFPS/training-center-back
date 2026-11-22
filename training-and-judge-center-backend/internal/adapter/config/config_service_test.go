package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	adapterconfig "github.com/training-judge-center/backend/internal/adapter/config"
	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func virtualObject() *config.VirtualObject {
	return &config.VirtualObject{
		SupportedLanguages: []string{"cpp20", "python310"},
		LanguageExtensions: map[string]string{".cpp": "cpp20", ".py": "python310"},
		MaxTimeLimitGlobal: 10000, MaxMemoryLimitGlobal: 1024,
		LanguageOverrides: []config.LanguageLimit{{Language: "python310", MaxTimeLimit: 5000, MaxMemoryLimit: 512}},
		Tags:              []string{"graphs", "dp"},
	}
}

func TestNewPlatformSettings_NilConfigIsAnInternalError(t *testing.T) {
	_, err := adapterconfig.NewPlatformSettings(nil)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindInternal, appErr.Kind)
}

func TestNewPlatformSettings_MapsLimitsLanguagesAndTags(t *testing.T) {
	settings, err := adapterconfig.NewPlatformSettings(virtualObject())

	require.NoError(t, err)
	time, memory := settings.GlobalLimits()
	assert.Equal(t, 10000, time)
	assert.Equal(t, 1024, memory)
	assert.True(t, settings.IsLanguageSupported("cpp20"))
	assert.False(t, settings.IsLanguageSupported("ruby"))
	language, ok := settings.LanguageByExtension(".py")
	assert.True(t, ok)
	assert.Equal(t, "python310", language)
	limit, ok := settings.LanguageLimit("python310")
	require.True(t, ok)
	assert.Equal(t, 5000, limit.MaxTimeLimit())
	assert.Equal(t, 512, limit.MaxMemoryLimit())
	assert.True(t, settings.IsValidTag("graphs"))
	assert.False(t, settings.IsValidTag("nope"))
}

func TestNewPlatformSettings_FillsDefaultsOnlyForTheFieldsLeftAtZero(t *testing.T) {
	settings, err := adapterconfig.NewPlatformSettings(virtualObject())
	require.NoError(t, err)
	assert.Equal(t, 10, settings.UploadMaxConcurrency())
	assert.Equal(t, 10, settings.MaxFileCountSample())
	assert.Equal(t, 200, settings.MaxZipUploadSizeMB())
	assert.Equal(t, 10000, settings.MaxFileCountTestCase())
	assert.Equal(t, 10, settings.MaxFileSizeDefaultMB())

	vo := virtualObject()
	vo.UploadMaxConcurrency, vo.MaxFileCountSample, vo.MaxZipUploadSizeMB = 4, 3, 50
	vo.MaxFileCountTestCase, vo.MaxFileSizeDefaultMB = 500, 5
	custom, err := adapterconfig.NewPlatformSettings(vo)
	require.NoError(t, err)
	assert.Equal(t, 4, custom.UploadMaxConcurrency())
	assert.Equal(t, 3, custom.MaxFileCountSample())
	assert.Equal(t, 50, custom.MaxZipUploadSizeMB())
	assert.Equal(t, 500, custom.MaxFileCountTestCase())
	assert.Equal(t, 5, custom.MaxFileSizeDefaultMB())
}

func TestNewPlatformSettings_RejectsAnInvalidLanguageOverride(t *testing.T) {
	vo := virtualObject()
	vo.LanguageOverrides = []config.LanguageLimit{{Language: "python310", MaxTimeLimit: 0, MaxMemoryLimit: 512}}

	_, err := adapterconfig.NewPlatformSettings(vo)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, apperror.KindValidation, appErr.Kind)
	assert.Equal(t, "maxTimeLimit", appErr.Details[0].Field)
}

func TestNewPlatformSettings_RejectsAnOverrideAboveTheGlobalLimit(t *testing.T) {
	vo := virtualObject()
	vo.LanguageOverrides = []config.LanguageLimit{{Language: "python310", MaxTimeLimit: 20000, MaxMemoryLimit: 512}}

	_, err := adapterconfig.NewPlatformSettings(vo)

	var appErr *apperror.AppError
	require.ErrorAs(t, err, &appErr)
	assert.Equal(t, "languageLimits.python310.maxTimeLimit", appErr.Details[0].Field)
}
