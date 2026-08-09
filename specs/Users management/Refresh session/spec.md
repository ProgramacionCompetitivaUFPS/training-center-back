# Feature Specification: Refresh Session

**Created**: 2026-08-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Renew access token silently (Priority: P1)

As a logged-in user, I want my access token to renew automatically in the background when it expires, so that I don't get logged out every hour while actively using the platform.

**Why this priority**: Without this, the short access token lifetime (1 hour, see [Login](../Login/spec.md)) would force re-authentication every hour — unacceptable during a multi-hour contest or training session. This is the feature that makes a short access token lifetime viable at all.

**Independent Test**: Can be tested independently by logging in to obtain a refresh token cookie, then calling `POST /auth/refresh` directly and validating a new access token is returned and the refresh token cookie is rotated.

**Acceptance Scenarios**:

1. **Scenario**: Successful refresh
   - **Given** a user has a valid, non-expired refresh token cookie from a previous login
   - **When** the client calls `POST /auth/refresh`
   - **Then** the system returns a new access token (1-hour expiration)
   - **And** the system rotates the refresh token: the old one is invalidated and a new one is set in the cookie
   - **And** the new refresh token belongs to the same session (family) as the one it replaced
   - **And** `sessionExpiresAt` in the response is unchanged from the original login — refreshing never extends the absolute ceiling

2. **Scenario**: Missing or malformed refresh token
   - **Given** the request has no refresh token cookie, or the cookie value does not match any stored token
   - **When** `POST /auth/refresh` is called
   - **Then** the system rejects with 401 Unauthorized (`UNAUTHORIZED`)
   - **And** the response does not distinguish between "missing," "malformed," or "unknown" tokens

3. **Scenario**: Absolute session ceiling reached
   - **Given** the refresh token is valid but its session's absolute ceiling (1 day, or 30 days with `rememberSession`, see [Login](../Login/spec.md)) has passed
   - **When** `POST /auth/refresh` is called
   - **Then** the system rejects with 401 Unauthorized (`UNAUTHORIZED`)
   - **And** no new access or refresh token is issued — the user must log in again

4. **Scenario**: User account no longer active
   - **Given** the refresh token is otherwise valid, but the owning user's status is no longer `ACTIVE` (deactivated after the token was issued)
   - **When** `POST /auth/refresh` is called
   - **Then** the system rejects with 403 Forbidden (`ACCOUNT_DEACTIVATED`)
   - **And** no new access or refresh token is issued

5. **Scenario**: Rate limit exceeded
   - **Given** the presented token resolves to a known session, and that session's user has already made 20 refresh requests in the last 10 minutes
   - **When** another refresh request is submitted
   - **Then** the system rejects with 429 Too Many Requests (`RATE_LIMIT_EXCEEDED`)

---

### User Story 2 – Detect and contain a stolen refresh token (Priority: P1)

As the platform operator, I want a stolen refresh token to be usable at most once before the entire session is shut down, so that a leaked token doesn't grant an attacker indefinite access.

**Why this priority**: This is the core security property that justifies moving off a single long-lived JWT. Without reuse detection, a stolen refresh token would be as dangerous as the 24h JWT it replaces — usable by an attacker until it naturally expires.

**Independent Test**: Can be tested independently by performing a valid refresh (obtaining token B from token A), then replaying the *original* token A a second time and validating that the entire session — including token B, the one currently in use — is rejected.

**Acceptance Scenarios**:

1. **Scenario**: Reused (already-rotated) refresh token, outside the grace window
   - **Given** a refresh token was already exchanged for a new one more than 10 seconds ago
   - **When** the same (now-superseded) token is presented again to `POST /auth/refresh`
   - **Then** the system treats this as a reuse/theft signal
   - **And** revokes every active token in that session's family, including the current legitimate one
   - **And** rejects with 401 Unauthorized (`UNAUTHORIZED`)
   - **And** the legitimate device, on its next request, is also forced to log in again

2. **Scenario**: Near-simultaneous refresh from two tabs of the same browser (benign race, not theft)
   - **Given** two requests are made to `POST /auth/refresh` with the same refresh token within milliseconds of each other (e.g., two open tabs both reacting to an expired access token)
   - **When** the second request arrives within 10 seconds of the first one's rotation
   - **Then** the system does NOT treat this as reuse
   - **And** returns the same new access/refresh token pair that the first (winning) request already received
   - **And** neither tab is logged out

