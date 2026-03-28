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
