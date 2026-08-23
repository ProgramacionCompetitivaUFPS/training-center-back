package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	adapterjudge "github.com/training-judge-center/backend/internal/adapter/judge"
	"github.com/training-judge-center/backend/internal/config"
	"gopkg.in/yaml.v3"
)

const (
	realConfigPath = "../../config/judge_config.yaml"
	testLang       = "cpp20"
	// maxConcurrentInTests is what validConfig is sized for, exactly.
	maxConcurrentInTests = 2
)

// The dind container limits, mirrored from deploy/k8s/judge/worker.yaml. They
// are the numbers the shipped budgets were chosen against.
const (
	clusterDindMemBytes = 24 << 30 // limits.memory: 24Gi
	clusterDindCores    = 6        // limits.cpu: "6"
)

// decodeRealConfig parses the config the worker actually ships with, the same
// strict way loadJudgeConfig does. loadJudgeConfig itself calls os.Exit on
// failure, so the decoding is exercised here instead.
func decodeRealConfig(t *testing.T) judgeConfigFile {
	t.Helper()
	data, err := os.ReadFile(realConfigPath)
	if err != nil {
		t.Fatalf("could not read %s: %v", realConfigPath, err)
	}
	var cfg judgeConfigFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		t.Fatalf("could not decode %s: %v", realConfigPath, err)
	}
	return cfg
}

// validConfig is the smallest config satisfying every rule. Each rule test
// breaks exactly one thing in it.
func validConfig() judgeConfigFile {
	return judgeConfigFile{Judge: judgeSection{
		Languages: map[string]judgeLanguageConfig{
			testLang: {
				Image:           "judge-runner:cpp20",
				Extension:       "cpp",
				MemoryFactor:    1.0,
				CompileCmd:      "g++ -o /sandbox/Solution /sandbox/Solution.cpp",
				RunCmd:          "/sandbox/Solution",
				ArtifactSource:  "/sandbox/{name}.cpp",
				ArtifactCompile: "g++ -o /sandbox/{name} /sandbox/{name}.cpp",
				ArtifactPath:    "/sandbox/{name}",
				ArtifactRun:     "/sandbox/{name}",
			},
			// The default checker: image and artifactRun only, like the shipped config.
			adapterjudge.CompareLanguage: {
				Image:       "judge-runner:compare",
				ArtifactRun: "/usr/local/bin/compare",
			},
		},
		// The budgets are exactly what maxConcurrentInTests demands, so any
		// mutation of a size or a budget shows up as a failure.
		Pools: map[string]judgePoolConfig{
			poolHeavy: {BudgetBytes: 2 << 30, Languages: map[string]judgePoolLanguageConfig{
				testLang: {CPU: "1", MemoryBytes: 1 << 30},
			}},
			poolLight: {BudgetBytes: 1 << 30, Languages: map[string]judgePoolLanguageConfig{
				testLang:                     {CPU: "1", MemoryBytes: 512 << 20},
				adapterjudge.CompareLanguage: {CPU: "1", MemoryBytes: 128 << 20},
			}},
		},
	}}
}

// withLang replaces a language entry: map values are not addressable in Go, so
// a field cannot be assigned in place.
func withLang(cfg *judgeConfigFile, mutate func(*judgeLanguageConfig)) {
	lc := cfg.Judge.Languages[testLang]
	mutate(&lc)
	cfg.Judge.Languages[testLang] = lc
}

func TestValidateJudgeConfig_AcceptsAValidConfig(t *testing.T) {
	if err := validateJudgeConfig(validConfig()); err != nil {
		t.Fatalf("expected the valid config to pass, got: %v", err)
	}
}

// The config the worker ships with has to satisfy the rules the worker enforces
// at startup, asserted by calling the real function instead of restating them.
func TestValidateJudgeConfig_AcceptsTheShippedConfig(t *testing.T) {
	if err := validateJudgeConfig(decodeRealConfig(t)); err != nil {
		t.Fatalf("%s does not pass startup validation: %v", realConfigPath, err)
	}
}

