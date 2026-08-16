package user

import (
	"context"
	"time"

	"github.com/training-judge-center/backend/internal/application/shared"
	appuser "github.com/training-judge-center/backend/internal/application/user"
	domainuser "github.com/training-judge-center/backend/internal/domain/user"
)

// mockHandlerEmailChangeRepo implements domainuser.EmailChangeRepository for handler tests.
type mockHandlerEmailChangeRepo struct {
	findByCodeAndUserIDFn func(ctx context.Context, code string, userID string) (*domainuser.EmailChangeRequest, error)
}

func (m *mockHandlerEmailChangeRepo) Save(_ context.Context, _ *domainuser.EmailChangeRequest) error {
	return nil
}
func (m *mockHandlerEmailChangeRepo) FindByID(_ context.Context, _ string) (*domainuser.EmailChangeRequest, error) {
	return nil, nil
}
func (m *mockHandlerEmailChangeRepo) FindByCodeAndUserID(ctx context.Context, code string, userID string) (*domainuser.EmailChangeRequest, error) {
	if m.findByCodeAndUserIDFn != nil {
		return m.findByCodeAndUserIDFn(ctx, code, userID)
	}
	return nil, nil
}
func (m *mockHandlerEmailChangeRepo) InvalidatePendingByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockHandlerEmailChangeRepo) Update(_ context.Context, _ *domainuser.EmailChangeRequest) error {
	return nil
}

// mockHandlerPasswordRecoveryRepo implements domainuser.PasswordRecoveryRepository for handler tests.
type mockHandlerPasswordRecoveryRepo struct {
	findPendingByUserIDFn func(ctx context.Context, userID string) (*domainuser.PasswordRecoveryRequest, error)
	updateFn              func(ctx context.Context, req *domainuser.PasswordRecoveryRequest) error
}

func (m *mockHandlerPasswordRecoveryRepo) Save(_ context.Context, _ *domainuser.PasswordRecoveryRequest) error {
	return nil
}
func (m *mockHandlerPasswordRecoveryRepo) FindByID(_ context.Context, _ string) (*domainuser.PasswordRecoveryRequest, error) {
	return nil, nil
}
func (m *mockHandlerPasswordRecoveryRepo) FindPendingByUserID(ctx context.Context, userID string) (*domainuser.PasswordRecoveryRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockHandlerPasswordRecoveryRepo) Update(ctx context.Context, req *domainuser.PasswordRecoveryRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockHandlerPasswordRecoveryRepo) InvalidatePendingByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}

// mockHandlerRateLimiter implements ratelimit.RateLimiter for handler tests.
// By default it allows all requests.
type mockHandlerRateLimiter struct {
	allowFn func(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
}

func (m *mockHandlerRateLimiter) Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error) {
	if m.allowFn != nil {
		return m.allowFn(ctx, key, maxAttempts, window)
	}
	return true, nil
}

func (m *mockHandlerRateLimiter) Reset(_ context.Context, _ string) error { return nil }

type mockTokenService struct {
	validateFn func(token string) (*domainuser.TokenClaims, error)
}

func (m *mockTokenService) GenerateToken(_ context.Context, _ *domainuser.User, _ string) (string, error) {
	return "", nil
}
func (m *mockTokenService) ValidateToken(token string) (*domainuser.TokenClaims, error) {
	return m.validateFn(token)
}

type mockHandlerUserRepo struct {
	findByIDFn    func(ctx context.Context, id string) (*domainuser.User, error)
	findByEmailFn func(ctx context.Context, email domainuser.Email) (*domainuser.User, error)
	updateFn      func(ctx context.Context, u *domainuser.User) error
}

func (m *mockHandlerUserRepo) Save(_ context.Context, _ *domainuser.User) error { return nil }
func (m *mockHandlerUserRepo) Update(ctx context.Context, u *domainuser.User) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, u)
	}
	return nil
}
func (m *mockHandlerUserRepo) FindByEmail(ctx context.Context, email domainuser.Email) (*domainuser.User, error) {
	if m.findByEmailFn != nil {
		return m.findByEmailFn(ctx, email)
	}
	return nil, nil
}
func (m *mockHandlerUserRepo) FindByID(ctx context.Context, id string) (*domainuser.User, error) {
	if m.findByIDFn != nil {
		return m.findByIDFn(ctx, id)
	}
	return nil, nil
}
func (m *mockHandlerUserRepo) FindByNickname(_ context.Context, _ domainuser.Nickname) (*domainuser.User, error) {
	return nil, nil
}
func (m *mockHandlerUserRepo) FindAll(_ context.Context, _ domainuser.UserFilter) ([]*domainuser.User, int, error) {
	return nil, 0, nil
}

type mockHandlerDeactRepo struct {
	findPendingByUserIDFn func(ctx context.Context, userID string) (*domainuser.DeactivationRequest, error)
	updateFn              func(ctx context.Context, req *domainuser.DeactivationRequest) error
}

