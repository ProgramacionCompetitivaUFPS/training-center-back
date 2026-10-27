package user

import (
	"context"
	"errors"
	"testing"
	"time"

	domain "github.com/training-judge-center/backend/internal/domain/user"
)

// close to the real time.Now() the use case reads internally (D10)
var logoutTestNow = time.Now()

const (
	testLogoutFamily = "logout-family-id"
	testLogoutHash   = "logout-hash"
)

func activeLogoutTokenFixture() *domain.RefreshToken {
	return domain.RestoreRefreshToken(
		"logout-token-id", testRefreshUserID, testLogoutFamily, testLogoutHash,
		logoutTestNow, logoutTestNow.Add(domain.DefaultSessionCeiling),
		nil, nil, nil, nil,
	)
}

func revokedLogoutTokenFixture() *domain.RefreshToken {
	revokedAt := logoutTestNow
	return domain.RestoreRefreshToken(
		"logout-token-id", testRefreshUserID, testLogoutFamily, testLogoutHash,
		logoutTestNow, logoutTestNow.Add(domain.DefaultSessionCeiling),
		&revokedAt, nil, nil, nil,
	)
}

func TestLogout_Success_RevokesSessionThenFamily_InThatOrder(t *testing.T) {
	var calls []string
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			return activeLogoutTokenFixture(), nil
		},
		revokeByFamilyIDFn: func(_ context.Context, familyID string, _ time.Time) error {
			if familyID != testLogoutFamily {
				t.Errorf("RevokeByFamilyID: got familyID %q, want %q", familyID, testLogoutFamily)
			}
			calls = append(calls, "RevokeByFamilyID")
			return nil
		},
	}
	sessionInv := &mockSessionInvalidator{
		invalidateSessionFn: func(_ context.Context, sessionID string, _ time.Time) error {
			if sessionID != testLogoutFamily {
				t.Errorf("InvalidateSession: got sessionID %q, want %q", sessionID, testLogoutFamily)
			}
			calls = append(calls, "InvalidateSession")
			return nil
		},
	}
	uc := NewLogoutUseCase(refreshRepo, sessionInv, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "wrapped-value"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	want := []string{"InvalidateSession", "RevokeByFamilyID"}
	if len(calls) != len(want) || calls[0] != want[0] || calls[1] != want[1] {
		t.Errorf("call order: got %v, want %v", calls, want)
	}
}

func TestLogout_EmptyCookie_NoOp(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			t.Fatal("FindByTokenHash should not be called when RefreshToken is empty")
			return nil, nil
		},
	}
	uc := NewLogoutUseCase(refreshRepo, &mockSessionInvalidator{}, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: ""})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLogout_UnwrapFails_NoOp(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			t.Fatal("FindByTokenHash should not be called when Unwrap fails")
			return nil, nil
		},
	}
	codec := &mockRefreshTokenCodec{
		unwrapFn: func(_ string) (string, string, error) {
			return "", "", errors.New("invalid signature")
		},
	}
	uc := NewLogoutUseCase(refreshRepo, &mockSessionInvalidator{}, codec)

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "tampered"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLogout_TokenNotFound_NoOp(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			return nil, nil
		},
		revokeByFamilyIDFn: func(_ context.Context, _ string, _ time.Time) error {
			t.Fatal("RevokeByFamilyID should not be called when token is not found")
			return nil
		},
	}
	sessionInv := &mockSessionInvalidator{
		invalidateSessionFn: func(_ context.Context, _ string, _ time.Time) error {
			t.Fatal("InvalidateSession should not be called when token is not found")
			return nil
		},
	}
	uc := NewLogoutUseCase(refreshRepo, sessionInv, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "wrapped-value"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLogout_AlreadyRevoked_NoOp(t *testing.T) {
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			return revokedLogoutTokenFixture(), nil
		},
		revokeByFamilyIDFn: func(_ context.Context, _ string, _ time.Time) error {
			t.Fatal("RevokeByFamilyID should not be called when token is already revoked")
			return nil
		},
	}
	sessionInv := &mockSessionInvalidator{
		invalidateSessionFn: func(_ context.Context, _ string, _ time.Time) error {
			t.Fatal("InvalidateSession should not be called when token is already revoked")
			return nil
		},
	}
	uc := NewLogoutUseCase(refreshRepo, sessionInv, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "wrapped-value"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestLogout_FindByTokenHashError_Propagates(t *testing.T) {
	wantErr := errors.New("db unavailable")
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			return nil, wantErr
		},
	}
	uc := NewLogoutUseCase(refreshRepo, &mockSessionInvalidator{}, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "wrapped-value"})

	if !errors.Is(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
}

func TestLogout_InvalidateSessionError_Propagates(t *testing.T) {
	wantErr := errors.New("redis timeout")
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			return activeLogoutTokenFixture(), nil
		},
		revokeByFamilyIDFn: func(_ context.Context, _ string, _ time.Time) error {
			t.Fatal("RevokeByFamilyID should not be called when InvalidateSession fails")
			return nil
		},
	}
	sessionInv := &mockSessionInvalidator{
		invalidateSessionFn: func(_ context.Context, _ string, _ time.Time) error {
			return wantErr
		},
	}
	uc := NewLogoutUseCase(refreshRepo, sessionInv, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "wrapped-value"})

	if !errors.Is(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
}

func TestLogout_RevokeByFamilyIDError_Propagates(t *testing.T) {
	wantErr := errors.New("db unavailable")
	refreshRepo := &mockRefreshTokenRepository{
		findByTokenHashFn: func(_ context.Context, _ string) (*domain.RefreshToken, error) {
			return activeLogoutTokenFixture(), nil
		},
		revokeByFamilyIDFn: func(_ context.Context, _ string, _ time.Time) error {
			return wantErr
		},
	}
	uc := NewLogoutUseCase(refreshRepo, &mockSessionInvalidator{}, &mockRefreshTokenCodec{})

	err := uc.Execute(context.Background(), LogoutInput{RefreshToken: "wrapped-value"})

	if !errors.Is(err, wantErr) {
		t.Errorf("expected error %v, got %v", wantErr, err)
	}
}
