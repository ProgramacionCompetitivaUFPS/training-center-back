# Feature Specification: Login (Authentication)

**Created**: 2026-03-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Login with email and password (Priority: P1)

As a registered user, I want to authenticate using my email and password so that I can receive an access token to interact with the platform's protected features.

> **Note**: This specification covers the authentication flow. The resulting JWT token is used across the entire system to resolve user identity (`userId`), role, and email.

**Why this priority**: Authentication is required before any protected action. Without this feature, no user can access any protected endpoint (profile management, problem viewing, contest participation, submissions, etc.).

**Independent Test**: This user story can be tested independently by consuming the `POST /auth/login` endpoint, validating successful login with valid credentials, error responses with invalid credentials, and access restrictions for deactivated accounts.

**Acceptance Scenarios**:

1. **Scenario**: Successful login
   - **Given** a user exists with ACTIVE status and valid credentials
   - **When** the user submits their email and password
   - **Then** the system returns a JWT access token and basic user information
   - **And** the token contains the user's id, email, and role as claims

2. **Scenario**: Invalid email (user not found)
   - **Given** no user exists with the provided email
   - **When** login credentials are submitted
   - **Then** the system rejects with 401 Unauthorized (INVALID_CREDENTIALS)
   - **And** the error message does not reveal whether the email or password was wrong

3. **Scenario**: Wrong password
   - **Given** a user exists with the provided email
   - **When** an incorrect password is submitted
   - **Then** the system rejects with 401 Unauthorized (INVALID_CREDENTIALS)
   - **And** the error message is identical to the "user not found" case (no information leakage)

4. **Scenario**: Deactivated account
   - **Given** a user exists with DEACTIVATED status
   - **When** the user attempts to login
   - **Then** the system rejects with 403 Forbidden (ACCOUNT_DEACTIVATED)

5. **Scenario**: Missing required fields
   - **Given** the request omits email, password, or both
   - **When** the login request is submitted
   - **Then** the system rejects with 400 Bad Request (VALIDATION_ERROR) with field-level details

6. **Scenario**: Invalid request body
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
- Token expiration boundary behavior.

## API Contract

### POST /auth/login

Authenticate a user and receive an access token.

> **Important**: The login endpoint is public (no authentication required). On success, it returns a JWT token that must be included as a Bearer token in the `Authorization` header of all subsequent protected requests.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Content-Type | string | Yes | application/json |

**Request Body**:
```json
{
  "email": "string",
  "password": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | User's registered email address |
| password | string | Yes | User's password |

**Responses**:

#### 200 OK
Login successful. Returns JWT access token and user information.

```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
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

> **Note**: The `id` is not returned in the response body, but is encoded inside the JWT token as a claim. The token also contains the user's `email` and `role` as claims.

#### JWT Token Claims

The generated JWT token MUST contain the following claims:

| Claim | Type | Description | Used By |
|-------|------|-------------|---------|
| `sub` | string (UUID) | User's unique identifier (`userId`) | All endpoints that need to resolve user identity (GET /users/me, PUT /users, submissions, standings, storage paths, audit logs) |
| `email` | string | User's email address | Endpoints that compare requester identity (e.g., "is this my own profile?") |
| `role` | string | User's role (ADMIN, COACH, CONTESTANT) | All permission checks (Admin endpoints, group management, problem access, etc.) |
| `exp` | integer | Token expiration time (Unix timestamp) | Token validation |
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
- **FR-007**: The system MUST generate a JWT token containing `sub` (userId), `email`, `role`, `exp`, and `iat` claims upon successful authentication.
- **FR-008**: The JWT token MUST have a configurable expiration time (default: 24 hours).
- **FR-009**: The system MUST return the token along with user profile information (excluding `id` and `password`) upon successful login.
- **FR-010**: The system MUST return validation, authentication, and authorization errors with a consistent structure and clear messages.

### Key Entities

- **User**: Represents a registered person in the system.
  Key attributes for this feature:
  - `id` (string, UUID, used as JWT `sub` claim — never returned in response body)
  - `email` (string, unique, used for lookup and as JWT claim)
  - `password` (string, bcrypt hash, used for verification — never returned in responses)
  - `role` (enum: ADMIN | COACH | CONTESTANT, included as JWT claim)
  - `status` (enum: ACTIVE | DEACTIVATED, determines login eligibility)

### Session Invalidation Context

Token-based sessions are invalidated in the following scenarios (handled by other features):
- Password change (all sessions, see [Update Password](../Update%20password/spec.md))
- Password recovery/reset (all sessions, see [Recover Password](../Recover%20password/spec.md))
- Account deactivation (all sessions, see [Self Deactivate](../Self%20deactivated%20user/spec.md))

> **Note**: Session invalidation is out of scope for this spec. It will be addressed as features that trigger invalidation are implemented. A possible approach includes tracking a `passwordChangedAt` or `tokenVersion` field and validating it against the token's `iat` claim.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system allows login with valid credentials, returning HTTP 200 with a JWT token and user data.
- **SC-002**: The system rejects login attempts with invalid credentials, returning HTTP 401 with a generic error message.
- **SC-003**: The system rejects login attempts from deactivated accounts, returning HTTP 403.
- **SC-004**: The system rejects login attempts with missing required fields, returning HTTP 400 with field-level validation errors.
- **SC-005**: The JWT token contains `sub` (userId), `email`, `role`, `exp`, and `iat` claims.
- **SC-006**: The error message for invalid credentials does not reveal whether the email exists in the system.
- **SC-007**: Login can be successfully completed in a single call to the API with no external dependencies.
