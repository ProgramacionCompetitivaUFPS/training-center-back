package user

import "testing"

func TestNewPassword_Valid(t *testing.T) {
	pw, err := NewPassword("Secret1!")
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
	pw, err := NewPassword(raw)
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

func TestNewPassword_TooShort(t *testing.T) {
	_, err := NewPassword("Sh1!")
	if err == nil {
		t.Fatal("expected error for short password, got nil")
	}
}

func TestNewPassword_TooLong(t *testing.T) {
	long := make([]byte, 73)
	for i := range long {
		long[i] = 'a'
	}
	long[0] = 'A'
	long[1] = '1'
	long[2] = '!'
	_, err := NewPassword(string(long))
	if err == nil {
		t.Fatal("expected error for 73-char password, got nil")
	}
}

func TestNewPassword_MissingUppercase(t *testing.T) {
	_, err := NewPassword("secret1!")
	if err == nil {
		t.Fatal("expected error for missing uppercase, got nil")
	}
}

func TestNewPassword_MissingDigit(t *testing.T) {
	_, err := NewPassword("Secret!!")
	if err == nil {
		t.Fatal("expected error for missing digit, got nil")
	}
}

func TestNewPassword_MissingSpecialChar(t *testing.T) {
	_, err := NewPassword("Secret12")
	if err == nil {
		t.Fatal("expected error for missing special char, got nil")
	}
}

func TestNewPasswordFromHash_Valid(t *testing.T) {
	hash := "$2a$10$somefakehashvalue"
	pw, err := NewPasswordFromHash(hash)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if pw.Hash() != hash {
		t.Errorf("expected hash %q, got %q", hash, pw.Hash())
	}
}

func TestNewPasswordFromHash_InvalidFormat(t *testing.T) {
	_, err := NewPasswordFromHash("Secret1!")
	if err == nil {
		t.Fatal("expected error for raw password passed as hash, got nil")
	}
}
