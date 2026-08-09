# Feature Specification: Login (Authentication)

**Created**: 2026-03-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Login with email and password (Priority: P1)

As a registered user, I want to authenticate using my email and password so that I can receive an access token to interact with the platform's protected features, and optionally stay logged in across visits without re-entering my credentials every hour.

> **Note**: This specification covers the authentication flow. The resulting JWT access token is used across the entire system to resolve user identity (`userId`), role, and email. Renewing an expired access token without re-entering credentials is covered by [Refresh Session](../Refresh%20session/spec.md) — this spec only covers issuing the initial access + refresh token pair at login.

**Why this priority**: Authentication is required before any protected action. Without this feature, no user can access any protected endpoint (profile management, problem viewing, contest participation, submissions, etc.).

**Independent Test**: This user story can be tested independently by consuming the `POST /auth/login` endpoint, validating successful login with valid credentials, error responses with invalid credentials, and access restrictions for deactivated accounts.

**Acceptance Scenarios**:

1. **Scenario**: Successful login
   - **Given** a user exists with ACTIVE status and valid credentials
   - **When** the user submits their email and password
   - **Then** the system returns a JWT access token (1-hour expiration) and basic user information
   - **And** the token contains the user's id, email, and role as claims
   - **And** the system sets a refresh token in an `httpOnly; Secure; SameSite=Strict` cookie, host-only scoped to the API's own domain (not shared with the frontend's domain)
   - **And** without `rememberSession` in the request, the refresh token's absolute ceiling is 1 day from login
   - **And** the response includes the absolute session ceiling (`sessionExpiresAt`) so the frontend can compare it against a contest's duration

2. **Scenario**: Successful login with "remember session"
   - **Given** a user exists with ACTIVE status and valid credentials
   - **When** the user submits their email and password with `rememberSession: true`
   - **Then** the system issues a refresh token with an absolute ceiling of 30 days from login (instead of the 1-day default)

3. **Scenario**: Invalid email (user not found)
   - **Given** no user exists with the provided email
   - **When** login credentials are submitted
   - **Then** the system rejects with 401 Unauthorized (INVALID_CREDENTIALS)
   - **And** the error message does not reveal whether the email or password was wrong

4. **Scenario**: Wrong password
   - **Given** a user exists with the provided email
   - **When** an incorrect password is submitted
   - **Then** the system rejects with 401 Unauthorized (INVALID_CREDENTIALS)
   - **And** the error message is identical to the "user not found" case (no information leakage)

5. **Scenario**: Deactivated account
   - **Given** a user exists with DEACTIVATED status
   - **When** the user attempts to login
   - **Then** the system rejects with 403 Forbidden (ACCOUNT_DEACTIVATED)

6. **Scenario**: Missing required fields
   - **Given** the request omits email, password, or both
   - **When** the login request is submitted
   - **Then** the system rejects with 400 Bad Request (VALIDATION_ERROR) with field-level details

7. **Scenario**: Invalid request body
   - **Given** the request body is not valid JSON
   - **When** the login request is submitted
   - **Then** the system rejects with 400 Bad Request (INVALID_JSON)

---

### Edge Cases

