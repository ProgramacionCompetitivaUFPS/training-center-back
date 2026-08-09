package user

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	domain "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// close to the real time.Now() the use case reads internally (D10) — not a fixed
// calendar date, which would drift out of sync with Execute's own clock over time.
var refreshTestNow = time.Now()

const (
	testRefreshTokenID = "token-id"
	testRefreshUserID  = "user-id"
	testRefreshFamily  = "family-id"
	testRefreshHash    = "old-hash"
)

func activeRefreshTokenFixture() *domain.RefreshToken {
	return domain.RestoreRefreshToken(
		testRefreshTokenID, testRefreshUserID, testRefreshFamily, testRefreshHash,
		refreshTestNow, refreshTestNow.Add(domain.DefaultSessionCeiling),
		nil, nil, nil, nil,
	)
}

func expiredRefreshTokenFixture() *domain.RefreshToken {
	return domain.RestoreRefreshToken(
		testRefreshTokenID, testRefreshUserID, testRefreshFamily, testRefreshHash,
		refreshTestNow.Add(-25*time.Hour), refreshTestNow.Add(-time.Hour),
		nil, nil, nil, nil,
	)
}

func revokedRefreshTokenFixture(revokedAt time.Time) *domain.RefreshToken {
	return domain.RestoreRefreshToken(
		testRefreshTokenID, testRefreshUserID, testRefreshFamily, testRefreshHash,
		refreshTestNow, refreshTestNow.Add(domain.DefaultSessionCeiling),
		&revokedAt, nil, nil, nil,
	)
}

func newRefreshDeps() (*mockRefreshTokenRepository, *mockUserRepository, *mockTokenService, *mockRotationCache) {
	userRepo := newNoConflictRepo()
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return newUserWithRole(testRefreshUserID, shared.RoleContestant, domain.StatusActive), nil
	}
	tokenSvc := &mockTokenService{
		generateTokenFn: func(_ context.Context, _ *domain.User) (string, error) { return "new-access-token", nil },
	}
	return &mockRefreshTokenRepository{}, userRepo, tokenSvc, &mockRotationCache{}
}

func TestRefresh_Success(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) {
		return true, nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	out, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Token != "new-access-token" {
		t.Errorf("expected access token, got %q", out.Token)
	}
	if out.RefreshToken == "" {
		t.Error("expected a non-empty refresh token")
	}
	want := activeRefreshTokenFixture().AbsoluteExpiresAt()
	if !out.SessionExpiresAt.Equal(want) {
		t.Errorf("expected sessionExpiresAt %v (inherited), got %v", want, out.SessionExpiresAt)
	}
}

func TestRefresh_EmptyToken(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: ""})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeUnauthorized {
		t.Errorf("expected code UNAUTHORIZED, got %q", appErr.Code)
	}
}

func TestRefresh_NotFound(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return nil, nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeUnauthorized {
		t.Errorf("expected code UNAUTHORIZED, got %q", appErr.Code)
	}
}

func TestRefresh_RateLimit_KeyedByUserID(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	var capturedKey string
	var capturedMax int
	var capturedWindow time.Duration
	rl := &mockRateLimiter{
		allowFn: func(_ context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
			capturedKey, capturedMax, capturedWindow = key, maxAttempts, window
			return true, nil
		},
	}
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) { return true, nil }
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, rl, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedKey != "rate_limit:refresh:user:"+testRefreshUserID {
		t.Errorf("expected key %q, got %q", "rate_limit:refresh:user:"+testRefreshUserID, capturedKey)
	}
	if capturedMax != 20 {
		t.Errorf("expected maxAttempts 20, got %d", capturedMax)
	}
	if capturedWindow != 10*time.Minute {
		t.Errorf("expected window 10m, got %v", capturedWindow)
	}
}

