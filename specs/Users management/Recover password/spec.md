# Feature Specification: Recover Password

**Created**: 2025-12-13  

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Request password recovery code (Priority: P1)

As a user who has forgotten my password, I want to request a recovery code sent to my registered email so that I can regain access to my account.

**Why this priority**: Password recovery is essential for user retention and system accessibility. Users who cannot recover their accounts are effectively locked out of the platform permanently.

**Independent Test**: This user story can be tested independently by consuming the `POST /password/forgot` endpoint, validating that a verification code is generated and sent to the email (if registered), and that the response is always ambiguous regardless of email existence.

**Acceptance Scenarios**:

1. **Scenario**: Successful recovery code request (email exists)
   - **Given** a user exists with the provided email
   - **When** a password recovery request is submitted
   - **Then** the system generates a 6-digit code, sends it to the email, and returns an ambiguous success message

2. **Scenario**: Recovery code request for non-existent email
   - **Given** no user exists with the provided email
   - **When** a password recovery request is submitted
   - **Then** the system returns the same ambiguous success message without sending any email

3. **Scenario**: Request code while another code is still valid
   - **Given** a user has already requested a recovery code that has not expired
   - **When** the user requests another recovery code
   - **Then** the system invalidates the previous code, generates a new one, and counts it as a new request against the rate limit

4. **Scenario**: Rate limit exceeded
   - **Given** a user has exceeded 5 recovery requests in the last hour
   - **When** another recovery request is submitted
   - **Then** the system rejects the operation with a rate limit error

5. **Scenario**: Invalid email format
   - **Given** the provided email does not comply with a valid format
   - **When** a password recovery request is submitted
   - **Then** the system rejects the operation with a validation error

---

### User Story 2 – Reset password with verification code (Priority: P1)

As a user who has received a recovery code, I want to submit the code along with my new password so that I can regain access to my account with a new password.

**Why this priority**: This completes the password recovery flow. Without code confirmation and password reset, users cannot regain access to their accounts.

**Independent Test**: This user story can be tested independently by consuming the `POST /password/reset` endpoint with a valid code, email, and new password, validating the password is updated successfully and all sessions are invalidated.

**Acceptance Scenarios**:

1. **Scenario**: Successful password reset
   - **Given** a user has a valid, non-expired recovery code
   - **When** the user submits the correct code, email, and a valid new password
   - **Then** the system updates the password, invalidates all active sessions, and returns success confirmation

2. **Scenario**: Invalid verification code
   - **Given** the user submits an incorrect recovery code
   - **When** the password reset is attempted
   - **Then** the system rejects the operation with an invalid code error

3. **Scenario**: Expired verification code
   - **Given** the recovery code has expired (after 15 minutes)
   - **When** the user submits the expired code
   - **Then** the system rejects the operation indicating the code has expired

4. **Scenario**: No pending recovery request
   - **Given** the user has no active password recovery request
   - **When** the user attempts to reset password with a code
   - **Then** the system rejects the operation indicating no pending request exists

5. **Scenario**: Password does not meet complexity requirements
   - **Given** the user has a valid recovery code
   - **When** the user submits a password that does not meet the complexity requirements
   - **Then** the system rejects the operation with a validation error detailing the requirements

6. **Scenario**: Email not found
   - **Given** the provided email does not exist in the system
   - **When** the password reset is attempted
   - **Then** the system rejects the operation with an invalid request error

---

### Edge Cases

- Attempt to reset password with an empty or whitespace-only password.
- Concurrent requests to reset password with the same code.
- User requests a new code immediately after the previous one was used successfully.
- Code submission with leading/trailing whitespace.
- User account is deactivated or suspended during the recovery process.
- Email delivery failure.
- Multiple recovery requests from different IP addresses for the same email.
- Attempt to reuse an already-used recovery code.
- Password that meets length but fails other complexity requirements.

## API Contract

### POST /password/forgot

Request a password recovery code to be sent to the registered email.

> **Note**: This is a **public endpoint** (no authentication required). The response is intentionally ambiguous to prevent email enumeration attacks.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Content-Type | string | Yes | application/json |

**Request Body**:
```json
{
  "email": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | The email address associated with the account |

**Responses**:

#### 200 OK
Request processed (response is the same whether email exists or not).

```json
{
  "message": "If the email is registered, a recovery code has been sent"
}
```

#### 400 Bad Request
Validation error (invalid email format).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "email",
      "message": "Invalid email format"
    }
  ]
}
```

