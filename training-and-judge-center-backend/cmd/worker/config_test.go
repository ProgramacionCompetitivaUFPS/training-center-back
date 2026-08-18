package main

import (
	"bytes"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

const realConfigPath = "../../config/judge_config.yaml"

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

// A mistyped struct tag does not fail the decode — the field just stays zero,
// and a zero reserve or timeout only shows up as a wrong limit much later. Each
// scalar is asserted to be populated so that drift is caught here.
func TestRealConfig_ScalarsArePopulated(t *testing.T) {
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

// Languages carry the image and the commands; pools carry the sizing. Both
// halves are needed to build a container, so neither can be missing.
func TestRealConfig_LanguagesAreComplete(t *testing.T) {
	langs := decodeRealConfig(t).Judge.Languages
	if len(langs) == 0 {
		t.Fatal("no languages declared")
	}
	for name, lc := range langs {
		if lc.Image == "" {
			t.Errorf("%s: no image", name)
		}
		if lc.RunCmd == "" {
			t.Errorf("%s: no runCmd", name)
		}
		if lc.Extension == "" {
			t.Errorf("%s: no extension", name)
		}
	}
}

// Mirrors the startup check: a pool sizing a language that has no image would
// only fail when that language is first claimed, mid-judging.
func TestRealConfig_PoolsOnlySizeDeclaredLanguages(t *testing.T) {
	j := decodeRealConfig(t).Judge

	if _, ok := j.Pools[poolSolutions]; !ok {
		t.Fatalf("no %q pool declared", poolSolutions)
	}
	for poolName, pool := range j.Pools {
		if len(pool.Languages) == 0 {
			t.Errorf("pool %s: no languages sized", poolName)
		}
		for lang, sizing := range pool.Languages {
			if _, ok := j.Languages[lang]; !ok {
				t.Errorf("pool %s sizes %q, which is not a declared language", poolName, lang)
			}
			if sizing.CPU == "" {
				t.Errorf("pool %s, %s: no cpu", poolName, lang)
			}
			if sizing.MemoryBytes <= 0 {
				t.Errorf("pool %s, %s: memoryBytes is %d", poolName, lang, sizing.MemoryBytes)
			}
		}
	}
}