func TestValidateJudgeConfig_RejectsBrokenConfigs(t *testing.T) {
	tests := []struct {
		name    string
		breaks  func(*judgeConfigFile)
		wantErr string
	}{
		{
			name:    "no languages at all",
			breaks:  func(c *judgeConfigFile) { c.Judge.Languages = nil },
			wantErr: "no languages defined",
		},
		{
			name:    "no heavy pool",
			breaks:  func(c *judgeConfigFile) { delete(c.Judge.Pools, poolHeavy) },
			wantErr: `no "heavy" pool defined`,
		},
		{
			name:    "no light pool",
			breaks:  func(c *judgeConfigFile) { delete(c.Judge.Pools, poolLight) },
			wantErr: `no "light" pool defined`,
		},
		{
			name: "a pool has no budget",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolLight] = judgePoolConfig{Languages: c.Judge.Pools[poolLight].Languages}
			},
			wantErr: `pool "light": budgetBytes is not positive`,
		},
		{
			// The budget is set because a pool with none trips the budget rule first.
			name:    "a pool sizes nothing",
			breaks:  func(c *judgeConfigFile) { c.Judge.Pools["extra"] = judgePoolConfig{BudgetBytes: 1} },
			wantErr: `pool "extra" sizes no languages`,
		},
		{
			name: "a pool sizes a language that is not declared",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolHeavy].Languages["rust"] = judgePoolLanguageConfig{CPU: "1", MemoryBytes: 1}
			},
			wantErr: `sizes undeclared language "rust"`,
		},
		{
			name: "a sized language has no cpu",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolHeavy].Languages[testLang] = judgePoolLanguageConfig{MemoryBytes: 1}
			},
			wantErr: "no cpu",
		},
		{
			name: "a sized language has no memory",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolHeavy].Languages[testLang] = judgePoolLanguageConfig{CPU: "1"}
			},
			wantErr: "memoryBytes is not positive",
		},
		{
			name:    "a language has no image",
			breaks:  func(c *judgeConfigFile) { withLang(c, func(l *judgeLanguageConfig) { l.Image = "" }) },
			wantErr: `language "cpp20" has no image`,
		},
		{
			name:    "a language that runs solutions has no extension",
			breaks:  func(c *judgeConfigFile) { withLang(c, func(l *judgeLanguageConfig) { l.Extension = "" }) },
			wantErr: "has no extension",
		},
		{
			// Added rather than removed: emptying the pool would trip the
			// "sizes no languages" rule first.
			name: "the heavy pool does not size a language that runs solutions",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Languages["python310"] = judgeLanguageConfig{
					Image:           "judge-runner:python310",
					Extension:       "py",
					MemoryFactor:    1.0,
					RunCmd:          "python3 /sandbox/Solution.py",
					ArtifactSource:  "/sandbox/{name}.py",
					ArtifactCompile: "python3 -m py_compile /sandbox/{name}.py",
					ArtifactPath:    "/sandbox/{name}.py",
					ArtifactRun:     "python3 /sandbox/{name}.py",
				}
			},
			wantErr: `language "python310" runs solutions but pool "heavy" does not size it`,
		},
		{
			// A checker or validator can be written in any language solutions are,
			// and those run in the light pool.
			name: "the light pool does not size a language that runs solutions",
			breaks: func(c *judgeConfigFile) {
				delete(c.Judge.Pools[poolLight].Languages, testLang)
				c.Judge.Pools[poolLight].Languages["python310"] = judgePoolLanguageConfig{CPU: "1", MemoryBytes: 1}
				c.Judge.Languages["python310"] = judgeLanguageConfig{Image: "judge-runner:python310"}
			},
			wantErr: `language "cpp20" can carry a checker but pool "light" does not size it`,
		},
		{
			name:    "a language that runs solutions has no memoryFactor",
			breaks:  func(c *judgeConfigFile) { withLang(c, func(l *judgeLanguageConfig) { l.MemoryFactor = 0 }) },
			wantErr: "memoryFactor is under 1",
		},
		{
			name:    "an artifact field is empty",
			breaks:  func(c *judgeConfigFile) { withLang(c, func(l *judgeLanguageConfig) { l.ArtifactPath = "" }) },
			wantErr: "artifactPath is empty",
		},
		{
			name: "an artifact field lost the name placeholder",
			breaks: func(c *judgeConfigFile) {
				withLang(c, func(l *judgeLanguageConfig) { l.ArtifactCompile = "g++ -o /sandbox/{nombre} /sandbox/x.cpp" })
			},
			wantErr: "artifactCompile has no {name}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.breaks(&cfg)

			err := validateJudgeConfig(cfg)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected the error to mention %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// A language with no runCmd is not one solutions are written in, so none of the
// solution-side rules reach it. That is what will keep the compare image, which
// only runs a fixed binary, from needing a heavy pool entry.
// A language nobody writes solutions in — the default checker is one, and any
// future tool would be another — is exempt from the rules that only make sense
// for a language that runs contestant code.
func TestValidateJudgeConfig_ExemptsLanguagesWithoutRunCmd(t *testing.T) {
	cfg := validConfig()
	cfg.Judge.Languages["some-tool"] = judgeLanguageConfig{Image: "judge-runner:some-tool"}

	if err := validateJudgeConfig(cfg); err != nil {
		t.Fatalf("expected a language without runCmd to be exempt, got: %v", err)
	}
}

func TestApplyJudgeConfigDefaults_FloorsAbsentValues(t *testing.T) {
	var cfg judgeConfigFile

	applyJudgeConfigDefaults(&cfg)

	for _, f := range []struct {
		name string
		got  int
		want int
	}{
		{"idleTimeoutMinutes", cfg.Judge.IdleTimeoutMinutes, defaultIdleTimeoutMinutes},
		{"dockerDaemonReserveCores", cfg.Judge.DockerDaemonReserveCores, defaultDockerDaemonReserveCores},
		{"staleRunningAfterMinutes", cfg.Judge.StaleRunningAfterMinutes, defaultStaleRunningAfterMinutes},
		{"staleValidationAfterMinutes", cfg.Judge.StaleValidationAfterMinutes, defaultStaleValidationAfterMinutes},
	} {
		if f.got != f.want {
			t.Errorf("%s: got %d, want the default %d", f.name, f.got, f.want)
		}
	}
	if cfg.Judge.DockerDaemonReserveBytes != defaultDockerDaemonReserveBytes {
		t.Errorf("dockerDaemonReserveBytes: got %d, want the default %d", cfg.Judge.DockerDaemonReserveBytes, defaultDockerDaemonReserveBytes)
	}
}

func TestApplyJudgeConfigDefaults_KeepsProvidedValues(t *testing.T) {
	cfg := judgeConfigFile{Judge: judgeSection{
		IdleTimeoutMinutes:          1,
		DockerDaemonReserveBytes:    2,
		DockerDaemonReserveCores:    3,
		StaleRunningAfterMinutes:    4,
		StaleValidationAfterMinutes: 5,
	}}

	applyJudgeConfigDefaults(&cfg)

	if cfg.Judge.IdleTimeoutMinutes != 1 || cfg.Judge.DockerDaemonReserveBytes != 2 ||
		cfg.Judge.DockerDaemonReserveCores != 3 || cfg.Judge.StaleRunningAfterMinutes != 4 ||
		cfg.Judge.StaleValidationAfterMinutes != 5 {
		t.Errorf("a provided value was overwritten by a default: %+v", cfg.Judge)
	}
}

// Not covered by the defaults test: a mistyped struct tag leaves the field at
// zero, and applyJudgeConfigDefaults would then hide that behind a default.
// This reads the shipped file before any defaulting happens.
func TestRealConfig_ScalarsComeFromTheFile(t *testing.T) {
	j := decodeRealConfig(t).Judge

	for _, f := range []struct {
		name  string
		value int
	}{
		{"idleTimeoutMinutes", j.IdleTimeoutMinutes},
		{"dockerDaemonReserveCores", j.DockerDaemonReserveCores},
		{"staleRunningAfterMinutes", j.StaleRunningAfterMinutes},
		{"staleValidationAfterMinutes", j.StaleValidationAfterMinutes},
	} {
		if f.value <= 0 {
			t.Errorf("%s: got %d, want > 0 — likely a struct tag that no longer matches the YAML", f.name, f.value)
		}
	}
	if j.DockerDaemonReserveBytes <= 0 {
		t.Errorf("dockerDaemonReserveBytes: got %d, want > 0", j.DockerDaemonReserveBytes)
	}
}

func TestValidatePoolBudgets_AcceptsAConfigThatFits(t *testing.T) {
	if err := validatePoolBudgets(validConfig(), 4<<30, maxConcurrentInTests); err != nil {
		t.Fatalf("expected the valid config to fit, got: %v", err)
	}
}

// The shipped budgets were written for one specific machine, so they are worth
// checking against it rather than only against themselves.
func TestValidatePoolBudgets_TheShippedConfigFitsTheCluster(t *testing.T) {
	cfg := decodeRealConfig(t)
	maxConcurrent := clusterDindCores - cfg.Judge.DockerDaemonReserveCores

	if err := validatePoolBudgets(cfg, clusterDindMemBytes, maxConcurrent); err != nil {
		t.Fatalf("%s does not fit the dind container in deploy/k8s/judge/worker.yaml: %v", realConfigPath, err)
	}
}

func TestValidatePoolBudgets_RejectsConfigsThatDoNotFit(t *testing.T) {
	tests := []struct {
		name          string
		breaks        func(*judgeConfigFile)
		dindMemBytes  int64
		maxConcurrent int
		wantErr       string
	}{
		{
			// validConfig's two budgets add up to 3 GiB exactly.
			name:          "the budgets alone overflow the dind container",
			breaks:        func(*judgeConfigFile) {},
			dindMemBytes:  3<<30 - 1,
			maxConcurrent: maxConcurrentInTests,
			wantErr:       "over the",
		},
		{
			name:          "the daemon reserve is what pushes them over",
			breaks:        func(c *judgeConfigFile) { c.Judge.DockerDaemonReserveBytes = 1 },
			dindMemBytes:  3 << 30,
			maxConcurrent: maxConcurrentInTests,
			wantErr:       "over the",
		},
		{
			name:          "the heavy pool cannot hold maxConcurrent of its largest language",
			breaks:        func(*judgeConfigFile) {},
			dindMemBytes:  4 << 30,
			maxConcurrent: maxConcurrentInTests + 1,
			wantErr:       `pool "heavy": budget`,
		},
		{
			// Every pool in validConfig sizes a single language, so on its own it
			// cannot tell "largest" from "smallest".
			name: "the budget covers the smallest language but not the largest",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolHeavy].Languages["heavier"] = judgePoolLanguageConfig{CPU: "1", MemoryBytes: 2 << 30}
			},
			dindMemBytes:  4 << 30,
			maxConcurrent: maxConcurrentInTests,
			wantErr:       `pool "heavy": budget`,
		},
		{
			// The light pool is checked too, not just whichever comes first.
			name: "the light pool cannot hold maxConcurrent of its largest language",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolLight] = judgePoolConfig{
					BudgetBytes: 512 << 20,
					Languages:   c.Judge.Pools[poolLight].Languages,
				}
			},
			dindMemBytes:  4 << 30,
			maxConcurrent: maxConcurrentInTests,
			wantErr:       `pool "light": budget`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.breaks(&cfg)

			err := validatePoolBudgets(cfg, tt.dindMemBytes, tt.maxConcurrent)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected the error to mention %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