func TestRefresh_RateLimitExceeded_ByUser(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	rl := &mockRateLimiter{
		allowFn: func(_ context.Context, key string, _ int, _ time.Duration) (bool, error) {
			return !strings.HasPrefix(key, "rate_limit:refresh:user:"), nil
		},
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, rl, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeTooManyRequests {
		t.Errorf("expected code TOO_MANY_REQUESTS, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindTooManyRequests {
		t.Errorf("expected kind TOO_MANY_REQUESTS, got %s", appErr.Kind)
	}
}

func TestRefresh_AccountDeactivated(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	userRepo.findByIDFn = func(_ context.Context, _ string) (*domain.User, error) {
		return newUserWithRole(testRefreshUserID, shared.RoleContestant, domain.StatusDeactivated), nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != ErrCodeAccountDeactivated {
		t.Errorf("expected code ACCOUNT_DEACTIVATED, got %q", appErr.Code)
	}
	if appErr.Kind != apperror.KindForbidden {
		t.Errorf("expected kind FORBIDDEN, got %s", appErr.Kind)
	}
}

func TestRefresh_Expired(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return expiredRefreshTokenFixture(), nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeUnauthorized {
		t.Errorf("expected code UNAUTHORIZED, got %q", appErr.Code)
	}
}

func TestRefresh_AlreadyRevoked_OutsideGraceWindow_RevokesFamily(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	revoked := revokedRefreshTokenFixture(refreshTestNow.Add(-time.Minute)) // way past the 10s window
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return revoked, nil
	}
	var revokedFamilyID string
	refreshRepo.revokeByFamilyIDFn = func(_ context.Context, familyID string, _ time.Time) error {
		revokedFamilyID = familyID
		return nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Code != apperror.ErrCodeUnauthorized {
		t.Errorf("expected code UNAUTHORIZED, got %q", appErr.Code)
	}
	if revokedFamilyID != testRefreshFamily {
		t.Errorf("expected RevokeByFamilyID called with %q, got %q", testRefreshFamily, revokedFamilyID)
	}
}

func TestRefresh_AlreadyRevoked_CacheHit_SkipsGraceWindowCheck(t *testing.T) {
	// revokedAt is deliberately outside the grace window, so a cache hit only looks
	// like a pass if it truly short-circuits before the window is ever checked.
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	revoked := revokedRefreshTokenFixture(refreshTestNow.Add(-time.Hour)) // way past the 10s window
	findCalls := 0
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		findCalls++
		return revoked, nil
	}
	cachedOutput := &RefreshOutput{Token: "cached-access", RefreshToken: "cached-refresh", SessionExpiresAt: refreshTestNow}
	cache.getFn = func(_ context.Context, _ string) (*RefreshOutput, error) {
		return cachedOutput, nil
	}
	revokeCalled := false
	refreshRepo.revokeByFamilyIDFn = func(_ context.Context, _ string, _ time.Time) error {
		revokeCalled = true
		return nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	out, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Token != cachedOutput.Token || out.RefreshToken != cachedOutput.RefreshToken {
		t.Errorf("expected cached output %+v, got %+v", *cachedOutput, *out)
	}
	if revokeCalled {
		t.Error("expected RevokeByFamilyID NOT to be called when the cache has a hit")
	}
	if findCalls != 1 {
		t.Errorf("expected FindByTokenHash called once (only the initial lookup), got %d — the cache hit should short-circuit before the grace-window re-check", findCalls)
	}
}

func TestRefresh_AlreadyRevoked_WithinGraceWindow_WithoutCache(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	// revokedAt very close to "now" so WithinGraceWindow(now) is true regardless of tiny
	// scheduling delays in the test run.
	revokedAt := time.Now().Add(-1 * time.Second)
	revoked := domain.RestoreRefreshToken(
		testRefreshTokenID, testRefreshUserID, testRefreshFamily, testRefreshHash,
		revokedAt.Add(-time.Hour), revokedAt.Add(domain.DefaultSessionCeiling),
		&revokedAt, nil, nil, nil,
	)
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return revoked, nil
	}
	activeSuccessor := domain.RestoreRefreshToken(
		"successor-id", testRefreshUserID, testRefreshFamily, "successor-hash",
		revokedAt, revoked.AbsoluteExpiresAt(),
		nil, nil, nil, nil,
	)
	refreshRepo.findActiveByFamilyIDFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeSuccessor, nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	out, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if out.Token != "new-access-token" {
		t.Errorf("expected new access token, got %q", out.Token)
	}
	if out.RefreshToken != "" {
		t.Errorf("expected empty RefreshToken (no cookie to set), got %q", out.RefreshToken)
	}
	if !out.SessionExpiresAt.Equal(activeSuccessor.AbsoluteExpiresAt()) {
		t.Errorf("expected sessionExpiresAt %v, got %v", activeSuccessor.AbsoluteExpiresAt(), out.SessionExpiresAt)
	}
}

func TestRefresh_RotateReturnsFalse_TreatedAsAlreadyRevoked(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	calls := 0
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		calls++
		if calls == 1 {
			return activeRefreshTokenFixture(), nil // looked active on the first read
		}
		return revokedRefreshTokenFixture(refreshTestNow.Add(-time.Minute)), nil // lost the race
	}
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) {
		return false, nil
	}
	var revokeCalled bool
	refreshRepo.revokeByFamilyIDFn = func(_ context.Context, _ string, _ time.Time) error {
		revokeCalled = true
		return nil
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !revokeCalled {
		t.Error("expected the lost-race path to revoke the family, same as a directly-observed revoked token")
	}
}

func TestRefresh_RotateDBError_Propagates(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) {
		return false, apperror.NewInternal()
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Kind != apperror.KindInternal {
		t.Errorf("expected a real internal error to propagate, not be collapsed into rotated=false handling, got kind %s", appErr.Kind)
	}
}

func TestRefresh_TokenServiceError_Propagates(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) {
		return true, nil
	}
	tokenSvc.generateTokenFn = func(_ context.Context, _ *domain.User) (string, error) {
		return "", apperror.NewInternal()
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		t.Fatalf("expected *apperror.AppError, got %T", err)
	}
	if appErr.Kind != apperror.KindInternal {
		t.Errorf("expected kind INTERNAL, got %s", appErr.Kind)
	}
}

