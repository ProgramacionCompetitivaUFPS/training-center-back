# Pending Architectural Implementations

This document tracks cross-cutting infrastructure capabilities that are deliberately deferred from individual feature implementations. Each item describes what it is, **why it exists**, and **which features depend on it**.

---

## 1. JWT Token Blacklist (Session Invalidation)

### What it is
A mechanism to invalidate JWT tokens before their natural expiration. Since JWTs are stateless and self-contained, the server cannot "revoke" them by default — it can only let them expire. A blacklist stores token identifiers (JTI claim or full token) that must be rejected even if the signature and expiry are valid.

### Why it's needed
The current auth middleware only validates the token signature and expiry. Any issued token is valid until it expires, with no way to force logout.

### Implementation options
- **In-memory store** (simplest): a Go `sync.Map` keyed by JTI. Lost on server restart. Only viable for single-instance deployments.
- **Redis** (recommended): low-latency, TTL-aware, survives restarts, scales horizontally.
- **Database table** (fallback): `token_blacklist(jti, expires_at)` with a cleanup job.

### Required interface (Port)
```go
// internal/domain/user/token_blacklist.go
type TokenBlacklist interface {
    Revoke(ctx context.Context, jti string, expiresAt time.Time) error
    IsRevoked(ctx context.Context, jti string) (bool, error)
}
```

### Features that need this
| Feature | Trigger |
|---------|---------|
| Update Password (`PUT /users/password`) | Successful password change → invalidate all sessions |
| Recover Password (`POST /password/reset`) | Successful password reset → invalidate all sessions |
| Self Deactivate User (`POST /users/deactivation/*`) | Account deactivated → invalidate current session |

### What needs to change
- **Auth middleware**: check blacklist on every request before accepting a token.
- **JTI claim**: add a `jti` (UUID) to the JWT payload in `LoginUseCase` so tokens can be individually targeted.
- **Platform adapter**: implement `TokenBlacklist` interface (e.g., `platform/redis/token_blacklist.go`).

---

## 2. Email Notification Service

### What it is
An outbound email delivery integration. The application needs to send transactional emails: security alerts, verification codes, and confirmation messages.

### Why it's needed
Several flows are security-critical and require out-of-band confirmation via email. Without this, those flows are incomplete.

### Implementation options
- **SMTP** (simplest): Go's `net/smtp` or `gopkg.in/gomail.v2`. Works with any SMTP provider (Gmail, Mailgun, AWS SES).
- **HTTP provider** (recommended): SendGrid, Mailgun, or AWS SES via their Go SDKs. Provides delivery tracking, retries, and template management.

### Required interface (Port)
```go
// internal/domain/notification/email_sender.go
type EmailSender interface {
    Send(ctx context.Context, to, subject, body string) error
}
```

### Features that need this
| Feature | Emails sent |
|---------|------------|
| Update Password (`PUT /users/password`) | Security alert to user's email |
| Update Email User (`POST /users/email-change/*`) | Verification code to **new** email + security alert to **old** email |
| Recover Password (`POST /password/forgot`) | Recovery code to registered email |
| Self Deactivate User (`POST /users/deactivation/*`) | Deactivation confirmation email |

### What needs to change
- **Platform adapter**: implement `EmailSender` (e.g., `platform/email/smtp_sender.go`).
- **Config**: add SMTP/API credentials to `config/config.go` and environment variables.
- **Wiring**: inject `EmailSender` into the relevant use cases via `main.go`.

---

## 3. Rate Limiter

### What it is
A per-user (or per-email) counter that tracks how many times a sensitive action has been attempted within a time window. If the threshold is exceeded, the action is rejected with HTTP 429.

### Why it's needed
Without rate limiting, a malicious actor can brute-force passwords, spam verification code endpoints, or abuse public recovery endpoints.

### Implementation options
- **In-memory** (`sync.Map` + goroutine cleanup): simple, no external dependency, but resets on restart and doesn't scale horizontally.
- **Redis** (recommended): atomic `INCR` + `EXPIRE`. Accurate across restarts and instances.
- **Database table** (fallback): `rate_limit_attempts(key, count, window_start)` with cleanup job.

