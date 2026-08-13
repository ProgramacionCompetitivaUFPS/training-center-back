package queue

import (
	"encoding/json"
	"testing"
)

func TestMessageKind_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		kind messageKind
		want string
	}{
		{"submission", kindSubmission, `"SUBMISSION"`},
		{"problem validation", kindProblemValidation, `"PROBLEM_VALIDATION"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.kind)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if string(got) != tt.want {
				t.Errorf("MarshalJSON(): got %s, want %s", got, tt.want)
			}
		})
	}
}

func TestMessageKind_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    messageKind
		wantErr bool
	}{
		{"submission is valid", `"SUBMISSION"`, kindSubmission, false},
		{"problem validation is valid", `"PROBLEM_VALIDATION"`, kindProblemValidation, false},
		{"unknown kind returns error", `"BANANA"`, messageKind{}, true},
		{"empty string returns error", `""`, messageKind{}, true},
		{"malformed JSON returns error", `not-json`, messageKind{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got messageKind
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("UnmarshalJSON(): got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMessageKind_RoundTrip(t *testing.T) {
	for _, k := range allMessageKinds {
		t.Run(k.String(), func(t *testing.T) {
			body, err := json.Marshal(k)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var got messageKind
			if err := json.Unmarshal(body, &got); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got != k {
				t.Errorf("round trip: got %v, want %v", got, k)
			}
		})
	}
}
