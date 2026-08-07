# Feature Specification: Update Password

**Created**: 2025-12-13  

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Update user password (Priority: P1)

As an authenticated user, I want to update my account password so that I can keep my account secure and regain control if my credentials may have been compromised.

**Why this priority**: Password security is critical for protecting user accounts and the integrity of the platform. Allowing users to update their password enables proactive security practices and reduces the risk of unauthorized access.

**Independent Test**: This user story can be tested independently by consuming the `PUT /users/password` endpoint, validating successful password updates, validation errors, and security constraints. The user identity is always resolved from the authentication token.

**Acceptance Scenarios**:

1. **Scenario**: Successful password update
   - **Given** a user exists in the system
   - **And** the user is authenticated and identified by the access token
   - **And** the provided current password matches the stored password
   - **And** the new password meets the security requirements
   - **When** the password update request is submitted
   - **Then** the system updates the user password, invalidates all sessions, and sends a notification email

2. **Scenario**: Incorrect current password
   - **Given** a user exists in the system
   - **And** the user is authenticated
   - **When** the provided current password does not match the stored password
   - **Then** the system rejects the operation with a validation error indicating the current password is incorrect

3. **Scenario**: Weak new password
   - **Given** a user exists in the system
   - **And** the user is authenticated
   - **When** the new password does not meet the defined security requirements
   - **Then** the system rejects the operation with a validation error describing the password rules

4. **Scenario**: New password same as current password
   - **Given** a user exists in the system
   - **And** the user is authenticated
   - **When** the new password is identical to the current password
   - **Then** the system rejects the operation with a validation error indicating that the new password must be different

5. **Scenario**: Too many failed attempts
   - **Given** a user has made 5 failed password update attempts
   - **When** another password update attempt is made
   - **Then** the system rejects the operation with a rate limit error and enforces a 1-hour cooldown

6. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a password update request is submitted
   - **Then** the system rejects the operation with an authentication error

7. **Scenario**: User not found
   - **Given** no user exists for the authenticated token
   - **When** a password update request is submitted
   - **Then** the system rejects the operation with a not found error

---

### Edge Cases

- Current password is empty or null.
- New password is empty or null.
- New password contains leading or trailing whitespace.
- Use of Unicode characters in the password.
- Concurrent password update requests for the same authenticated user.
- User attempts password update during cooldown period.
- Token expires mid-request during password update.

## API Contract

### PUT /users/password

Update the authenticated user's password.

> **Important**: The user identity is resolved exclusively from the authentication token. No user identifier is accepted in the request. Upon successful update, all active sessions (including the current one) are invalidated, requiring the user to log in again. This includes revoking every active refresh token for the user — see [Refresh Session](../Refresh%20session/spec.md) for why both the access-token-level and refresh-token-level revocation are required.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Request Body**:
```json
{
  "currentPassword": "string",
  "newPassword": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| currentPassword | string | Yes | User's current password for verification |
| newPassword | string | Yes | New password (must meet security requirements) |

**Password Requirements**:
- Minimum 8 characters
- At least 1 uppercase letter (A-Z)
- At least 1 special character (!@#$%^&*()_+-=[]{}|;:',.<>?/)
- At least 1 number (0-9)
- Must be different from current password

**Responses**:

#### 204 No Content
Password updated successfully. All sessions invalidated. A notification email is sent to the user.

*(No response body)*

#### 400 Bad Request
Validation error (incorrect current password, weak new password, same password).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Current password is incorrect"
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Password does not meet security requirements",
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

```json
{
  "error": "VALIDATION_ERROR",
  "message": "New password must be different from current password"
}
```

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 404 Not Found
User not found for the authenticated token.

```json
{
  "error": "NOT_FOUND",
  "message": "User not found"
}
```

#### 429 Too Many Requests
Rate limit exceeded (5 failed attempts, 1-hour cooldown).

```json
{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Too many failed password update attempts. Please try again later",
  "retryAfter": 3600
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow authenticated users to update their own password.
- **FR-002**: The system MUST identify the user exclusively from the authentication token.
- **FR-003**: The system MUST validate that the authenticated user exists before processing the request.
- **FR-004**: The system MUST verify that the provided current password matches the stored password.
- **FR-005**: The system MUST enforce password security rules for the new password:
  - Minimum 8 characters
  - At least 1 uppercase letter
  - At least 1 numeric character
  - At least 1 special character
- **FR-006**: The system MUST reject the update if the new password is equal to the current password.
- **FR-007**: The system MUST securely hash the new password before persisting it.
- **FR-008**: The system MUST invalidate all active sessions (including the current one) upon successful password change — both by bumping the access-token revocation timestamp and by revoking every active refresh token row for the user (see [Refresh Session](../Refresh%20session/spec.md)).
- **FR-009**: The system MUST send a notification email to the user upon successful password change.
- **FR-010**: The system MUST update the `updatedAt` timestamp after a successful password change.
- **FR-011**: The system MUST limit failed password update attempts to 5 per hour, enforcing a 1-hour cooldown after exceeding the limit.
- **FR-012**: The system MUST return validation and security errors with a consistent structure and clear messages.

### Key Entities

- **User**: Registered person in the system.  
  Relevant attributes for this feature:
  - `id` (string, UUID)
  - `email` (string, for notification)
  - `password` (string, hashed, never exposed)
  - `updatedAt` (timestamp)

- **PasswordUpdateAttempt**: Tracks failed password update attempts per user.  
  Key attributes:
  - `userId` (string, reference to User)
  - `failedAttempts` (integer)
  - `lastAttemptAt` (timestamp)
  - `cooldownUntil` (timestamp, nullable)

> **Note**: The password is never stored or returned in plain text and is never exposed through any API response.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system successfully updates the user password and returns HTTP 204.
- **SC-002**: The system rejects password updates when the current password is incorrect.
- **SC-003**: The system enforces all password security rules (8+ chars, 1 uppercase, 1 special char, 1 number).
- **SC-004**: The system rejects password updates when the new password equals the current password.
- **SC-005**: All active sessions (including current) are invalidated upon successful password change.
- **SC-006**: A notification email is sent to the user upon successful password change.
- **SC-007**: The system enforces rate limiting (5 failed attempts, 1-hour cooldown).
- **SC-008**: The system never returns or logs plain text passwords.
- **SC-009**: The `updatedAt` value is updated after every successful password change.
- **SC-010**: Validation errors include clear messages and a consistent structure.