### Required interface (Port)
```go
// internal/domain/ratelimit/rate_limiter.go
type RateLimiter interface {
    // Allow checks if the action is permitted and increments the counter.
    // Returns (true, nil) if allowed. Returns (false, nil) if limit exceeded.
    Allow(ctx context.Context, key string, maxAttempts int, window time.Duration) (bool, error)
    // Reset clears the counter for the given key (e.g. after a successful action).
    Reset(ctx context.Context, key string) error
}
```

### Features that need this
| Feature | Key | Limit |
|---------|-----|-------|
| Update Password (`PUT /users/password`) | `pw-update:{userID}` | 5 failed attempts / 1 hour |
| Recover Password (`POST /password/forgot`) | `pw-recover:{email}` | 5 requests / 1 hour |

### What needs to change
- **Platform adapter**: implement `RateLimiter` (e.g., `platform/ratelimit/memory_rate_limiter.go`).
- **Use cases**: inject `RateLimiter` and call `Allow` before processing; call `Reset` on success.
- **`apperror` package**: add `NewTooManyRequests(retryAfter int)` constructor for HTTP 429.

---

## 4. Verification Code Store

### What it is
A temporary, keyed store for short-lived verification codes (6-digit numeric). These codes are generated by the server, stored with an expiry, and later validated by the user.

### Why it's needed
Two features generate out-of-band time-limited codes sent via email. The server must persist the code between the "request" and "confirm" steps.

### Implementation options
- **Database table** (recommended for this project): `email_change_requests` / `password_recovery_requests` tables with `status`, `expires_at`, and `code` fields. Matches the entities defined in the specs.
- **Redis** with TTL: simpler, but requires an extra dependency and loses audit trail.

### Features that need this
| Feature | Entity | Expiry | Table / struct |
|---------|--------|--------|----------------|
| Update Email User (`POST /users/email-change/*`) | `EmailChangeRequest` | 15 min | `email_change_requests` |
| Recover Password (`POST /password/forgot` + reset) | `PasswordRecoveryRequest` | 15 min | `password_recovery_requests` |

### What needs to change
- **Domain**: define `EmailChangeRequest` and `PasswordRecoveryRequest` entities and their repository ports.
- **Platform**: implement repository adapters + DB migrations for the new tables.
- **Application**: use cases for request + confirm steps.

---

## 5. `apperror` — Missing HTTP 429 Constructor

### What it is
The `pkg/apperror` package currently has no constructor for HTTP 429 Too Many Requests. This is needed to return consistent error responses for rate-limited actions.

### What needs to change
Add to `pkg/apperror/errors.go`:

```go
func NewTooManyRequests(retryAfter int) *AppError {
    return &AppError{
        Code:       "RATE_LIMIT_EXCEEDED",
        Message:    "Too many requests. Please try again later",
        StatusCode: http.StatusTooManyRequests,
        // RetryAfter is not in AppError yet — see below
    }
}
```

The `AppError` struct itself may also need a `RetryAfter int` field to serialize the `retryAfter` value the spec requires in the 429 response body.

---

## Summary Table

| Item | Complexity | Needed by |
|------|-----------|-----------|
| JWT Token Blacklist | Medium | Update Password, Recover Password, Deactivate User |
| Email Notification Service | Medium | Update Password, Update Email, Recover Password, Deactivate User |
| Rate Limiter | Medium | Update Password, Recover Password |
| Verification Code Store | Medium | Update Email, Recover Password |
| ~~`apperror` HTTP 429~~ | ~~Low~~ | ~~Update Password, Recover Password~~ |

> **Recommended implementation order**: Start with `apperror` HTTP 429 (trivial), then Email Service (unblocks the most features), then Verification Code Store + Rate Limiter together (they're always co-implemented), and finally the Token Blacklist (requires a Redis/cache dependency decision).