// The judge reads one exit code as "out of memory" for every language, and for
// Java that is only true because of this flag: without it the JVM enforces its
// own heap cap and exits 1, which the use cases report as a runtime error. The
// quotes are load-bearing too — runCmd is interpolated into an sh -c string, so
// unquoted the shell splits the flag into three arguments and it is ignored.
func TestJudgeConfig_JavaSignalsOutOfMemoryWithTheExitCodeEveryoneElseUses(t *testing.T) {
	cfg := decodeRealConfig(t)

	const want = `'-XX:OnOutOfMemoryError=kill -9 %p'`
	runCmd := cfg.Judge.Languages["java17"].RunCmd
	if !strings.Contains(runCmd, want) {
		t.Errorf("java17 runCmd is %q,\nit must carry %s or a Java MLE reports RUNTIME_ERROR", runCmd, want)
	}
}

// The light pool runs checkers and validators, and a Java one that exceeds its
// container has to exit 137 like every other language. Without this flag it
// exits 1, which the adapter reads as the checker rejecting the contestant's
// output — a silent wrong answer for a correct solution.
func TestJudgeConfig_JavaArtifactsSignalOutOfMemoryTheSameWay(t *testing.T) {
	cfg := decodeRealConfig(t)

	const want = `'-XX:OnOutOfMemoryError=kill -9 %p'`
	artifactRun := cfg.Judge.Languages["java17"].ArtifactRun
	if !strings.Contains(artifactRun, want) {
		t.Errorf("java17 artifactRun is %q,\nit must carry %s or a killed checker looks like a rejection", artifactRun, want)
	}
}