func (m *mockHandlerDeactRepo) Save(_ context.Context, _ *domainuser.DeactivationRequest) error {
	return nil
}
func (m *mockHandlerDeactRepo) FindPendingByUserID(ctx context.Context, userID string) (*domainuser.DeactivationRequest, error) {
	if m.findPendingByUserIDFn != nil {
		return m.findPendingByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockHandlerDeactRepo) FindByID(_ context.Context, _ string) (*domainuser.DeactivationRequest, error) {
	return nil, nil
}
func (m *mockHandlerDeactRepo) Update(ctx context.Context, req *domainuser.DeactivationRequest) error {
	if m.updateFn != nil {
		return m.updateFn(ctx, req)
	}
	return nil
}
func (m *mockHandlerDeactRepo) InvalidatePendingByUserID(_ context.Context, _ string, _ time.Time) error {
	return nil
}

type mockHandlerDeactAuditRepo struct{}

func (m *mockHandlerDeactAuditRepo) Save(_ context.Context, _ *domainuser.DeactivationAuditLog) error {
	return nil
}

type mockHandlerEmailSender struct{}

func (m *mockHandlerEmailSender) Send(_ context.Context, _ shared.EmailMessage) error {
	return nil
}

type mockHandlerSessionInvalidator struct {
	invalidateAllUserSessionsFn func(ctx context.Context, userID string, timestamp time.Time) error
}

func (m *mockHandlerSessionInvalidator) InvalidateAllUserSessions(ctx context.Context, userID string, timestamp time.Time) error {
	if m.invalidateAllUserSessionsFn != nil {
		return m.invalidateAllUserSessionsFn(ctx, userID, timestamp)
	}
	return nil
}
func (m *mockHandlerSessionInvalidator) IsAllUserSessionRevoked(_ context.Context, _ string, _ time.Time) (bool, error) {
	return false, nil
}
func (m *mockHandlerSessionInvalidator) InvalidateSession(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockHandlerSessionInvalidator) IsSessionInvalidated(_ context.Context, _ string) (bool, error) {
	return false, nil
}

type mockHandlerTxManager struct{}

func (m *mockHandlerTxManager) WithTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// mockHandlerRefreshTokenRepo implements domainuser.RefreshTokenRepository for handler tests.
type mockHandlerRefreshTokenRepo struct {
	revokeAllByUserIDFn func(ctx context.Context, userID string, now time.Time) error
}

func (m *mockHandlerRefreshTokenRepo) Save(_ context.Context, _ *domainuser.RefreshToken) error {
	return nil
}
func (m *mockHandlerRefreshTokenRepo) FindByTokenHash(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
	return nil, nil
}
func (m *mockHandlerRefreshTokenRepo) FindActiveByFamilyID(_ context.Context, _ string) (*domainuser.RefreshToken, error) {
	return nil, nil
}
func (m *mockHandlerRefreshTokenRepo) Rotate(_ context.Context, _ string, _ *domainuser.RefreshToken) (bool, error) {
	return false, nil
}
func (m *mockHandlerRefreshTokenRepo) RevokeByFamilyID(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (m *mockHandlerRefreshTokenRepo) RevokeAllByUserID(ctx context.Context, userID string, now time.Time) error {
	if m.revokeAllByUserIDFn != nil {
		return m.revokeAllByUserIDFn(ctx, userID, now)
	}
	return nil
}
func (m *mockHandlerRefreshTokenRepo) DeleteRevokedOrExpiredBefore(_ context.Context, _ time.Time) error {
	return nil
}

// ── dashboard provider mocks ──────────────────────────────────────────────────

type mockDashboardSubmissionProvider struct {
	getRecentSubmissionsFn func(ctx context.Context, userID string, limit int) ([]appuser.DashboardSubmission, error)
	getSubmissionDatesFn   func(ctx context.Context, userID string) ([]time.Time, error)
}

func (m *mockDashboardSubmissionProvider) GetRecentSubmissions(ctx context.Context, userID string, limit int) ([]appuser.DashboardSubmission, error) {
	if m.getRecentSubmissionsFn != nil {
		return m.getRecentSubmissionsFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockDashboardSubmissionProvider) GetSubmissionDates(ctx context.Context, userID string) ([]time.Time, error) {
	if m.getSubmissionDatesFn != nil {
		return m.getSubmissionDatesFn(ctx, userID)
	}
	return nil, nil
}

type mockDashboardContestProvider struct {
	getUpcomingContestsFn       func(ctx context.Context, userID string, limit int) ([]appuser.DashboardContest, error)
	getActiveContestsFn         func(ctx context.Context, userID string, limit int) ([]appuser.DashboardContest, error)
	getFinishedContestResultsFn func(ctx context.Context, userID string, limit int) ([]appuser.DashboardContestResult, error)
}

func (m *mockDashboardContestProvider) GetUpcomingContests(ctx context.Context, userID string, limit int) ([]appuser.DashboardContest, error) {
	if m.getUpcomingContestsFn != nil {
		return m.getUpcomingContestsFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockDashboardContestProvider) GetActiveContests(ctx context.Context, userID string, limit int) ([]appuser.DashboardContest, error) {
	if m.getActiveContestsFn != nil {
		return m.getActiveContestsFn(ctx, userID, limit)
	}
	return nil, nil
}

func (m *mockDashboardContestProvider) GetFinishedContestResults(ctx context.Context, userID string, limit int) ([]appuser.DashboardContestResult, error) {
	if m.getFinishedContestResultsFn != nil {
		return m.getFinishedContestResultsFn(ctx, userID, limit)
	}
	return nil, nil
}

type mockDashboardMaterialProvider struct {
	getRecentMaterialsCountFn func(ctx context.Context, userID string, windowDays int) (int, error)
}

func (m *mockDashboardMaterialProvider) GetRecentMaterialsCount(ctx context.Context, userID string, windowDays int) (int, error) {
	if m.getRecentMaterialsCountFn != nil {
		return m.getRecentMaterialsCountFn(ctx, userID, windowDays)
	}
	return 0, nil
}

type mockProblemsSolvedProvider struct {
	getProblemsSolvedFn func(ctx context.Context, userID string) (int, error)
}

func (m *mockProblemsSolvedProvider) GetProblemsSolved(ctx context.Context, userID string) (int, error) {
	if m.getProblemsSolvedFn != nil {
		return m.getProblemsSolvedFn(ctx, userID)
	}
	return 0, nil
}

// ── stats provider mocks ──────────────────────────────────────────────────────

type mockRankingProvider struct {
	getRankingFn func(ctx context.Context, userID string) (int, *int, int, error)
}

func (m *mockRankingProvider) GetRanking(ctx context.Context, userID string) (int, *int, int, error) {
	if m.getRankingFn != nil {
		return m.getRankingFn(ctx, userID)
	}
	return 0, nil, 0, nil
}

type mockSubmissionStatsProvider struct {
	getSubmissionCountsFn func(ctx context.Context, userID string) (int, int, error)
}

func (m *mockSubmissionStatsProvider) GetSubmissionCounts(ctx context.Context, userID string) (int, int, error) {
	if m.getSubmissionCountsFn != nil {
		return m.getSubmissionCountsFn(ctx, userID)
	}
	return 0, 0, nil
}

type mockContestParticipationProvider struct {
	getContestsParticipatedCountFn func(ctx context.Context, userID string) (int, error)
}

func (m *mockContestParticipationProvider) GetContestsParticipatedCount(ctx context.Context, userID string) (int, error) {
	if m.getContestsParticipatedCountFn != nil {
		return m.getContestsParticipatedCountFn(ctx, userID)
	}
	return 0, nil
}

type mockTopicStatsProvider struct {
	getTopicBreakdownFn func(ctx context.Context, userID string) ([]appuser.TopicStat, error)
}

func (m *mockTopicStatsProvider) GetTopicBreakdown(ctx context.Context, userID string) ([]appuser.TopicStat, error) {
	if m.getTopicBreakdownFn != nil {
		return m.getTopicBreakdownFn(ctx, userID)
	}
	return nil, nil
}

// mockHandlerOAuthIdentityRepo implements domainuser.OAuthIdentityRepository for handler tests.
type mockHandlerOAuthIdentityRepo struct {
	saveFn           func(ctx context.Context, identity *domainuser.OAuthIdentity) error
	findByProviderFn func(ctx context.Context, provider domainuser.OAuthProvider, providerUserID string) (*domainuser.OAuthIdentity, error)
	findByUserIDFn   func(ctx context.Context, userID string) (*domainuser.OAuthIdentity, error)
	deleteByUserIDFn func(ctx context.Context, userID string) error
}

func (m *mockHandlerOAuthIdentityRepo) Save(ctx context.Context, identity *domainuser.OAuthIdentity) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, identity)
	}
	return nil
}
func (m *mockHandlerOAuthIdentityRepo) FindByProvider(ctx context.Context, provider domainuser.OAuthProvider, providerUserID string) (*domainuser.OAuthIdentity, error) {
	if m.findByProviderFn != nil {
		return m.findByProviderFn(ctx, provider, providerUserID)
	}
	return nil, nil
}
func (m *mockHandlerOAuthIdentityRepo) FindByUserID(ctx context.Context, userID string) (*domainuser.OAuthIdentity, error) {
	if m.findByUserIDFn != nil {
		return m.findByUserIDFn(ctx, userID)
	}
	return nil, nil
}
func (m *mockHandlerOAuthIdentityRepo) DeleteByUserID(ctx context.Context, userID string) error {
	if m.deleteByUserIDFn != nil {
		return m.deleteByUserIDFn(ctx, userID)
	}
	return nil
}

// mockHandlerGoogleVerifier implements appuser.GoogleIDTokenVerifier for handler tests.
type mockHandlerGoogleVerifier struct {
	verifyFn func(ctx context.Context, idToken string) (*appuser.GoogleClaims, error)
}

func (m *mockHandlerGoogleVerifier) Verify(ctx context.Context, idToken string) (*appuser.GoogleClaims, error) {
	if m.verifyFn != nil {
		return m.verifyFn(ctx, idToken)
	}
	return &appuser.GoogleClaims{Sub: "google-sub-123", Email: "test@example.com", EmailVerified: true, Name: "Test User"}, nil
}
