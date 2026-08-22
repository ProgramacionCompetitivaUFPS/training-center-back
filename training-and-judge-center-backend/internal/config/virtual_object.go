package config

import (
	"encoding/json"
	"log/slog"
	"os"
)

type LanguageLimit struct {
	Language       string `json:"language"`
	MaxTimeLimit   int    `json:"maxTimeLimit"`
	MaxMemoryLimit int    `json:"maxMemoryLimit"`
}

type VirtualObject struct {
	SupportedLanguages    []string          `json:"supportedLanguages"`
	LanguageExtensions    map[string]string `json:"languageExtensions"`
	UploadMaxConcurrency  int               `json:"uploadMaxConcurrency"`
	MaxFileCountSample    int               `json:"maxFileCountSample"`
	MaxFileSizeTestCaseMB int               `json:"maxFileSizeTestCaseMB"`
	MaxFileCountTestCase  int               `json:"maxFileCountTestCase"`
	MaxFileSizeDefaultMB  int               `json:"maxFileSizeDefaultMB"`
	MaxTimeLimitGlobal    int               `json:"maxTimeLimitGlobal"`
	// MaxMemoryLimitGlobal is the largest memory limit a problem may declare, and
	// it reaches past the API: the judge sizes its heavy pool containers to cover
	// it, times each language's memoryFactor. Raising it without resizing them
	// makes the pool cap the request and hand the solution less memory than the
	// problem promised, silently. A test in cmd/worker guards the relationship.
	MaxMemoryLimitGlobal int             `json:"maxMemoryLimitGlobal"`
	LanguageOverrides    []LanguageLimit `json:"languageOverrides"`
	Tags                 []string        `json:"tags"`
}

func loadVirtualObject() *VirtualObject {
	path := getEnv("VIRTUAL_OBJECT_CONFIG", "config/virtual_object.json")

	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("failed to read virtual object config", "path", path, "error", err)
		os.Exit(1)
	}

	var vo VirtualObject
	if err := json.Unmarshal(data, &vo); err != nil {
		slog.Error("failed to parse virtual object config", "path", path, "error", err)
		os.Exit(1)
	}

	return &vo
}
