package config

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func validVirtualObject() VirtualObject {
	return VirtualObject{
		MaxFileSizeTestCaseMB:       100,
		MaxFileSizeTestCaseInputMB:  64,
		MaxFileSizeTestCaseAnswerMB: 8,
	}
}

func TestValidateVirtualObject_Rules(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*VirtualObject)
		wantMsg string // empty means the config must be accepted
	}{
		{"a consistent config is accepted", func(*VirtualObject) {}, ""},
		{
			"the input cap is missing",
			func(vo *VirtualObject) { vo.MaxFileSizeTestCaseInputMB = 0 },
			"maxFileSizeTestCaseInputMB must be positive",
		},
		{
			"the answer cap is missing",
			func(vo *VirtualObject) { vo.MaxFileSizeTestCaseAnswerMB = 0 },
			"maxFileSizeTestCaseAnswerMB must be positive",
		},
		{
			"a single input may not exceed the whole package",
			func(vo *VirtualObject) { vo.MaxFileSizeTestCaseInputMB = 200 },
			"exceeds maxFileSizeTestCaseMB",
		},
		{
			"an answer may not exceed an input",
			func(vo *VirtualObject) { vo.MaxFileSizeTestCaseAnswerMB = 65 },
			"exceeds maxFileSizeTestCaseInputMB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vo := validVirtualObject()
			tt.mutate(&vo)

			err := validateVirtualObject(&vo)
			if tt.wantMsg == "" {
				if err != nil {
					t.Fatalf("expected the config to be accepted, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected the config to be rejected, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantMsg) {
				t.Errorf("error: got %q, want it to mention %q", err, tt.wantMsg)
			}
		})
	}
}

// Without this the API refuses to start instead, which is a much later and much
// worse place to find out that the shipped config contradicts itself.
func TestValidateVirtualObject_TheShippedConfigPasses(t *testing.T) {
	const path = "../../config/virtual_object.json"

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("could not read %s: %v", path, err)
	}
	var vo VirtualObject
	if err := json.Unmarshal(data, &vo); err != nil {
		t.Fatalf("could not decode %s: %v", path, err)
	}
	if err := validateVirtualObject(&vo); err != nil {
		t.Errorf("the shipped %s does not pass its own validation: %v", path, err)
	}
}