3. **Scenario**: Concurrent requests with the exact same still-valid token (race on the winning rotation itself)
   - **Given** two requests are made to `POST /auth/refresh` with the same, not-yet-rotated refresh token at the same time
   - **When** both are processed by the system
   - **Then** exactly one request performs the rotation (creates the new token)
   - **And** the other request is handled by the grace-window path (Scenario 2 above), not treated as an error

---

### Edge Cases

- Clock skew between server and the time recorded in `absolute_expires_at` (server-authoritative; no client-supplied timestamps are trusted).
- Refresh token cookie present but for a session that was already revoked by an unrelated event (password change, admin deactivation — see [Login](../Login/spec.md) Session Invalidation Context).
- A device refreshes so infrequently that its access token has been expired for a long time (e.g., laptop closed for days) — refresh still succeeds as long as the refresh token itself hasn't hit its absolute ceiling.
- Two different devices (different sessions/families) never interfere with each other's rotation or reuse detection.
- Rate limit reached while the user is actively working (should be effectively unreachable under normal use — see FR-010).
- Refresh called with a well-formed but entirely fabricated token value (not found in storage) — must be handled identically to Scenario 2 of Story 1, not leak whether *any* row was found.

## API Contract

### POST /auth/refresh

Exchange a valid refresh token (sent automatically via cookie) for a new access token, rotating the refresh token in the process.

> **Important**: Public endpoint in the sense that it requires no `Authorization` header — but it does require the `refresh_token` cookie set by [Login](../Login/spec.md). It is designed to be called silently by the frontend's HTTP client whenever a request fails with 401 due to access token expiration.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Cookie | string | Yes | Must include `refresh_token`, set automatically by the browser (not read or set manually by client code) |

**Request Body**: none.

**Responses**:

#### 200 OK
Refresh successful. Returns a new access token and session metadata; sets a new refresh token cookie.

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "sessionExpiresAt": "2026-08-08T14:30:00Z"
}
```

| Field | Type | Description |
|-------|------|-------------|
| token | string | New JWT access token (1-hour expiration), same claim shape as [Login](../Login/spec.md) |
| sessionExpiresAt | string (ISO 8601) | Absolute ceiling of this session, fixed at the original login — identical to the value returned by login, never extended by refreshing |

**Set-Cookie**:
| Cookie | Attributes | Description |
|--------|-----------|--------------|
| `refresh_token` | `HttpOnly; Secure; SameSite=Strict; Path=/auth; Max-Age=<seconds remaining until sessionExpiresAt>` | New opaque token replacing the one just used. `Max-Age` shrinks on every rotation as the absolute ceiling approaches — it is never reset to a full-length value. |

> **Note**: No user profile data is returned here — the frontend already has it from login and doesn't need it re-sent on every refresh.

#### 401 Unauthorized
Refresh token missing, unrecognized, expired past its absolute ceiling, or reuse of an already-rotated token detected. All four cases return the same generic response — the client's only correct reaction is to treat the session as over and prompt re-login.

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or expired session. Please log in again"
}
```

#### 403 Forbidden
The user's account is no longer active.

```json
{
  "error": "ACCOUNT_DEACTIVATED",
  "message": "This account has been deactivated"
}
```

#### 429 Too Many Requests
Rate limit exceeded.