// The JVM applies MinRAMPercentage instead of MaxRAMPercentage when the
// container is under ~250 MB, so setting only the max leaves a Java solution
// with 47% of a small container against the 99% C++ gets. Both together hold
// 85-87% across the whole range. Dropping either one degrades silently: the
// solution just gets less memory than the problem promised.
func TestJudgeConfig_JavaGetsItsShareOfEveryContainerSize(t *testing.T) {
	cfg := decodeRealConfig(t)

	runCmd := cfg.Judge.Languages["java17"].RunCmd
	for _, want := range []string{"-XX:MaxRAMPercentage=90", "-XX:MinRAMPercentage=90"} {
		if !strings.Contains(runCmd, want) {
			t.Errorf("java17 runCmd is %q,\nit must carry %s", runCmd, want)
		}
	}

	// Those percentages hand at least 10% of the container to non-heap, so the
	// container has to exceed the problem's limit by at least as much or the
	// contestant pays for the JVM's reserve out of their own budget.
	if f := cfg.Judge.Languages["java17"].MemoryFactor; f < 1.1 {
		t.Errorf("java17 memoryFactor is %v, too low to buy back what the JVM reserves", f)
	}
}

// virtualObjectPath is the API's own config, and the worker never reads it at
// runtime: both files ship baked into the same image, so they can only drift in
// the repository, and a test catches that in CI before the image exists.
const virtualObjectPath = "../../config/virtual_object.json"

