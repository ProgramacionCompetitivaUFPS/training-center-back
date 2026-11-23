package main

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	adapterjudge "github.com/training-judge-center/backend/internal/adapter/judge"
	judgepool "github.com/training-judge-center/backend/internal/adapter/judge/pool"
	"github.com/training-judge-center/backend/internal/config"
)

// Floors applied when the matching judge config key is missing or non-positive.
// KnownFields rejects unknown keys but says nothing about absent ones, so every
// optional value needs one.
const (
	defaultIdleTimeoutMinutes          = 10
	defaultDockerDaemonReserveBytes    = 512 << 20 // 512 MiB
	defaultDockerDaemonReserveCores    = 1
	defaultStaleRunningAfterMinutes    = 10
	defaultStalePendingAfterMinutes    = 2
	defaultStaleValidationAfterMinutes = 20
	defaultTestDataDir                 = "/test-data"
	defaultTestDataBudgetBytes         = 3072 << 20 // 3 GiB: twelve problems like the largest one in use
)

// judgeLanguageConfig holds what is true of a language regardless of which pool
// runs it: its image and how its code is compiled and executed.
type judgeLanguageConfig struct {
	Image           string  `yaml:"image"`
	Extension       string  `yaml:"extension"`
	MemoryFactor    float64 `yaml:"memoryFactor"`
	CompileCmd      string  `yaml:"compileCmd"`
	RunCmd          string  `yaml:"runCmd"`
	ArtifactSource  string  `yaml:"artifactSource"`
	ArtifactCompile string  `yaml:"artifactCompile"`
	ArtifactPath    string  `yaml:"artifactPath"`
	ArtifactRun     string  `yaml:"artifactRun"`
}

// judgePoolLanguageConfig is the sizing of one language's containers inside one
// pool. The same language can need different limits depending on what the
// container is used for, so this lives under the pool, not under the language.
type judgePoolLanguageConfig struct {
	CPU         string `yaml:"cpu"`
	MemoryBytes int64  `yaml:"memoryBytes"`
}

type judgePoolConfig struct {
	BudgetBytes int64                              `yaml:"budgetBytes"`
	Languages   map[string]judgePoolLanguageConfig `yaml:"languages"`
}

type judgeSection struct {
	IdleTimeoutMinutes int `yaml:"idleTimeoutMinutes"`
	// Both reserves are carved out of the dind container's limits, where the
	// Docker daemon and the containers it spawns live. Neither has anything to
	// do with the worker process, which runs in a separate container with its
	// own cgroup and its own limits.
	DockerDaemonReserveBytes    int64 `yaml:"dockerDaemonReserveBytes"`
	DockerDaemonReserveCores    int   `yaml:"dockerDaemonReserveCores"`
	StaleRunningAfterMinutes    int   `yaml:"staleRunningAfterMinutes"`
	StalePendingAfterMinutes    int   `yaml:"stalePendingAfterMinutes"`
	StaleValidationAfterMinutes int   `yaml:"staleValidationAfterMinutes"`
	// TestDataDir is where test case archives are cached, on a volume of the worker pod and not
	// in the dind container. TestDataBudgetBytes is the most disk they may take.
	TestDataDir         string                         `yaml:"testDataDir"`
	TestDataBudgetBytes int64                          `yaml:"testDataBudgetBytes"`
	Languages           map[string]judgeLanguageConfig `yaml:"languages"`
	Pools               map[string]judgePoolConfig     `yaml:"pools"`
}

type judgeConfigFile struct {
	Judge judgeSection `yaml:"judge"`
}

func loadJudgeConfig() judgeConfigFile {
	path := getEnv("JUDGE_CONFIG", "config/judge_config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		slog.Error("worker: failed to read judge config", "path", path, "error", err)
		os.Exit(1)
	}
	var cfg judgeConfigFile
	// Strict: an unknown key means the file and these structs have drifted, and
	// a silently ignored key would show up much later as a zero-valued limit.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		slog.Error("worker: failed to parse judge config", "path", path, "error", err)
		os.Exit(1)
	}
	if err := validateJudgeConfig(cfg); err != nil {
		slog.Error("worker: invalid judge config", "path", path, "error", err)
		os.Exit(1)
	}
	applyJudgeConfigDefaults(&cfg)
	return cfg
}