#### 429 Too Many Requests
Rate limit exceeded.

```json
{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Too many recovery requests. Please try again later",
  "retryAfter": 3600
}
```

---

### POST /password/reset

Reset the password using the verification code received via email.

> **Note**: This is a **public endpoint** (no authentication required). Upon successful reset, all active sessions for the user are invalidated — including every active refresh token, not only outstanding access tokens (see [Refresh Session](../Refresh%20session/spec.md)).

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Content-Type | string | Yes | application/json |

**Request Body**:
```json
{
  "email": "string",
  "code": "string",
  "newPassword": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | The email address associated with the account |
| code | string | Yes | 6-digit verification code received via email |
| newPassword | string | Yes | New password (min 8 chars, 1 uppercase, 1 special char, 1 number) |

**Password Requirements**:
- Minimum 8 characters
- At least 1 uppercase letter (A-Z)
- At least 1 special character (!@#$%^&*()_+-=[]{}|;:',.<>?/)
- At least 1 number (0-9)

**Responses**:

#### 200 OK
Password reset successfully. All sessions invalidated.

```json
{
  "message": "Password has been reset successfully. Please log in with your new password"
}
```

#### 400 Bad Request
Validation error (invalid code, expired code, password complexity, or invalid request).

```json
{
  "error": "INVALID_CODE",
  "message": "The recovery code is invalid or has expired"
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Password does not meet complexity requirements",
  "details": [
    {
      "field": "newPassword",
      "message": "Password must contain at least one uppercase letter"
    },
    {
      "field": "newPassword",
      "message": "Password must contain at least one special character"
    }
  ]
}
```

#### 404 Not Found
No pending recovery request or email not found.

```json
{
  "error": "NOT_FOUND",
  "message": "No pending password recovery request found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow users to request a password recovery code via a public endpoint.
- **FR-002**: The system MUST generate a 6-digit numeric verification code and send it to the registered email.
- **FR-003**: The system MUST return an ambiguous response regardless of whether the email exists in the system.
- **FR-004**: The system MUST expire recovery codes after 15 minutes.
- **FR-005**: The system MUST limit recovery requests to 5 per email per hour.
- **FR-006**: The system MUST invalidate previous recovery codes when a new one is requested.
- **FR-007**: The system MUST validate that the new password meets complexity requirements (min 8 chars, 1 uppercase, 1 special char, 1 number).
- **FR-008**: The system MUST securely hash the new password before storing it.
- **FR-009**: The system MUST invalidate all active sessions for the user upon successful password reset — both by bumping the access-token revocation timestamp and by revoking every active refresh token row for the user (see [Refresh Session](../Refresh%20session/spec.md)).
- **FR-010**: The system MUST invalidate the recovery code after successful password reset.
- **FR-011**: The system MUST return validation errors with a consistent structure and clear messages.

### Key Entities

- **User**: Registered person in the system (as defined in Create User spec).  
  Relevant attributes for this feature:
  - `id` (string, UUID)
  - `email` (string, unique)
  - `password` (string, hashed)

- **PasswordRecoveryRequest**: Represents a pending password recovery verification.  
  Key attributes:
  - `id` (string, UUID)
  - `userId` (string, reference to User)
  - `email` (string)
  - `verificationCode` (string, 6-digit numeric code)
  - `expiresAt` (timestamp, 15 minutes from creation)
  - `status` (enum: PENDING, COMPLETED, EXPIRED)
  - `createdAt` (timestamp)

- **RecoveryRateLimit**: Tracks recovery request attempts per email.  
  Key attributes:
  - `email` (string)
  - `requestCount` (integer)
  - `windowStart` (timestamp, 1-hour window)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system returns an ambiguous response for password recovery requests regardless of email existence.
- **SC-002**: The system successfully sends a 6-digit recovery code to registered emails.
- **SC-003**: Recovery codes expire after 15 minutes.
- **SC-004**: The system enforces a rate limit of 5 requests per email per hour.
- **SC-005**: The system validates password complexity (8+ chars, 1 uppercase, 1 special char, 1 number).
- **SC-006**: All active sessions are invalidated upon successful password reset.
- **SC-007**: Only one active recovery code exists per user at any time; requesting a new code invalidates the previous one.
- **SC-008**: New passwords are securely hashed before storage.
- **SC-009**: Validation errors include clear messages and a consistent structure.

