package user_test

import (
	"strings"
	"testing"

	"github.com/training-judge-center/backend/internal/domain/user"
)

func TestNewPassword_Valid(t *testing.T) {
	pw, err := user.NewPassword("Secret1!")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pw.Hash() == "" {
		t.Fatal("expected non-empty hash")
	}
	if pw.Hash() == "Secret1!" {
		t.Fatal("hash should not equal the raw password")
	}
}

func TestNewPassword_Compare(t *testing.T) {
	raw := "Secret1!"
	pw, err := user.NewPassword(raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !pw.Compare(raw) {
		t.Error("Compare should return true for the original password")
	}
	if pw.Compare("WrongPassword1!") {
		t.Error("Compare should return false for a different password")
	}
}

func TestNewPassword_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"too short", "Sh1!"},
		{"too long (73 bytes)", "A1!" + strings.Repeat("a", 70)},
		{"missing uppercase", "secret1!"},
		{"missing digit", "Secret!!"},
		{"missing special char", "Secret12"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := user.NewPassword(tt.input)
			if err == nil {
				t.Fatalf("input %q: expected error, got nil", tt.input)
			}
		})
	}
}

func TestRestorePassword(t *testing.T) {
	hash := "$2a$10$somefakehashvalue"
	pw := user.RestorePassword(hash)
	if pw.Hash() != hash {
		t.Errorf("expected hash %q, got %q", hash, pw.Hash())
	}
}

func TestNewPassword_ExactlyAtBcryptLimit(t *testing.T) {
	raw := "A1!" + strings.Repeat("a", 69)
	pw, err := user.NewPassword(raw)
	if err != nil {
		t.Fatalf("expected no error at 72 bytes, got %v", err)
	}
	if !pw.Compare(raw) {
		t.Error("Compare should return true at 72-byte boundary")
	}
}

func TestRestorePassword_RoundTrip(t *testing.T) {
	pw, err := user.NewPassword("Secret1!")
	if err != nil {
		t.Fatalf("unexpected error creating password: %v", err)
	}
	restored := user.RestorePassword(pw.Hash())
	if !restored.Compare("Secret1!") {
		t.Error("restored password should compare correctly against original")
	}
	if restored.Compare("Wrong1!") {
		t.Error("restored password should not match a different input")
	}
}

func TestPassword_CompareEmpty(t *testing.T) {
	pw, err := user.NewPassword("Secret1!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pw.Compare("") {
		t.Error("Compare should return false for empty string")
	}
}
