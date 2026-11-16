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
	SupportedLanguages   []string          `json:"supportedLanguages"`
	LanguageExtensions   map[string]string `json:"languageExtensions"`
	UploadMaxConcurrency int               `json:"uploadMaxConcurrency"`
	MaxFileCountSample   int               `json:"maxFileCountSample"`
	MaxZipUploadSizeMB int `json:"maxZipUploadSizeMB"`
	// MaxExtractedSizeMB caps the total uncompressed content of an imported ZIP.
	// The worker loads all test-case files into memory at once (parseTestCasesZip),
	// so this value directly determines peak worker memory usage:
	//   peak ≈ MaxExtractedSizeMB × maxConcurrent × 1.5  (GC overhead factor)
	//   maxConcurrent = floor(pod_cpu_cores) - docker_daemon_reserve_cores (min 1)
	// Current pod: 2 Gi memory, 1 CPU, 1 reserve core → maxConcurrent = 1.
	// Raise this value only after verifying the new math fits within the worker
	// pod's memory limit (deploy/k8s/judge/worker.yaml). If the pod is also
	// scaled up in CPU, re-derive maxConcurrent before raising this limit.
	MaxExtractedSizeMB int `json:"maxExtractedSizeMB"`
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
// is how maxZipUploadSizeMB kept the 200 MB that was deliberately lowered.
// The remaining fields are still defaulted in adapter/config.
func validateVirtualObject(vo *VirtualObject) error {
	if vo.MaxFileSizeTestCaseInputMB <= 0 {
		return fmt.Errorf("maxFileSizeTestCaseInputMB must be positive, got %d", vo.MaxFileSizeTestCaseInputMB)
	}
	if vo.MaxFileSizeTestCaseAnswerMB <= 0 {
		return fmt.Errorf("maxFileSizeTestCaseAnswerMB must be positive, got %d", vo.MaxFileSizeTestCaseAnswerMB)
	}
	if vo.MaxExtractedSizeMB <= 0 {
		return fmt.Errorf("maxExtractedSizeMB must be positive, got %d", vo.MaxExtractedSizeMB)
	}
	if vo.MaxFileSizeTestCaseInputMB > vo.MaxExtractedSizeMB {
		return fmt.Errorf("maxFileSizeTestCaseInputMB (%d) exceeds maxExtractedSizeMB (%d)",
			vo.MaxFileSizeTestCaseInputMB, vo.MaxExtractedSizeMB)
	}
	if vo.MaxFileSizeTestCaseAnswerMB > vo.MaxFileSizeTestCaseInputMB {
		return fmt.Errorf("maxFileSizeTestCaseAnswerMB (%d) exceeds maxFileSizeTestCaseInputMB (%d)",
			vo.MaxFileSizeTestCaseAnswerMB, vo.MaxFileSizeTestCaseInputMB)
	}
	return nil
}
