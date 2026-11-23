package config

import (
	"encoding/base64"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalVirtualObjectJSON = `{
  "supportedLanguages": ["cpp20"],
  "maxZipUploadSizeMB": 50,
  "maxExtractedSizeMB": 100,
  "maxFileSizeTestCaseInputMB": 10,
  "maxFileSizeTestCaseAnswerMB": 5,
  "maxTimeLimitGlobal": 10000,
  "maxMemoryLimitGlobal": 1024,
  "tags": ["graphs", "dp"]
}`

func writeVirtualObject(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "virtual_object.json")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

func TestGetEnv_ReturnsTheValueOrTheFallbackWhenUnsetOrEmpty(t *testing.T) {
	t.Setenv("CFG_PRESENT", "value")
	t.Setenv("CFG_EMPTY", "")

	assert.Equal(t, "value", getEnv("CFG_PRESENT", "fallback"))
	assert.Equal(t, "fallback", getEnv("CFG_EMPTY", "fallback"))
	assert.Equal(t, "fallback", getEnv("CFG_UNSET_KEY", "fallback"))
}

func TestGetEnvAsSlice_SplitsOnCommasAndTrimsSpaces(t *testing.T) {
	t.Setenv("CFG_LIST", " http://a.com ,http://b.com,  http://c.com")

	assert.Equal(t, []string{"http://a.com", "http://b.com", "http://c.com"}, getEnvAsSlice("CFG_LIST", "x"))
	assert.Equal(t, []string{"only"}, getEnvAsSlice("CFG_LIST_UNSET", "only"))
}

func TestLoadDatabase_DefaultsAndOverrides(t *testing.T) {
	for _, key := range []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME"} {
		t.Setenv(key, "")
	}

	defaults := LoadDatabase()
	assert.Equal(t, "localhost", defaults.DBHost)
	assert.Equal(t, "5432", defaults.DBPort)
	assert.Equal(t, "postgres", defaults.DBUser)
	assert.Equal(t, "training_center", defaults.DBName)
	assert.Empty(t, defaults.JWTSecret, "LoadDatabase must not demand secrets it never uses")

	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_PORT", "6543")
	t.Setenv("DB_NAME", "tc_prod")
	overridden := LoadDatabase()
	assert.Equal(t, "db.internal", overridden.DBHost)
	assert.Equal(t, "6543", overridden.DBPort)
	assert.Equal(t, "tc_prod", overridden.DBName)
}

func TestLoad_ReadsEveryRequiredSettingAndDecodesTheEncryptionKey(t *testing.T) {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	t.Setenv("JWT_SECRET", "s3cret")
	t.Setenv("GOOGLE_CLIENT_ID", "client-id")
	t.Setenv("ROTATION_CACHE_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(key))
	t.Setenv("VIRTUAL_OBJECT_CONFIG", writeVirtualObject(t, minimalVirtualObjectJSON))
	t.Setenv("JWT_EXPIRATION_HOURS", "12")
	t.Setenv("ALLOWED_ORIGINS", "http://a.com, http://b.com")
	t.Setenv("STORAGE_BACKEND", "")
	t.Setenv("PORT", "")

	cfg := Load()

	assert.Equal(t, "s3cret", cfg.JWTSecret)
	assert.Equal(t, "client-id", cfg.GoogleClientID)
	assert.Equal(t, key, cfg.RotationCacheEncryptionKey)
	assert.Equal(t, 12, cfg.JWTExpirationHours)
	assert.Equal(t, []string{"http://a.com", "http://b.com"}, cfg.AllowedOrigins)
	assert.Equal(t, "8080", cfg.Port)
	assert.Equal(t, "local", cfg.StorageBackend)
	require.NotNil(t, cfg.VirtualObject)
	assert.Equal(t, []string{"graphs", "dp"}, cfg.VirtualObject.Tags)
	assert.Equal(t, 10000, cfg.VirtualObject.MaxTimeLimitGlobal)
}

func TestLoadVirtualObject_ReadsTheConfiguredFile(t *testing.T) {
	t.Setenv("VIRTUAL_OBJECT_CONFIG", writeVirtualObject(t, minimalVirtualObjectJSON))

	vo := LoadVirtualObject()

	assert.Equal(t, 100, vo.MaxExtractedSizeMB)
	assert.Equal(t, []string{"cpp20"}, vo.SupportedLanguages)
}

// A process that cannot start exits with status 1; the case runs in a child
// process because os.Exit would end the test binary.
func TestFatalConfig_ExitsWithStatusOne(t *testing.T) {
	cases := map[string]func(t *testing.T) map[string]string{
		"missing required env": func(t *testing.T) map[string]string { return map[string]string{} },
		"invalid base64 key": func(t *testing.T) map[string]string {
			return map[string]string{"CFG_REQUIRED": "not base64 !!!"}
		},
		"missing virtual object file": func(t *testing.T) map[string]string {
			return map[string]string{"VIRTUAL_OBJECT_CONFIG": filepath.Join(t.TempDir(), "absent.json")}
		},
		"malformed virtual object json": func(t *testing.T) map[string]string {
			return map[string]string{"VIRTUAL_OBJECT_CONFIG": writeVirtualObject(t, "{not json")}
		},
		"invalid virtual object limits": func(t *testing.T) map[string]string {
			return map[string]string{"VIRTUAL_OBJECT_CONFIG": writeVirtualObject(t, `{"maxFileSizeTestCaseInputMB": 0}`)}
		},
	}
	for name, env := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestFatalConfigChild$")
			cmd.Env = append(os.Environ(), "CFG_FATAL_CASE="+name, "CFG_REQUIRED=")
			for k, v := range env(t) {
				cmd.Env = append(cmd.Env, k+"="+v)
			}

			err := cmd.Run()

			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr)
			assert.Equal(t, 1, exitErr.ExitCode())
		})
	}
}

func TestFatalConfigChild(t *testing.T) {
	switch os.Getenv("CFG_FATAL_CASE") {
	case "":
		t.Skip("only runs as a child of TestFatalConfig_ExitsWithStatusOne")
	case "missing required env":
		getRequiredEnv("CFG_REQUIRED")
	case "invalid base64 key":
		getRequiredBase64Env("CFG_REQUIRED")
	default:
		loadVirtualObject()
	}
}