- Email with different casing than the stored value (should match case-insensitively since emails are stored in lowercase).
- Empty email or password fields.
- Very long password (up to bcrypt's 72-byte limit).
- Concurrent login requests from the same user.
- Request with extra fields in the body (should be ignored).
- Access token expiration boundary behavior (1 hour).
- `rememberSession` sent as a non-boolean value (should be rejected as a validation error, not silently coerced).
- User already has an active session on another device — a new login MUST NOT revoke that device's session; each login starts an independent refresh token family (see [Refresh Session](../Refresh%20session/spec.md)).

## API Contract

### POST /auth/login

Authenticate a user and receive an access token plus a refresh token.

> **Important**: The login endpoint is public (no authentication required). On success, it returns a short-lived JWT access token that must be included as a Bearer token in the `Authorization` header of all subsequent protected requests, and sets a long-lived refresh token in an `httpOnly` cookie used exclusively by [`POST /auth/refresh`](../Refresh%20session/spec.md) to obtain a new access token without re-entering credentials.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Content-Type | string | Yes | application/json |

**Request Body**:
```json
{
  "email": "string",
  "password": "string",
  "rememberSession": false
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | User's registered email address |
| password | string | Yes | User's password |
| rememberSession | boolean | No (default `false`) | If `true`, the refresh token's absolute session ceiling is 30 days from login. If `false`/omitted, the ceiling is 1 day from login — see [Refresh Session](../Refresh%20session/spec.md) for the full rotation/ceiling model. |

**Responses**:

#### 200 OK
Login successful. Returns JWT access token, session metadata, and user information. Also sets the refresh token cookie.

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "sessionExpiresAt": "2026-08-08T14:30:00Z",
  "user": {
    "email": "user@example.com",
    "name": "John Doe",
    "nickname": "johnd",
    "country": "Colombia",
    "city": "Cúcuta",
    "institution": "UFPS",
    "role": "CONTESTANT",
    "createdAt": "2025-12-13T10:30:00Z"
  }
}
```

**Set-Cookie**:
| Cookie | Attributes | Description |
|--------|-----------|--------------|
| `refresh_token` | `HttpOnly; Secure; SameSite=Strict; Path=/auth; Max-Age=<seconds until sessionExpiresAt>` | Opaque, high-entropy random token (not a JWT — never readable by client JS). Host-only cookie (no `Domain` attribute), sent only to `/auth/*` endpoints on the API's own origin. |

> **Note**: The `id` is not returned in the response body, but is encoded inside the JWT access token as a claim. `sessionExpiresAt` is the absolute ceiling of the refresh token issued in the cookie (see the absolute ceiling model in [Refresh Session](../Refresh%20session/spec.md)) — the frontend uses it to warn the user if a session will expire mid-contest (separate pending frontend item), not to read/manage the cookie itself.

#### JWT Access Token Claims

The generated JWT access token MUST contain the following claims:

| Claim | Type | Description | Used By |
|-------|------|-------------|---------|
| `sub` | string (UUID) | User's unique identifier (`userId`) | All endpoints that need to resolve user identity (GET /users/me, PUT /users, submissions, standings, storage paths, audit logs) |
| `email` | string | User's email address | Endpoints that compare requester identity (e.g., "is this my own profile?") |
| `role` | string | User's role (ADMIN, COACH, CONTESTANT) | All permission checks (Admin endpoints, group management, problem access, etc.) |
| `exp` | integer | Token expiration time (Unix timestamp, 1 hour from `iat`) | Token validation |
| `iat` | integer | Token issued at (Unix timestamp) | Token metadata |

> **Why these claims?** Based on system-wide usage analysis:
> - **`sub` (userId)**: Used by every protected endpoint to resolve user identity. Referenced explicitly in storage paths (`{problemId}/{userId}/...`), submission ownership (`submittedBy`), standings (`standingId`), and audit logs.
> - **`email`**: Used for identity comparisons (e.g., determining if a user is viewing their own profile to decide which fields to return).
> - **`role`**: Used for authorization checks on every protected endpoint (Admin-only endpoints, modifier checks, group management permissions).

#### 400 Bad Request
Validation error (missing required fields).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "email",
      "message": "Email is required"
    },
    {
      "field": "password",
      "message": "Password is required"
    }
  ]
}
```

#### 401 Unauthorized
Invalid credentials (wrong email or password).

```json
{
  "error": "INVALID_CREDENTIALS",
  "message": "Invalid email or password"
}
```

> **Security**: The error message is intentionally generic. It never reveals whether the email exists in the system or the password was incorrect, preventing account enumeration attacks.

#### 403 Forbidden
Account is deactivated.

```json
{
  "error": "ACCOUNT_DEACTIVATED",
  "message": "This account has been deactivated"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow users to authenticate via a public login endpoint.
- **FR-002**: The system MUST validate the presence of required fields (email, password) before processing.
- **FR-003**: The system MUST look up users by email in a case-insensitive manner (emails are stored in lowercase).
- **FR-004**: The system MUST verify the provided password against the stored bcrypt hash.
- **FR-005**: The system MUST NOT allow login for users with DEACTIVATED status, returning 403 Forbidden.
- **FR-006**: The system MUST return identical error messages for "user not found" and "wrong password" cases to prevent account enumeration.
- **FR-007**: The system MUST generate a JWT access token containing `sub` (userId), `email`, `role`, `exp`, and `iat` claims upon successful authentication.
- **FR-008**: The JWT access token MUST have a configurable expiration time (default: 1 hour).
- **FR-009**: The system MUST return the access token, `sessionExpiresAt`, and user profile information (excluding `id` and `password`) upon successful login.
- **FR-010**: The system MUST return validation, authentication, and authorization errors with a consistent structure and clear messages.
- **FR-011**: The system MUST issue a refresh token alongside the access token on every successful login, set as an `httpOnly; Secure; SameSite=Strict` cookie, never exposed in the JSON response body.
- **FR-012**: The system MUST accept an optional `rememberSession` boolean in the login request. When `true`, the refresh token's absolute expiration MUST be 30 days from login; when `false` or omitted, it MUST be 1 day from login.
- **FR-013**: Each login MUST start an independent refresh token session (its own family), never invalidating sessions from other devices — see [Refresh Session](../Refresh%20session/spec.md) for rotation, reuse detection, and revocation behavior.

### Key Entities

- **User**: Represents a registered person in the system.
  Key attributes for this feature:
  - `id` (string, UUID, used as JWT `sub` claim — never returned in response body)
  - `email` (string, unique, used for lookup and as JWT claim)
  - `password` (string, bcrypt hash, used for verification — never returned in responses)
  - `role` (enum: ADMIN | COACH | CONTESTANT, included as JWT claim)
  - `status` (enum: ACTIVE | DEACTIVATED, determines login eligibility)

- **RefreshToken**: Represents one session issued at login, owned by a `User`. Full lifecycle (rotation,
  reuse detection, absolute ceiling, storage schema) is defined in [Refresh Session](../Refresh%20session/spec.md).
  This spec only creates the first row of a new session (family) at login time.

### Session Invalidation Context

Sessions are invalidated in the following scenarios (handled by other features):
- Password change (all sessions, see [Update Password](../Update%20password/spec.md))
- Password recovery/reset (all sessions, see [Recover Password](../Recover%20password/spec.md))
- Account deactivation (all sessions, see [Self Deactivate](../Self%20deactivated%20user/spec.md))
- Refresh token reuse detected (that session only, see [Refresh Session](../Refresh%20session/spec.md))
- Refresh token absolute ceiling reached (that session only, see [Refresh Session](../Refresh%20session/spec.md))

> **Note**: Two independent mechanisms cooperate here, and both MUST fire together on every "invalidate all
> sessions" event — they are not interchangeable:
> - `SessionInvalidator` (revocation by timestamp) invalidates access tokens already in circulation. It has
>   to be checked on every authenticated request, so it stays a cheap timestamp comparison, independent of
>   the refresh token table.
> - Revoking every active row in `refresh_tokens` for the user (all families, not just one) stops that user
>   from minting *new* access tokens afterwards. Without this, a still-valid refresh token issued before the
>   event could keep calling `/auth/refresh` and produce access tokens with an `iat` *after* the
>   `SessionInvalidator` cutoff — which would pass its check and defeat the "logout everywhere" intent.
>
> `POST /auth/refresh` MUST also re-check the user's current `status` before issuing a new access token
> (defense in depth for deactivation, independent of whether the revocation call above already ran).
> See [Refresh Session](../Refresh%20session/spec.md) for the revocation and status-check details.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system allows login with valid credentials, returning HTTP 200 with a JWT access token, `sessionExpiresAt`, and user data, and sets the refresh token cookie.
- **SC-002**: The system rejects login attempts with invalid credentials, returning HTTP 401 with a generic error message.
- **SC-003**: The system rejects login attempts from deactivated accounts, returning HTTP 403.
- **SC-004**: The system rejects login attempts with missing required fields, returning HTTP 400 with field-level validation errors.
- **SC-005**: The JWT access token contains `sub` (userId), `email`, `role`, `exp`, and `iat` claims, with `exp` 1 hour after `iat`.
- **SC-006**: The error message for invalid credentials does not reveal whether the email exists in the system.
- **SC-007**: Login can be successfully completed in a single call to the API with no external dependencies.
- **SC-008**: With `rememberSession: true`, the issued refresh token's absolute ceiling is 30 days from login; without it, 1 day.
- **SC-009**: Logging in from a second device does not invalidate the refresh token session of the first device.
