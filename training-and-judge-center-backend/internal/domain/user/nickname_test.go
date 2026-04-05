package user

import "testing"

func TestNewNickname_Valid(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple nickname", "john", "john"},
		{"uppercase normalized", "JohnDoe", "johndoe"},
		{"with spaces trimmed", "  john  ", "john"},
		{"min length 3", "abc", "abc"},
		{"max length 30", "abcdefghijklmnopqrstuvwxyzabcd", "abcdefghijklmnopqrstuvwxyzabcd"},
		{"with hyphen", "john-doe", "john-doe"},
		{"with underscore", "john_doe", "john_doe"},
		{"alphanumeric mix", "user42", "user42"},
		{"hyphen and underscore combined", "j_o-h_n", "j_o-h_n"},
		{"numeric only", "123", "123"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nick, err := NewNickname(tt.input)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if nick.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, nick.String())
			}
		})
	}
}

func TestNewNickname_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"only spaces", "   "},
		{"too short (2 chars)", "ab"},
		{"too long (31 chars)", "abcdefghijklmnopqrstuvwxyzabcde"},
		{"internal space", "john doe"},
		{"special char less-than", "nick<name"},
		{"special char greater-than", "nick>name"},
		{"special char double-quote", `nick"name`},
		{"special char single-quote", "nick'name"},
		{"unicode symbol", "niño"},
		{"at sign", "nick@name"},
		{"dot", "nick.name"},
		{"with slash", "ab/cd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewNickname(tt.input)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
		})
	}
}
