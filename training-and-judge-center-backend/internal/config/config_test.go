package config

import "testing"

func TestGetEnvAsInt_ValidPositiveValue_ReturnsIt(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "5")

	got := getEnvAsInt("TEST_INT_VAR", 1)

	if got != 5 {
		t.Errorf("expected 5, got %d", got)
	}
}

func TestGetEnvAsInt_Unset_ReturnsFallback(t *testing.T) {
	got := getEnvAsInt("TEST_INT_VAR_UNSET", 1)

	if got != 1 {
		t.Errorf("expected fallback 1, got %d", got)
	}
}

func TestGetEnvAsInt_NonNumeric_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "not-a-number")

	got := getEnvAsInt("TEST_INT_VAR", 1)

	if got != 1 {
		t.Errorf("expected fallback 1, got %d", got)
	}
}

func TestGetEnvAsInt_Zero_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "0")

	got := getEnvAsInt("TEST_INT_VAR", 1)

	if got != 1 {
		t.Errorf("expected fallback 1 for zero (would produce an immediately-expired JWT), got %d", got)
	}
}

func TestGetEnvAsInt_Negative_ReturnsFallback(t *testing.T) {
	t.Setenv("TEST_INT_VAR", "-5")

	got := getEnvAsInt("TEST_INT_VAR", 1)

	if got != 1 {
		t.Errorf("expected fallback 1 for a negative value (would produce an already-expired JWT), got %d", got)
	}
}
