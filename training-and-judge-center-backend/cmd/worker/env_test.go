package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const exitHelperEnv = "WORKER_EXIT_HELPER"

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string
		set   bool
		want  string
	}{
		{"returns the value when it is set", "from-env", true, "from-env"},
		{"falls back when it is empty", "", true, "fallback"},
		{"falls back when it is not set", "", false, "fallback"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("WORKER_TEST_GETENV", "placeholder")
			os.Unsetenv("WORKER_TEST_GETENV")
			if tt.set {
				t.Setenv("WORKER_TEST_GETENV", tt.value)
			}

			got := getEnv("WORKER_TEST_GETENV", "fallback")

			if got != tt.want {
				t.Errorf("getEnv = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGetRequiredEnv_ReturnsTheValueWhenSet(t *testing.T) {
	t.Setenv("WORKER_TEST_REQUIRED", "amqp://broker")

	got := getRequiredEnv("WORKER_TEST_REQUIRED")

	if got != "amqp://broker" {
		t.Errorf("getRequiredEnv = %q, want %q", got, "amqp://broker")
	}
}

func TestGetRequiredEnvInt64_ParsesIntegers(t *testing.T) {
	tests := []struct {
		raw  string
		want int64
	}{
		{"42", 42},
		{"-5", -5},
		{"3221225472", 3221225472},
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			t.Setenv("WORKER_TEST_INT", tt.raw)

			got := getRequiredEnvInt64("WORKER_TEST_INT")

			if got != tt.want {
				t.Errorf("getRequiredEnvInt64(%q) = %d, want %d", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLoadJudgeConfig_AppliesDefaultsToAbsentKeys(t *testing.T) {
	data, err := os.ReadFile(realConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	var kept []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "  dockerDaemonReserveBytes:") {
			kept = append(kept, line)
		}
	}
	path := filepath.Join(t.TempDir(), "judge_config.yaml")
	if err := os.WriteFile(path, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("JUDGE_CONFIG", path)

	cfg := loadJudgeConfig()

	if cfg.Judge.DockerDaemonReserveBytes != 536870912 {
		t.Errorf("DockerDaemonReserveBytes = %d, want 536870912 (the 512 MiB default)", cfg.Judge.DockerDaemonReserveBytes)
	}
	if cfg.Judge.StaleRunningAfterMinutes != 10 {
		t.Errorf("StaleRunningAfterMinutes = %d, want 10 (the value in the file)", cfg.Judge.StaleRunningAfterMinutes)
	}
}

func TestExitHelper(t *testing.T) {
	switch os.Getenv(exitHelperEnv) {
	case "":
		t.Skip("runs only as a subprocess of TestWorkerRefusesToStart")
	case "required-env":
		getRequiredEnv("WORKER_TEST_MISSING")
	case "required-int":
		getRequiredEnvInt64("WORKER_TEST_INT")
	case "load-config":
		loadJudgeConfig()
	}
}

func TestWorkerRefusesToStart(t *testing.T) {
	dir := t.TempDir()
	unknownKey := filepath.Join(dir, "unknown-key.yaml")
	if err := os.WriteFile(unknownKey, []byte("judge:\n  notAKey: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	noLanguages := filepath.Join(dir, "no-languages.yaml")
	if err := os.WriteFile(noLanguages, []byte("judge:\n  idleTimeoutMinutes: 10\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name       string
		helper     string
		env        []string
		wantStderr string
	}{
		{"a required variable is not set", "required-env", nil, "required env var is not set"},
		{"a required integer is not an integer", "required-int", []string{"WORKER_TEST_INT=abc"}, "env var must be an integer"},
		{"the judge config file does not exist", "load-config", []string{"JUDGE_CONFIG=" + filepath.Join(dir, "absent.yaml")}, "failed to read judge config"},
		{"the judge config has an unknown key", "load-config", []string{"JUDGE_CONFIG=" + unknownKey}, "failed to parse judge config"},
		{"the judge config defines no languages", "load-config", []string{"JUDGE_CONFIG=" + noLanguages}, "invalid judge config"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, output := runExitHelper(t, tt.helper, tt.env...)

			if code != 1 {
				t.Errorf("exit code = %d, want 1; output:\n%s", code, output)
			}
			if !strings.Contains(output, tt.wantStderr) {
				t.Errorf("output does not contain %q:\n%s", tt.wantStderr, output)
			}
		})
	}
}

func runExitHelper(t *testing.T, helper string, env ...string) (int, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestExitHelper$")
	cmd.Env = append(os.Environ(), exitHelperEnv+"="+helper)
	cmd.Env = append(cmd.Env, env...)

	out, err := cmd.CombinedOutput()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), string(out)
	}
	if err != nil {
		t.Fatalf("run helper %s: %v", helper, err)
	}
	return 0, string(out)
}
