package material

import "testing"

func TestToPrefixTsQuery(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"single word", "stack", "stack:*"},
		{"partial word", "progr", "progr:*"},
		{"two words", "stack overflow", "stack:* & overflow:*"},
		{"extra whitespace", "  stack   overflow  ", "stack:* & overflow:*"},
		{"punctuation stripped", "C++ node.js", "c:* & node:* & js:*"},
		{"only punctuation", "!!!", ""},
		{"empty", "", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toPrefixTsQuery(tc.input); got != tc.want {
				t.Errorf("toPrefixTsQuery(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