// validateJudgeConfig reports the first inconsistency found. It is separate from
// loadJudgeConfig, which exits the process, so it can be tested.
func validateJudgeConfig(cfg judgeConfigFile) error {
	if len(cfg.Judge.Languages) == 0 {
		return errors.New("no languages defined")
	}
	for _, name := range []string{poolHeavy, poolLight} {
		if _, ok := cfg.Judge.Pools[name]; !ok {
			return fmt.Errorf("no %q pool defined", name)
		}
	}
	// A pool sizing a language with no image would only fail when that language
	// is first claimed, mid-judging.
	for poolName, pool := range cfg.Judge.Pools {
		if pool.BudgetBytes <= 0 {
			return fmt.Errorf("pool %q: budgetBytes is not positive", poolName)
		}
		if len(pool.Languages) == 0 {
			return fmt.Errorf("pool %q sizes no languages", poolName)
		}
		for lang, sizing := range pool.Languages {
			if _, ok := cfg.Judge.Languages[lang]; !ok {
				return fmt.Errorf("pool %q sizes undeclared language %q", poolName, lang)
			}
			// An empty cpu parses as no limit at all, which silently breaks the
			// one-CPU-per-container assumption the concurrency formula rests on.
			if sizing.CPU == "" {
				return fmt.Errorf("pool %q, language %q: no cpu", poolName, lang)
			}
			if sizing.MemoryBytes <= 0 {
				return fmt.Errorf("pool %q, language %q: memoryBytes is not positive", poolName, lang)
			}
		}
	}
	// The default checker is claimed by every submission to a problem without a
	// custom one, which is most of the traffic. Missing here, the first such
	// submission fails with "unknown language" — mid-competition.
	if cfg.Judge.Languages[adapterjudge.CompareLanguage].ArtifactRun == "" {
		return fmt.Errorf("language %q has no artifactRun: nothing would run for a problem without a custom checker", adapterjudge.CompareLanguage)
	}
	if _, ok := cfg.Judge.Pools[poolLight].Languages[adapterjudge.CompareLanguage]; !ok {
		return fmt.Errorf("pool %q does not size %q, so the default checker could never be claimed", poolLight, adapterjudge.CompareLanguage)
	}
	for lang, lc := range cfg.Judge.Languages {
		if lc.Image == "" {
			return fmt.Errorf("language %q has no image", lang)
		}
		// A language with a runCmd is one solutions are written in, so it must
		// be sized by the heavy pool and able to build a checker or validator.
		// Entries without one are not languages code is sent in.
		if lc.RunCmd == "" {
			continue
		}
		if lc.Extension == "" {
			return fmt.Errorf("language %q runs solutions but has no extension", lang)
		}
		// The container is sized as the problem's limit times this, so a language
		// that omits it silently hands its runtime's overhead to the contestant.
		if lc.MemoryFactor < 1 {
			return fmt.Errorf("language %q runs solutions but its memoryFactor is under 1 (%v)", lang, lc.MemoryFactor)
		}
		if _, ok := cfg.Judge.Pools[poolHeavy].Languages[lang]; !ok {
			return fmt.Errorf("language %q runs solutions but pool %q does not size it", lang, poolHeavy)
		}
		// The artifact fields below mean a checker or validator can be written
		// in this language, and those run in the light pool.
		if _, ok := cfg.Judge.Pools[poolLight].Languages[lang]; !ok {
			return fmt.Errorf("language %q can carry a checker but pool %q does not size it", lang, poolLight)
		}
		for _, f := range []struct{ name, value string }{
			{"artifactSource", lc.ArtifactSource},
			{"artifactCompile", lc.ArtifactCompile},
			{"artifactPath", lc.ArtifactPath},
			{"artifactRun", lc.ArtifactRun},
		} {
			if f.value == "" {
				return fmt.Errorf("language %q: %s is empty", lang, f.name)
			}
			if !strings.Contains(f.value, adapterjudge.ArtifactNamePlaceholder) {
				return fmt.Errorf("language %q: %s has no %s", lang, f.name, adapterjudge.ArtifactNamePlaceholder)
			}
		}
	}
	return nil
}

// validatePoolBudgets checks the config against the machine it runs on. Its
// numbers come from the environment rather than the file, which is why it is
// separate from validateJudgeConfig.
// poolConfigFor joins the two config sections a pool needs: the pool says how
// big a container is, the language which image it runs.
func poolConfigFor(cfg judgeConfigFile, poolName string, idleTimeout time.Duration, volumeSource string) judgepool.PoolConfig {
	pool := cfg.Judge.Pools[poolName]
	languages := make(map[string]judgepool.LanguageConfig, len(pool.Languages))
	for lang, sizing := range pool.Languages {
		languages[lang] = judgepool.LanguageConfig{
			Image:       cfg.Judge.Languages[lang].Image,
			CPU:         sizing.CPU,
			MemoryBytes: sizing.MemoryBytes,
		}
	}
	return judgepool.PoolConfig{
		BudgetBytes:        pool.BudgetBytes,
		IdleTimeout:        idleTimeout,
		ReapInterval:       reapInterval,
		Languages:          languages,
		SharedVolumeSource: volumeSource,
		// The light pool only ever reads the judging directory.
		SharedVolumeReadOnly: poolName == poolLight,
	}
}