```json
{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Too many refresh attempts. Please try again later",
  "retryAfter": 600
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose `POST /auth/refresh`, requiring only the `refresh_token` cookie (no `Authorization` header).
- **FR-002**: The system MUST store refresh tokens as an irreversible hash; the plaintext token MUST NOT be persisted anywhere.
- **FR-003**: The system MUST rotate the refresh token on every successful use: the presented token is invalidated and a new one, belonging to the same session, is issued.
- **FR-004**: Rotation MUST be atomic — under concurrent requests with the same not-yet-rotated token, exactly one MUST succeed in creating the successor token.
- **FR-005**: The system MUST tolerate replay of an already-rotated token within a 10-second grace window from its rotation, returning the same successor token pair instead of an error, to absorb benign multi-tab races.
- **FR-006**: The system MUST treat replay of an already-rotated token *outside* the grace window as reuse/theft, and respond by revoking every active token belonging to that session (all tokens sharing the same family), not just the presented one.
- **FR-007**: Every token in a session MUST carry the same absolute expiration, fixed at the original login (1 day by default, 30 days with `rememberSession` — see [Login](../Login/spec.md)); no rotation, including the grace-window path, may extend it.
- **FR-008**: The system MUST reject refresh attempts once a session's absolute expiration has passed, regardless of grace window, forcing re-authentication.
- **FR-009**: The system MUST re-verify the owning user's status is `ACTIVE` before issuing a new access token, independently of whether the token itself is otherwise valid.
- **FR-010**: The system MUST rate-limit refresh attempts by the resolved `userId` (20 requests per 10 minutes), once the presented token resolves to a known session — sized to tolerate normal client-side retry behavior, not to deter brute force, which is infeasible regardless against a 256-bit secret. Rate-limiting by IP was considered and deliberately dropped: it is the only place in the codebase that would have needed one (every other rate limit here keys off an identifier already known from the request — email, userId — never a proxy-dependent header), and correctly attributing a client IP behind the project's load balancers turned out to need infrastructure-specific logic disproportionate to what it defends (cheap probing against the token-lookup query, not a hard security boundary). A general, endpoint-agnostic rate limiter — protecting this and every other public endpoint uniformly — is the intended fix if abuse is ever observed, not a per-endpoint IP check.
- **FR-011**: The refresh token MUST never be included in the JSON response body — only delivered via `Set-Cookie`.
- **FR-012**: Two sessions (families) belonging to the same user, created by separate logins (e.g., different devices), MUST be fully independent — revoking or rotating one MUST NOT affect the other.
- **FR-013**: The system MUST periodically purge refresh token records that have been revoked or past their absolute expiration for more than 15 days.
- **FR-014**: Whenever another feature invalidates all of a user's sessions (password change, password reset, self/admin account deactivation), it MUST revoke every active refresh token row for that user (all families), in addition to the existing access-token-level invalidation — see [Login](../Login/spec.md) Session Invalidation Context for why both are required.

### Key Entities

- **RefreshToken**: Represents one token in a session's rotation chain.
  Key attributes:
  - `id` (string, UUID)
  - `userId` (string, reference to User)
  - `familyId` (string, UUID — constant across every rotation of the same session; a new login always starts a new family)
  - `tokenHash` (string — hash of the opaque token value; the plaintext token is never stored)
  - `issuedAt` (timestamp)
  - `absoluteExpiresAt` (timestamp — fixed for the entire family at login time)
  - `revokedAt` (timestamp, nullable — null means this is the currently active token of its family)
  - `replacedById` (string, nullable — the token that superseded this one, if rotated)
  - `userAgent`, `ipAddress` (optional metadata, reserved for a future "active sessions" view — out of scope for this spec)

- **User**: as defined in [Login](../Login/spec.md). This feature additionally depends on `status` being re-checked on every refresh, not only at login.

> **Out of scope for this spec**: a user-facing "manage active sessions" screen (list/revoke individual devices). The data model above is designed to support it later without migration, but no such endpoint is specified here.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can remain authenticated across a multi-hour session without re-entering credentials, as long as at least one request/refresh happens before the access token expires.
- **SC-002**: A stolen refresh token, once the legitimate user's device rotates it, can be used by an attacker at most once before the entire session (including the legitimate device) is revoked.
- **SC-003**: Two tabs of the same browser refreshing near-simultaneously never both get logged out due to the race.
- **SC-004**: No session's effective lifetime ever exceeds its absolute ceiling (1 day or 30 days from login), regardless of how many times it is refreshed.
- **SC-005**: A deactivated account loses the ability to obtain new access tokens on its very next refresh attempt, even if its refresh token is otherwise unexpired.
- **SC-006**: Refresh token records are not retained indefinitely — none should persist more than ~22 days past revocation or expiration (15-day retention, purged on a 7-day cycle).
- **SC-007**: Under normal use, a legitimate user never encounters the refresh rate limit.
- **SC-008**: The refresh token value is never observable in any JSON response body, browser JS context, or server log in plaintext.
