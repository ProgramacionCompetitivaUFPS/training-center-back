package main

import (
	"os"
	"regexp"
	"strconv"
	"testing"

	adapterjudge "github.com/training-judge-center/backend/internal/adapter/judge"
	"github.com/training-judge-center/backend/internal/config"
)

const workerManifestPath = "../../deploy/k8s/judge/worker.yaml"

// The node's allocatable ephemeral storage, measured on the e2-standard-8 the judge runs on.
const nodeEphemeralBytes = 17 << 30

func TestApplyJudgeConfigDefaults_FloorsTheTestDataSettings(t *testing.T) {
	var cfg judgeConfigFile

	applyJudgeConfigDefaults(&cfg)

	if cfg.Judge.TestDataDir != defaultTestDataDir {
		t.Errorf("testDataDir: got %q, want the default %q", cfg.Judge.TestDataDir, defaultTestDataDir)
	}
	if cfg.Judge.TestDataBudgetBytes != defaultTestDataBudgetBytes {
		t.Errorf("testDataBudgetBytes: got %d, want the default %d", cfg.Judge.TestDataBudgetBytes, defaultTestDataBudgetBytes)
	}
}

func TestApplyJudgeConfigDefaults_KeepsProvidedTestDataSettings(t *testing.T) {
	cfg := judgeConfigFile{Judge: judgeSection{TestDataDir: "/elsewhere", TestDataBudgetBytes: 7}}

	applyJudgeConfigDefaults(&cfg)

	if cfg.Judge.TestDataDir != "/elsewhere" || cfg.Judge.TestDataBudgetBytes != 7 {
		t.Errorf("a provided value was overwritten by a default: %+v", cfg.Judge)
	}
}

// A mistyped struct tag leaves the field at zero and the default would hide it.
func TestRealConfig_TestDataSettingsComeFromTheFile(t *testing.T) {
	j := decodeRealConfig(t).Judge

	if j.TestDataDir == "" {
		t.Error("testDataDir is empty: likely a struct tag that no longer matches the YAML")
	}
	if j.TestDataBudgetBytes <= 0 {
		t.Errorf("testDataBudgetBytes: got %d, want > 0", j.TestDataBudgetBytes)
	}
}

func TestTestDataConfig_TakesItsLimitsFromThePlatformLimits(t *testing.T) {
	vo := &config.VirtualObject{
		MaxZipUploadSizeMB:          150,
		MaxExtractedSizeMB:          512,
		MaxFileCountTestCase:        10000,
		MaxFileSizeTestCaseInputMB:  64,
		MaxFileSizeTestCaseAnswerMB: 8,
	}
	cfg := judgeConfigFile{Judge: judgeSection{TestDataDir: "/d", TestDataBudgetBytes: 99}}

	got := testDataConfig(cfg, vo)

	want := adapterjudge.TestDataLimits{
		MaxZipBytes:       150 << 20,
		MaxExtractedBytes: 512 << 20,
		MaxFiles:          10000,
		MaxInputBytes:     64 << 20,
		MaxAnswerBytes:    8 << 20,
	}
	if got.Limits != want {
		t.Errorf("limits: got %+v, want %+v", got.Limits, want)
	}
	if got.Dir != "/d" || got.BudgetBytes != 99 {
		t.Errorf("dir/budget: got %q / %d", got.Dir, got.BudgetBytes)
	}
}

func TestValidateTestDataBudget(t *testing.T) {
	cfg := adapterjudge.TestDataConfig{
		BudgetBytes: 1250, // five archives of 150 and the extraction of 500
		Limits:      adapterjudge.TestDataLimits{MaxZipBytes: 150, MaxExtractedBytes: 500},
	}

	if err := validateTestDataBudget(cfg, 5); err != nil {
		t.Errorf("a budget of exactly maxConcurrent archives plus one extraction must pass, got: %v", err)
	}
	if err := validateTestDataBudget(cfg, 6); err == nil {
		t.Error("a budget that cannot cover every archive and still extract one must be refused: a judging could wait for disk forever")
	}
}

// The numbers the worker ships with have to hold together: the budget covers the pod's
// concurrency, it fits the volume the manifest gives it, and the volumes fit the node.
func TestShippedConfig_TestDataBudgetFitsTheClusterAndTheLimits(t *testing.T) {
	cfg := decodeRealConfig(t)
	applyJudgeConfigDefaults(&cfg)
	vo := decodeVirtualObject(t)
	maxConcurrent := clusterDindCores - cfg.Judge.DockerDaemonReserveCores

	testData := testDataConfig(cfg, &vo)
	if err := validateTestDataBudget(testData, maxConcurrent); err != nil {
		t.Fatalf("the shipped test data budget does not cover the shipped limits: %v", err)
	}

	manifest, err := os.ReadFile(workerManifestPath)
	if err != nil {
		t.Fatalf("could not read %s: %v", workerManifestPath, err)
	}
	limit := emptyDirLimit(t, string(manifest), "test-data")
	if cfg.Judge.TestDataBudgetBytes >= limit {
		t.Errorf("the budget (%d) must stay under the test-data volume's sizeLimit (%d): the filesystem's own overhead needs room beyond it",
			cfg.Judge.TestDataBudgetBytes, limit)
	}
}

func TestWorkerManifest_VolumesFitTheNode(t *testing.T) {
	manifest, err := os.ReadFile(workerManifestPath)
	if err != nil {
		t.Fatalf("could not read %s: %v", workerManifestPath, err)
	}

	var total int64
	for _, name := range []string{"dind-storage", "judging", "test-data"} {
		total += emptyDirLimit(t, string(manifest), name)
	}
	if total >= nodeEphemeralBytes {
		t.Errorf("the emptyDir sizeLimits add up to %d bytes, which the node's %d allocatable cannot hold", total, int64(nodeEphemeralBytes))
	}
}

// emptyDirLimit reads the sizeLimit (in Gi or Mi) of a named emptyDir from the manifest.
func emptyDirLimit(t *testing.T, manifest, name string) int64 {
	t.Helper()
	re := regexp.MustCompile(`- name: ` + regexp.QuoteMeta(name) + `\n(?:\s*#[^\n]*\n)*\s*emptyDir:\n\s*sizeLimit: (\d+)(Gi|Mi)`)
	m := re.FindStringSubmatch(manifest)
	if m == nil {
		t.Fatalf("no emptyDir named %q with a sizeLimit in Gi or Mi in %s", name, workerManifestPath)
	}
	n, _ := strconv.ParseInt(m[1], 10, 64)
	if m[2] == "Mi" {
		return n << 20
	}
	return n << 30
}
