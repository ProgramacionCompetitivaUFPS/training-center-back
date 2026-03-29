package config

import (
	"encoding/json"
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
	UploadMaxConcurrency  int             `json:"uploadMaxConcurrency"`
	MaxFileCountSample    int             `json:"maxFileCountSample"`
	MaxFileSizeTestCaseMB int             `json:"maxFileSizeTestCaseMB"`
	MaxFileCountTestCase  int             `json:"maxFileCountTestCase"`
	MaxFileSizeDefaultMB  int             `json:"maxFileSizeDefaultMB"`
	MaxTimeLimitGlobal   int             `json:"maxTimeLimitGlobal"`
	MaxMemoryLimitGlobal int             `json:"maxMemoryLimitGlobal"`
	LanguageOverrides    []LanguageLimit `json:"languageOverrides"`
	Tags                 []string        `json:"tags"`
}

func loadVirtualObject() *VirtualObject {
	path := getEnv("VIRTUAL_OBJECT_CONFIG", "config/virtual_object.json")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var vo VirtualObject
	if err := json.Unmarshal(data, &vo); err != nil {
		return nil
	}

	return &vo
}
