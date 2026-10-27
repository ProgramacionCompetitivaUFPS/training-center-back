package config

import (
	"encoding/json"
	"fmt"
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
	// Per-file caps. The answer's also bounds the largest token cmd/compare can meet.
	MaxFileSizeTestCaseInputMB  int `json:"maxFileSizeTestCaseInputMB"`
	MaxFileSizeTestCaseAnswerMB int `json:"maxFileSizeTestCaseAnswerMB"`
	MaxFileCountTestCase        int `json:"maxFileCountTestCase"`
	MaxFileSizeDefaultMB        int `json:"maxFileSizeDefaultMB"`
	MaxTimeLimitGlobal          int `json:"maxTimeLimitGlobal"`
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

	if err := validateVirtualObject(&vo); err != nil {
		slog.Error("invalid virtual object config", "path", path, "error", err)
		os.Exit(1)
	}

	return &vo
}

// validateVirtualObject checks the test case file limits against each other. A
// missing cap is a broken config and not something to fill in: defaulting one
// is how maxFileSizeTestCaseMB kept the 200 MB that was deliberately lowered.
// The remaining fields are still defaulted in adapter/config.
func validateVirtualObject(vo *VirtualObject) error {
	if vo.MaxFileSizeTestCaseInputMB <= 0 {
		return fmt.Errorf("maxFileSizeTestCaseInputMB must be positive, got %d", vo.MaxFileSizeTestCaseInputMB)
	}
	if vo.MaxFileSizeTestCaseAnswerMB <= 0 {
		return fmt.Errorf("maxFileSizeTestCaseAnswerMB must be positive, got %d", vo.MaxFileSizeTestCaseAnswerMB)
	}
	if vo.MaxFileSizeTestCaseInputMB > vo.MaxFileSizeTestCaseMB {
		return fmt.Errorf("maxFileSizeTestCaseInputMB (%d) exceeds maxFileSizeTestCaseMB (%d)",
			vo.MaxFileSizeTestCaseInputMB, vo.MaxFileSizeTestCaseMB)
	}
	if vo.MaxFileSizeTestCaseAnswerMB > vo.MaxFileSizeTestCaseInputMB {
		return fmt.Errorf("maxFileSizeTestCaseAnswerMB (%d) exceeds maxFileSizeTestCaseInputMB (%d)",
			vo.MaxFileSizeTestCaseAnswerMB, vo.MaxFileSizeTestCaseInputMB)
	}
	return nil
}
