# Feature Specification: Update User Email

**Created**: 2025-12-13  

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Request email change verification code (Priority: P1)

As an authenticated user, I want to request an email change by providing my password and new email address, so that a verification code is sent to the new email to confirm I have access to it.

**Why this priority**: This is the entry point for the email change flow. Requiring the password ensures account ownership, and sending the code to the new email prevents typos and ensures the user has access to the new address.

**Independent Test**: This user story can be tested independently by consuming the `POST /users/email-change/request` endpoint with password and new email, validating that a verification code is generated and sent to the new email address. The user is identified via the authentication token.

**Acceptance Scenarios**:

1. **Scenario**: Successful verification code request
   - **Given** an authenticated user provides their correct password and a valid new email
   - **When** the user requests an email change verification code
   - **Then** the system generates a verification code and sends it to the new email address

2. **Scenario**: Incorrect password
   - **Given** an authenticated user provides an incorrect password
   - **When** the user requests an email change verification code
   - **Then** the system rejects the operation with an authentication error

3. **Scenario**: New email already in use
   - **Given** the new email address is already registered to another user
   - **When** the user requests an email change verification code
   - **Then** the system rejects the operation with a duplicate email error

4. **Scenario**: Invalid new email format
   - **Given** the new email address does not comply with a valid format
   - **When** the user requests an email change verification code
   - **Then** the system rejects the operation with a validation error

5. **Scenario**: Request code while another code is still valid
   - **Given** a user has already requested a verification code that has not expired
   - **When** the user requests another verification code with correct password
   - **Then** the system invalidates the previous code and generates a new one for the new email

6. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** an email change verification code is requested
   - **Then** the system rejects the operation with an authentication error

---

### User Story 2 – Confirm email change with verification code (Priority: P1)

As an authenticated user, I want to submit the verification code I received in my new email so that I can complete the email change process securely.

**Why this priority**: This completes the email change flow. Without code confirmation, there's no way to verify the user has access to the new email address.

**Independent Test**: This user story can be tested independently by consuming the `POST /users/email-change/confirm` endpoint with a valid code, validating the email is updated successfully. The user is identified via the authentication token.

**Acceptance Scenarios**:

1. **Scenario**: Successful email change
   - **Given** a user has a valid, non-expired verification code
   - **When** the user submits the correct code
   - **Then** the system updates the user's email to the new address and returns success confirmation

2. **Scenario**: Invalid verification code
   - **Given** the user submits an incorrect verification code
   - **When** the email change confirmation is attempted
   - **Then** the system rejects the operation with an invalid code error

3. **Scenario**: Expired verification code
   - **Given** the verification code has expired
   - **When** the user submits the expired code
   - **Then** the system rejects the operation indicating the code has expired

4. **Scenario**: No pending email change request
   - **Given** the user has no active email change request
   - **When** the user attempts to confirm with a code
   - **Then** the system rejects the operation indicating no pending request exists

5. **Scenario**: Multiple failed attempts within valid period
   - **Given** the user has made several failed verification attempts
   - **When** another confirmation attempt is made with the correct code before expiration
   - **Then** the system accepts the operation and updates the email (no attempt limit while code is valid)

---

### Edge Cases

- Attempt to change email to the same current email address.
- Concurrent requests to confirm email change with the same code.
- User requests a new code immediately after the previous one was used successfully.
- New email contains Unicode characters or unusual but valid formats.
- Code submission with leading/trailing whitespace.
- User account is deactivated or suspended during the email change process.
- Email delivery failure to the new email address.
- Another user registers the new email between request and confirmation.
- Password change occurs between request and confirmation.

## API Contract

### POST /users/email-change/request

Request a verification code to change the user's email address.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Request Body**:
```json
{
  "password": "string",
  "newEmail": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| password | string | Yes | User's current password for identity verification |
| newEmail | string | Yes | The new email address to change to |

**Responses**:

#### 200 OK
Verification code sent successfully to the new email address.

```json
{
  "message": "Verification code sent to the new email address",
  "expiresAt": "2025-12-13T15:30:00Z"
}
```

#### 400 Bad Request
Validation error in the request.

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "newEmail",
      "message": "Invalid email format"
    }
  ]
}
```

#### 401 Unauthorized
Authentication failed (invalid token or incorrect password).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid credentials"
}
```

#### 409 Conflict
The new email is already registered to another user.

```json
{
  "error": "EMAIL_ALREADY_EXISTS",
  "message": "The email address is already in use"
}
```

---

### POST /users/email-change/confirm

Confirm the email change with the verification code received.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Request Body**:
```json
{
  "code": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| code | string | Yes | 6-digit verification code received in the new email |

**Responses**:

#### 200 OK
Email changed successfully.

```json
{
  "message": "Email updated successfully",
  "email": "newemail@example.com"
}
```

#### 400 Bad Request
Invalid or expired verification code.

```json
{
  "error": "INVALID_CODE",
  "message": "The verification code is invalid or has expired"
}
```

#### 404 Not Found
No pending email change request exists.

```json
{
  "error": "NO_PENDING_REQUEST",
  "message": "No pending email change request found"
}
```

#### 409 Conflict
The new email was registered by another user after the request was initiated.

```json
{
  "error": "EMAIL_ALREADY_EXISTS",
  "message": "The email address is already in use"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST require user authentication to initiate an email change request.
- **FR-002**: The system MUST require the user's current password to initiate an email change request.
- **FR-003**: The system MUST validate the new email format and uniqueness before generating a verification code.
- **FR-004**: The system MUST generate a verification code and send it to the NEW email address.
- **FR-005**: The system MUST validate that the verification code matches before updating the email.
- **FR-006**: The system MUST ensure the new email is unique across the whole system at the moment of confirmation.
- **FR-007**: The system MUST expire verification codes after 15 minutes.
- **FR-008**: The system MUST allow unlimited verification attempts while the code has not expired.
- **FR-009**: The system MUST invalidate the verification code after successful email change.
- **FR-010**: The system MUST return validation errors with a consistent structure and clear messages.
- **FR-011**: The system MUST ensure only one active verification code exists per user at any time.
- **FR-012**: The system MUST send a security notification to the old email address after successful email change.
- **FR-013**: The system MUST send a confirmation notification to the new email address after successful email change.

### Key Entities

- **User**: Registered person in the system (as defined in Create User spec).  
  Relevant attributes for this feature:
  - `id` (string, UUID)
  - `email` (string, unique)

- **EmailChangeRequest**: Represents a pending email change verification.  
  Key attributes:
  - `id` (string, UUID)
  - `userId` (string, reference to User)
  - `verificationCode` (string, 6-digit numeric code)
  - `newEmail` (string)
  - `expiresAt` (timestamp, 15 minutes from creation)
  - `status` (enum: PENDING, COMPLETED, EXPIRED, CANCELLED)
  - `createdAt` (timestamp)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system requires correct password before generating a verification code.
- **SC-002**: The system successfully sends a verification code to the NEW email address upon valid request.
- **SC-003**: The system allows email change only when the correct, non-expired verification code is provided.
- **SC-004**: The system rejects email change requests with duplicate emails, returning a validation error.
- **SC-005**: 100% of email changes maintain email uniqueness across the system.
- **SC-006**: Verification codes expire after 15 minutes.
- **SC-007**: Only one active verification code exists per user at any time; requesting a new code invalidates the previous one.
- **SC-008**: Notifications are sent to both the old and new email addresses after successful change.
- **SC-009**: Validation errors include clear messages and a consistent structure.