func decodeVirtualObject(t *testing.T) config.VirtualObject {
	t.Helper()
	data, err := os.ReadFile(virtualObjectPath)
	if err != nil {
		t.Fatalf("could not read %s: %v", virtualObjectPath, err)
	}
	var vo config.VirtualObject
	if err := json.Unmarshal(data, &vo); err != nil {
		t.Fatalf("could not decode %s: %v", virtualObjectPath, err)
	}
	return vo
}

// The invariant nobody was checking: a pool's ceiling has to cover the largest
// limit a problem may declare in that language, times whatever its runtime
// reserves for itself. Without it a problem declares 2048 MB, Claim caps the
// request, and the solution silently gets less than it was promised — which is
// exactly the python310 bug D14 had to fix by hand.
func TestJudgeConfig_EveryHeavyLanguageCanHonourTheLimitsThePlatformAllows(t *testing.T) {
	cfg := decodeRealConfig(t)
	vo := decodeVirtualObject(t)

	// The global is what bounds every language, not the per-language limits: a
	// problem's own memoryLimit is validated against the global and applies to
	// whatever language is submitted, and languageOverrides can only cap what an
	// override declares, never raise it past the global.
	maxMB := vo.MaxMemoryLimitGlobal

	for lang, sizing := range cfg.Judge.Pools[poolHeavy].Languages {
		factor := cfg.Judge.Languages[lang].MemoryFactor
		need := int64(float64(maxMB) * 1024 * 1024 * factor)
		if sizing.MemoryBytes < need {
			t.Errorf("pool %q, language %q: memoryBytes is %d, but a problem may declare %d MB, which needs %d with memoryFactor %v",
				poolHeavy, lang, sizing.MemoryBytes, maxMB, need, factor)
		}
	}
}

// The default checker is claimed by every submission to a problem without a
// custom one. Missing from either place, nothing fails until the first such
// submission — which in a contest is the worst possible moment.
func TestValidateJudgeConfig_RequiresTheDefaultChecker(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*judgeConfigFile)
		wantMsg string
	}{
		{
			// Caught by the older rule about pools sizing undeclared languages,
			// which fires first. Pinned anyway: what matters is that it is caught.
			"the language is not declared",
			func(c *judgeConfigFile) { delete(c.Judge.Languages, adapterjudge.CompareLanguage) },
			"sizes undeclared language",
		},
		{
			"the language is declared with nothing to run",
			func(c *judgeConfigFile) {
				c.Judge.Languages[adapterjudge.CompareLanguage] = judgeLanguageConfig{Image: "judge-runner:compare"}
			},
			"has no artifactRun",
		},
		{
			"the light pool does not size it",
			func(c *judgeConfigFile) {
				delete(c.Judge.Pools[poolLight].Languages, adapterjudge.CompareLanguage)
			},
			"does not size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			err := validateJudgeConfig(cfg)
			if err == nil {
				t.Fatal("expected an error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("expected the error to mention %q, got: %v", tt.wantMsg, err)
			}
		})
	}
}