// validatePoolBudgets checks the two things the config file cannot know on its
// own: that the pools fit inside the dind container, and D13's invariant that a
// pool can serve maxConcurrent claims of its largest language without blocking.
func validatePoolBudgets(cfg judgeConfigFile, dindMemBytes int64, maxConcurrent int) error {
	var total int64
	for _, pool := range cfg.Judge.Pools {
		total += pool.BudgetBytes
	}
	total += cfg.Judge.DockerDaemonReserveBytes
	if total > dindMemBytes {
		return fmt.Errorf("pool budgets plus the daemon reserve add up to %d bytes, over the %d the dind container has",
			total, dindMemBytes)
	}
	// D13: every judging holds one container in each pool, so a pool that
	// cannot fit maxConcurrent of its largest language blocks forever once they
	// are all busy — a silent hang instead of a loud failure.
	for _, name := range slices.Sorted(maps.Keys(cfg.Judge.Pools)) {
		pool := cfg.Judge.Pools[name]
		var largest int64
		for _, sizing := range pool.Languages {
			largest = max(largest, sizing.MemoryBytes)
		}
		if need := int64(maxConcurrent) * largest; pool.BudgetBytes < need {
			return fmt.Errorf("pool %q: budget %d is under the %d needed for %d concurrent containers of its largest language (%d bytes)",
				name, pool.BudgetBytes, need, maxConcurrent, largest)
		}
	}
	return nil
}

// applyJudgeConfigDefaults floors the optional values. KnownFields rejects
// unknown keys but says nothing about absent ones.
func applyJudgeConfigDefaults(cfg *judgeConfigFile) {
	if cfg.Judge.IdleTimeoutMinutes <= 0 {
		cfg.Judge.IdleTimeoutMinutes = defaultIdleTimeoutMinutes
	}
	if cfg.Judge.DockerDaemonReserveBytes <= 0 {
		cfg.Judge.DockerDaemonReserveBytes = defaultDockerDaemonReserveBytes
	}
	if cfg.Judge.DockerDaemonReserveCores <= 0 {
		cfg.Judge.DockerDaemonReserveCores = defaultDockerDaemonReserveCores
	}
	if cfg.Judge.StaleRunningAfterMinutes <= 0 {
		cfg.Judge.StaleRunningAfterMinutes = defaultStaleRunningAfterMinutes
	}
	if cfg.Judge.StalePendingAfterMinutes <= 0 {
		cfg.Judge.StalePendingAfterMinutes = defaultStalePendingAfterMinutes
	}
	if cfg.Judge.StaleValidationAfterMinutes <= 0 {
		cfg.Judge.StaleValidationAfterMinutes = defaultStaleValidationAfterMinutes
	}
	if cfg.Judge.TestDataDir == "" {
		cfg.Judge.TestDataDir = defaultTestDataDir
	}
	if cfg.Judge.TestDataBudgetBytes <= 0 {
		cfg.Judge.TestDataBudgetBytes = defaultTestDataBudgetBytes
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getRequiredEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("worker: required env var is not set", "key", key)
		os.Exit(1)
	}
	return v
}

func getRequiredEnvInt64(key string) int64 {
	s := getRequiredEnv(key)
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		slog.Error("worker: env var must be an integer", "key", key, "value", s)
		os.Exit(1)
	}
	return n
}

// ensureSharedVolume fails fast when the judging volume is not mounted where it
// is expected. Docker creates a missing bind source as an empty directory, so a
// broken mount would otherwise surface as every submission failing its checker.
func ensureSharedVolume(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	// 0711 lets a sandbox enter a judging directory whose path it was handed,
	// while leaving it unable to list the root and discover anyone else's.
	if err := os.Chmod(path, 0o711); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

// testDataConfig turns the shared platform limits and the worker's own sizing into what
// the test data cache enforces.
func testDataConfig(cfg judgeConfigFile, vo *config.VirtualObject) adapterjudge.TestDataConfig {
	return adapterjudge.TestDataConfig{
		Dir:         cfg.Judge.TestDataDir,
		BudgetBytes: cfg.Judge.TestDataBudgetBytes,
		Limits: adapterjudge.TestDataLimits{
			MaxZipBytes:       int64(vo.MaxZipUploadSizeMB) << 20,
			MaxExtractedBytes: int64(vo.MaxExtractedSizeMB) << 20,
			MaxFiles:          vo.MaxFileCountTestCase,
			MaxInputBytes:     int64(vo.MaxFileSizeTestCaseInputMB) << 20,
			MaxAnswerBytes:    int64(vo.MaxFileSizeTestCaseAnswerMB) << 20,
		},
	}
}

// validateTestDataBudget checks the invariant that keeps a judging from waiting for disk forever.
// A problem being cached holds its archive while it waits for room to extract it, so in the worst
// case every one of the maxConcurrent judgings is in that state; for one of them to always be able
// to go on, the budget must cover all those archives plus the extraction of the largest package.
// Judgings that wait beyond that always do so for one that is about to end.
func validateTestDataBudget(cfg adapterjudge.TestDataConfig, maxConcurrent int) error {
	need := int64(maxConcurrent)*cfg.Limits.MaxZipBytes + cfg.Limits.MaxExtractedBytes
	if cfg.BudgetBytes < need {
		return fmt.Errorf("test data budget %d is under the %d needed for %d concurrent archives (%d bytes each) and the extraction of the largest package (%d bytes)",
			cfg.BudgetBytes, need, maxConcurrent, cfg.Limits.MaxZipBytes, cfg.Limits.MaxExtractedBytes)
	}
	return nil
}
