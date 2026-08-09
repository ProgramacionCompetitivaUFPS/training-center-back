package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var _ appuser.RotationCache = (*RedisRotationCache)(nil)

type RedisRotationCache struct {
	client *redis.Client
	gcm    cipher.AEAD
}

func NewRedisRotationCache(client *redis.Client, encryptionKey []byte) (*RedisRotationCache, error) {
	// aes.NewCipher accepts 16/24/32 bytes — without this check, a 16-byte key in the env
	// would silently produce AES-128 instead of failing startup.
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("rotation cache encryption key must be 32 bytes, got %d", len(encryptionKey))
	}
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &RedisRotationCache{client: client, gcm: gcm}, nil
}

func (c *RedisRotationCache) Save(ctx context.Context, oldTokenHash string, output appuser.RefreshOutput, ttl time.Duration) error {
	plaintext, err := json.Marshal(output)
	if err != nil {
		slog.ErrorContext(ctx, "failed to marshal cached refresh output", "error", err)
		return apperror.NewInternal()
	}
	nonce := make([]byte, c.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		slog.ErrorContext(ctx, "failed to generate nonce", "error", err)
		return apperror.NewInternal()
	}
	// AAD = oldTokenHash: binds the ciphertext to its Redis key. Without this, an attacker with
	// write access could copy the blob from rotation:A to rotation:B and it would still decrypt.
	ciphertext := c.gcm.Seal(nonce, nonce, plaintext, []byte(oldTokenHash))
	if err := c.client.Set(ctx, "rotation:"+oldTokenHash, ciphertext, ttl).Err(); err != nil {
		slog.ErrorContext(ctx, "failed to cache refresh rotation", "error", err)
		return apperror.NewInternal()
	}
	return nil
}

func (c *RedisRotationCache) Get(ctx context.Context, oldTokenHash string) (*appuser.RefreshOutput, error) {
	data, err := c.client.Get(ctx, "rotation:"+oldTokenHash).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		slog.ErrorContext(ctx, "failed to read cached refresh rotation", "error", err)
		return nil, apperror.NewInternal()
	}
	nonceSize := c.gcm.NonceSize()
	if len(data) < nonceSize {
		slog.ErrorContext(ctx, "cached rotation payload too short", "len", len(data))
		return nil, apperror.NewInternal()
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := c.gcm.Open(nil, nonce, ciphertext, []byte(oldTokenHash))
	if err != nil {
		slog.ErrorContext(ctx, "failed to decrypt cached rotation", "error", err)
		return nil, apperror.NewInternal()
	}
	var output appuser.RefreshOutput
	if err := json.Unmarshal(plaintext, &output); err != nil {
		slog.ErrorContext(ctx, "failed to unmarshal cached refresh output", "error", err)
		return nil, apperror.NewInternal()
	}
	return &output, nil
}
