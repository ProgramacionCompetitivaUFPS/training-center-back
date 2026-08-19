package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const (
	realConfigPath = "../../config/judge_config.yaml"
	testLang       = "cpp20"
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
				CompileCmd:      "g++ -o /sandbox/Solution /sandbox/Solution.cpp",
				RunCmd:          "/sandbox/Solution",
				ArtifactSource:  "/sandbox/{name}.cpp",
				ArtifactCompile: "g++ -o /sandbox/{name} /sandbox/{name}.cpp",
				ArtifactPath:    "/sandbox/{name}",
				ArtifactRun:     "/sandbox/{name}",
			},
		},
		Pools: map[string]judgePoolConfig{
			poolSolutions: {Languages: map[string]judgePoolLanguageConfig{
				testLang: {CPU: "1", MemoryBytes: 1 << 30},
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
			name:    "no solutions pool",
			breaks:  func(c *judgeConfigFile) { delete(c.Judge.Pools, poolSolutions) },
			wantErr: `no "solutions" pool defined`,
		},
		{
			name:    "a pool sizes nothing",
			breaks:  func(c *judgeConfigFile) { c.Judge.Pools["checkers"] = judgePoolConfig{} },
			wantErr: `pool "checkers" sizes no languages`,
		},
		{
			name: "a pool sizes a language that is not declared",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolSolutions].Languages["rust"] = judgePoolLanguageConfig{CPU: "1", MemoryBytes: 1}
			},
			wantErr: `sizes undeclared language "rust"`,
		},
		{
			name: "a sized language has no cpu",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolSolutions].Languages[testLang] = judgePoolLanguageConfig{MemoryBytes: 1}
			},
			wantErr: "no cpu",
		},
		{
			name: "a sized language has no memory",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Pools[poolSolutions].Languages[testLang] = judgePoolLanguageConfig{CPU: "1"}
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
			name: "the solutions pool does not size a language that runs solutions",
			breaks: func(c *judgeConfigFile) {
				c.Judge.Languages["python310"] = judgeLanguageConfig{
					Image:           "judge-runner:python310",
					Extension:       "py",
					RunCmd:          "python3 /sandbox/Solution.py",
					ArtifactSource:  "/sandbox/{name}.py",
					ArtifactCompile: "python3 -m py_compile /sandbox/{name}.py",
					ArtifactPath:    "/sandbox/{name}.py",
					ArtifactRun:     "python3 /sandbox/{name}.py",
				}
			},
			wantErr: `language "python310" runs solutions but pool "solutions" does not size it`,
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
// only runs a fixed binary, from needing a solutions pool entry.
func TestValidateJudgeConfig_ExemptsLanguagesWithoutRunCmd(t *testing.T) {
	cfg := validConfig()
	cfg.Judge.Languages["compare"] = judgeLanguageConfig{Image: "judge-runner:compare"}

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