func TestRefresh_TokenServiceError_DoesNotRotate(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	rotateCalled := false
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) {
		rotateCalled = true
		return true, nil
	}
	tokenSvc.generateTokenFn = func(_ context.Context, _ *domain.User) (string, error) {
		return "", apperror.NewInternal()
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	_, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if rotateCalled {
		t.Error("expected Rotate NOT to be called when GenerateToken fails — the old refresh token must stay valid")
	}
}

func TestRefresh_RotationCacheSaveError_DoesNotFailRefresh(t *testing.T) {
	refreshRepo, userRepo, tokenSvc, cache := newRefreshDeps()
	refreshRepo.findByTokenHashFn = func(_ context.Context, _ string) (*domain.RefreshToken, error) {
		return activeRefreshTokenFixture(), nil
	}
	refreshRepo.rotateFn = func(_ context.Context, _ string, _ *domain.RefreshToken) (bool, error) {
		return true, nil
	}
	cache.saveFn = func(_ context.Context, _ string, _ RefreshOutput, _ time.Duration) error {
		return apperror.NewInternal()
	}
	uc := NewRefreshUseCase(refreshRepo, userRepo, tokenSvc, &mockRateLimiter{}, cache)

	out, err := uc.Execute(context.Background(), RefreshInput{RefreshToken: "raw-secret"})
	if err != nil {
		t.Fatalf("expected the refresh to succeed despite the cache write failing, got %v", err)
	}
	if out == nil || out.Token == "" {
		t.Error("expected a valid output despite the cache write failing")
	}
}
