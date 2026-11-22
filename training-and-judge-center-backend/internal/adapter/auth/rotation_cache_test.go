package auth_test

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/training-judge-center/backend/internal/adapter/auth"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testEncryptionKey = bytes.Repeat([]byte("a"), 32)
var otherEncryptionKey = bytes.Repeat([]byte("b"), 32)

func newTestCache(t *testing.T) (*auth.RedisRotationCache, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache, err := auth.NewRedisRotationCache(client, testEncryptionKey)
	if err != nil {
		t.Fatalf("unexpected error building cache: %v", err)
	}
	return cache, mr
}

func TestSave_Get_RoundTrips(t *testing.T) {
	// Arrange
	cache, _ := newTestCache(t)
	ctx := context.Background()
	output := appuser.RefreshOutput{
		Token:            "access-token",
		RefreshToken:     "refresh-secret",
		SessionExpiresAt: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
	}

	// Act
	if err := cache.Save(ctx, "old-hash", output, time.Minute); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}
	got, err := cache.Get(ctx, "old-hash")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error on Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected a cached output, got nil")
	}
	if got.Token != output.Token || got.RefreshToken != output.RefreshToken || !got.SessionExpiresAt.Equal(output.SessionExpiresAt) {
		t.Errorf("expected %+v, got %+v", output, *got)
	}
}

func TestGet_UnknownHash_ReturnsNilNil(t *testing.T) {
	// Arrange
	cache, _ := newTestCache(t)
	ctx := context.Background()

	// Act
	got, err := cache.Get(ctx, "never-saved")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil for an unknown hash, got %+v", got)
	}
}

func TestSave_SetsExactTTL(t *testing.T) {
	// Arrange
	cache, mr := newTestCache(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	// Act
	if err := cache.Save(ctx, "old-hash", appuser.RefreshOutput{}, ttl); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Assert
	got := mr.TTL("rotation:old-hash")
	if got != ttl {
		t.Errorf("expected TTL %v, got %v", ttl, got)
	}
}

func TestGet_AfterTTLExpires_ReturnsNilNil(t *testing.T) {
	// Arrange
	cache, mr := newTestCache(t)
	ctx := context.Background()
	ttl := 10 * time.Second

	if err := cache.Save(ctx, "old-hash", appuser.RefreshOutput{}, ttl); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	// Act — advance simulated time past the TTL
	mr.FastForward(ttl + time.Second)
	got, err := cache.Get(ctx, "old-hash")

	// Assert
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after TTL expiry, got %+v", got)
	}
}

func TestSave_RedisUnreachable_ReturnsError(t *testing.T) {
	// Arrange
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache, err := auth.NewRedisRotationCache(client, testEncryptionKey)
	if err != nil {
		t.Fatalf("unexpected error building cache: %v", err)
	}
	ctx := context.Background()
	mr.Close()

	// Act
	err = cache.Save(ctx, "old-hash", appuser.RefreshOutput{}, time.Minute)

	// Assert
	if err == nil {
		t.Fatal("expected error when Redis is unreachable, got nil")
	}
}

func TestGet_RedisUnreachable_ReturnsError(t *testing.T) {
	// Arrange
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	cache, err := auth.NewRedisRotationCache(client, testEncryptionKey)
	if err != nil {
		t.Fatalf("unexpected error building cache: %v", err)
	}
	ctx := context.Background()
	mr.Close()

	// Act
	_, err = cache.Get(ctx, "old-hash")

	// Assert
	if err == nil {
		t.Fatal("expected error when Redis is unreachable, got nil")
	}
}

func TestNewRedisRotationCache_RejectsWrongKeyLength(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	if _, err := auth.NewRedisRotationCache(client, []byte("too-short")); err == nil {
		t.Fatal("expected error for a non-32-byte key, got nil")
	}
}

func TestSave_RawRedisValue_IsNotReadableJSON(t *testing.T) {
	cache, mr := newTestCache(t)
	ctx := context.Background()
	output := appuser.RefreshOutput{
		Token:            "super-secret-access-token",
		RefreshToken:     "super-secret-refresh-token",
		SessionExpiresAt: time.Date(2026, 8, 8, 12, 0, 0, 0, time.UTC),
	}

	if err := cache.Save(ctx, "old-hash", output, time.Minute); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	raw, err := mr.Get("rotation:old-hash")
	if err != nil {
		t.Fatalf("unexpected error reading raw Redis value: %v", err)
	}
	if strings.Contains(raw, output.Token) || strings.Contains(raw, output.RefreshToken) {
		t.Error("expected the raw Redis value to not contain the plaintext tokens")
	}
	if json.Valid([]byte(raw)) {
		t.Error("expected the raw Redis value to not be readable JSON")
	}
}

func TestGet_WrongKey_FailsToDecrypt(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	writer, err := auth.NewRedisRotationCache(client, testEncryptionKey)
	if err != nil {
		t.Fatalf("unexpected error building cache: %v", err)
	}
	reader, err := auth.NewRedisRotationCache(client, otherEncryptionKey)
	if err != nil {
		t.Fatalf("unexpected error building cache: %v", err)
	}
	ctx := context.Background()

	if err := writer.Save(ctx, "old-hash", appuser.RefreshOutput{Token: "access"}, time.Minute); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	if _, err := reader.Get(ctx, "old-hash"); err == nil {
		t.Error("expected Get with the wrong key to fail to decrypt, got nil error")
	}
}

func TestSave_AAD_PreventsCiphertextSwapAcrossKeys(t *testing.T) {
	cache, mr := newTestCache(t)
	ctx := context.Background()

	if err := cache.Save(ctx, "hash-a", appuser.RefreshOutput{Token: "access-a"}, time.Minute); err != nil {
		t.Fatalf("unexpected error on Save: %v", err)
	}

	raw, err := mr.Get("rotation:hash-a")
	if err != nil {
		t.Fatalf("unexpected error reading raw Redis value: %v", err)
	}
	if err := mr.Set("rotation:hash-b", raw); err != nil {
		t.Fatalf("unexpected error copying raw Redis value: %v", err)
	}

	if _, err := cache.Get(ctx, "hash-b"); err == nil {
		t.Error("expected Get to fail when the ciphertext was swapped to a different Redis key")
	}
}

func assertInternal(t *testing.T, err error) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Kind != apperror.KindInternal {
		t.Errorf("expected an internal apperror, got %v", err)
	}
}

func TestGet_PayloadShorterThanTheNonce_ReturnsInternal(t *testing.T) {
	cache, mr := newTestCache(t)
	if err := mr.Set("rotation:old-hash", "short"); err != nil {
		t.Fatal(err)
	}

	got, err := cache.Get(context.Background(), "old-hash")

	assertInternal(t, err)
	if got != nil {
		t.Errorf("expected no output for a truncated payload, got %+v", *got)
	}
}

func TestGet_AuthenticPayloadThatIsNotJSON_ReturnsInternal(t *testing.T) {
	cache, mr := newTestCache(t)
	block, err := aes.NewCipher(testEncryptionKey)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatal(err)
	}
	nonce := bytes.Repeat([]byte{1}, gcm.NonceSize())
	sealed := gcm.Seal(nonce, nonce, []byte("not json"), []byte("old-hash"))
	if err := mr.Set("rotation:old-hash", string(sealed)); err != nil {
		t.Fatal(err)
	}

	got, err := cache.Get(context.Background(), "old-hash")

	assertInternal(t, err)
	if got != nil {
		t.Errorf("expected no output for an undecodable payload, got %+v", *got)
	}
}
