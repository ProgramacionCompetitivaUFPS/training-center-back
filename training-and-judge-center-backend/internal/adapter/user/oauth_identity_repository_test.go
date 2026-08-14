package user

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	domainUser "github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// ── Test fixtures ─────────────────────────────────────────────────────────────

const (
	testIdentityID     = "identity-aaaaaaaa-0001"
	testIdentityUserID = "user-bbbbbbbb-0001"
	testProviderUserID = "google-sub-123"
)

func newTestOAuthIdentity(t *testing.T) *domainUser.OAuthIdentity {
	t.Helper()
	identity, err := domainUser.NewOAuthIdentity(testIdentityID, testIdentityUserID, domainUser.OAuthProviderGoogle, testProviderUserID, testNow)
	if err != nil {
		t.Fatalf("newTestOAuthIdentity: %v", err)
	}
	return identity
}

func oauthIdentityScanFn(dest ...any) error {
	*(dest[0].(*string)) = testIdentityID
	*(dest[1].(*string)) = testIdentityUserID
	*(dest[2].(*time.Time)) = testNow
	return nil
}

// ── Save ──────────────────────────────────────────────────────────────────────

func TestOAuthIdentityRepository_Save_Success(t *testing.T) {
	var capturedArgs []interface{}
	repo := NewOAuthIdentityRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, args ...interface{}) (pgconn.CommandTag, error) {
			capturedArgs = args
			return pgconn.NewCommandTag("INSERT 0 1"), nil
		},
	})

	if err := repo.Save(context.Background(), newTestOAuthIdentity(t)); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	providerArg, ok := capturedArgs[2].(string)
	if !ok || providerArg != domainUser.OAuthProviderGoogle.String() {
		t.Errorf("expected provider arg %q as string, got %v", domainUser.OAuthProviderGoogle.String(), capturedArgs[2])
	}
}

func TestOAuthIdentityRepository_Save_DBError_ReturnsInternal(t *testing.T) {
	repo := NewOAuthIdentityRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, errors.New("db failure")
		},
	})

	err := repo.Save(context.Background(), newTestOAuthIdentity(t))
	assertAppErrorKind(t, err, apperror.KindInternal)
}

func TestOAuthIdentityRepository_Save_ProviderUserConflict(t *testing.T) {
	repo := NewOAuthIdentityRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, &pgconn.PgError{Code: infraPostgres.UniqueViolation, ConstraintName: "idx_oauth_identities_provider_user"}
		},
	})

	err := repo.Save(context.Background(), newTestOAuthIdentity(t))
	assertOAuthConflict(t, err, domainUser.ErrCodeOAuthIdentityConflict)
}

func TestOAuthIdentityRepository_Save_UserProviderConflict(t *testing.T) {
	repo := NewOAuthIdentityRepository(&mockQuerier{
		execFn: func(_ context.Context, _ string, _ ...interface{}) (pgconn.CommandTag, error) {
			return pgconn.CommandTag{}, &pgconn.PgError{Code: infraPostgres.UniqueViolation, ConstraintName: "idx_oauth_identities_user_provider"}
		},
	})

	err := repo.Save(context.Background(), newTestOAuthIdentity(t))
	assertOAuthConflict(t, err, domainUser.ErrCodeOAuthIdentityAlreadyLinked)
}

// ── FindByProvider ────────────────────────────────────────────────────────────

func TestOAuthIdentityRepository_FindByProvider_Success(t *testing.T) {
	repo := NewOAuthIdentityRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: oauthIdentityScanFn}
		},
	})

	identity, err := repo.FindByProvider(context.Background(), domainUser.OAuthProviderGoogle, testProviderUserID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if identity.UserID() != testIdentityUserID {
		t.Errorf("expected UserID %q, got %q", testIdentityUserID, identity.UserID())
	}
	if identity.Provider() != domainUser.OAuthProviderGoogle {
		t.Errorf("expected provider %q, got %q", domainUser.OAuthProviderGoogle, identity.Provider())
	}
}

func TestOAuthIdentityRepository_FindByProvider_ProviderRoundTrips(t *testing.T) {
	var capturedArgs []interface{}
	repo := NewOAuthIdentityRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, args ...interface{}) pgx.Row {
			capturedArgs = args
			return &mockRow{scanFn: oauthIdentityScanFn}
		},
	})

	if _, err := repo.FindByProvider(context.Background(), domainUser.OAuthProviderGoogle, testProviderUserID); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	providerArg, ok := capturedArgs[0].(string)
	if !ok || providerArg != domainUser.OAuthProviderGoogle.String() {
		t.Fatalf("expected query arg %q as string, got %v", domainUser.OAuthProviderGoogle.String(), capturedArgs[0])
	}
}

func TestOAuthIdentityRepository_FindByProvider_NotFound(t *testing.T) {
	repo := NewOAuthIdentityRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return pgx.ErrNoRows }}
		},
	})

	identity, err := repo.FindByProvider(context.Background(), domainUser.OAuthProviderGoogle, "does-not-exist")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if identity != nil {
		t.Errorf("expected nil identity, got %+v", identity)
	}
}

func TestOAuthIdentityRepository_FindByProvider_DBError_ReturnsInternal(t *testing.T) {
	repo := NewOAuthIdentityRepository(&mockQuerier{
		queryRowFn: func(_ context.Context, _ string, _ ...interface{}) pgx.Row {
			return &mockRow{scanFn: func(_ ...any) error { return errors.New("db failure") }}
		},
	})

	_, err := repo.FindByProvider(context.Background(), domainUser.OAuthProviderGoogle, testProviderUserID)
	assertAppErrorKind(t, err, apperror.KindInternal)
}

// ── test helpers ──────────────────────────────────────────────────────────────

func assertOAuthConflict(t *testing.T, err error, wantCode string) {
	t.Helper()
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *apperror.AppError, got %T: %v", err, err)
	}
	if appErr.Kind != apperror.KindConflict {
		t.Errorf("expected kind %q, got %q", apperror.KindConflict, appErr.Kind)
	}
	if appErr.Code != wantCode {
		t.Errorf("expected code %q, got %q", wantCode, appErr.Code)
	}
}
