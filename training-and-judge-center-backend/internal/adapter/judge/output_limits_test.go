package judge

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/training-judge-center/backend/internal/config"
	"github.com/training-judge-center/backend/pkg/judgelimits"
)

// virtualObjectPath is the API's own config, and the judge never reads it at
// runtime: both files ship baked into the same image, so they can only drift in
// the repository, and a test catches that in CI before the image exists.
const virtualObjectPath = "../../../config/virtual_object.json"

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

// Three numbers in three places have to agree, and violating any of them is
// silent: the default checker exits non-zero and the adapter reads that as the
// contestant being wrong, so a whole problem turns into wrong answers.
func TestOutputLimits_AgreeAcrossTheBinaries(t *testing.T) {
	answerCap := decodeVirtualObject(t).MaxFileSizeTestCaseAnswerMB * 1024 * 1024

	// A correct answer is by definition something a correct solution can print,
	// and a solution that prints past the output limit has its output cut.
	if answerCap > maxOutputBytes {
		t.Errorf("the test case answer cap is %d, above the %d a solution may print",
			answerCap, maxOutputBytes)
	}

	// bufio rejects a token of exactly its buffer cap, so both files the default
	// checker reads have to stay STRICTLY under it. The contestant's file can
	// reach one block past the limit, which is where the kernel cuts it.
	if contestantCap := outputLimitBlocks * 512; contestantCap >= judgelimits.MaxTokenBytes {
		t.Errorf("a contestant output may reach %d, at or above the %d token cap in cmd/compare",
			contestantCap, judgelimits.MaxTokenBytes)
	}
	if answerCap >= judgelimits.MaxTokenBytes {
		t.Errorf("a jury answer may reach %d, at or above the %d token cap in cmd/compare",
			answerCap, judgelimits.MaxTokenBytes)
	}
}
