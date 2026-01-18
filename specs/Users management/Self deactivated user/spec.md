# Feature Specification: Self-Deactivate Account

**Created**: 2025-12-20

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Request account deactivation (Priority: P1)

As an authenticated user (Coach or Contestant), I want to request the deactivation of my account to initiate the process of leaving the platform, receiving a confirmation code via email.

**Why this priority**: Protects user privacy and meets expectations of control over their information, while adding a security layer through code confirmation.

**Independent Test**: Testing the `POST /users/deactivation` endpoint validates that a confirmation code is generated, sent to the user's email, and a pending request is registered without yet changing the user's status to DEACTIVATED.

> **Note**: Only users with COACH or CONTESTANT role can request self-deactivation. ADMIN users cannot deactivate their own accounts.

**Acceptance Scenarios**:

1. **Scenario**: Successful deactivation request
   - **Given** an authenticated user with ACTIVE status and COACH or CONTESTANT role
   - **When** they request account deactivation
   - **Then** the system generates a 6-digit confirmation code, sends it to the user's email, registers a pending request, and responds 200 OK

2. **Scenario**: Admin attempts self-deactivation
   - **Given** an authenticated user with ADMIN role
   - **When** they request account deactivation
   - **Then** the system rejects with 403 Forbidden indicating admins cannot self-deactivate

3. **Scenario**: Unauthenticated request
   - **Given** a request without a valid token
   - **When** deactivation is requested
   - **Then** the system rejects with 401 Unauthorized

4. **Scenario**: User not found
   - **Given** the token is valid but does not resolve an existing user
   - **When** deactivation is requested
   - **Then** the system rejects with 404 Not Found

5. **Scenario**: User already deactivated
   - **Given** an authenticated user with DEACTIVATED status
   - **When** they request deactivation again
   - **Then** the system rejects with 409 Conflict (ALREADY_DEACTIVATED) without side effects

6. **Scenario**: Request while another is pending
   - **Given** a user has a pending deactivation request with a valid code
   - **When** they request deactivation again
   - **Then** the system invalidates the previous code, generates a new one, and sends it via email

---

### User Story 2 – Confirm account deactivation with code (Priority: P1)

As a user who has requested account deactivation, I want to confirm the operation by entering the code received via email to complete the deactivation, unlink my email from the account, and have my identity hidden in historical content without deleting my contributions.

**Why this priority**: Completes the deactivation flow and protects against accidental or unauthorized deactivations through explicit confirmation.

**Independent Test**: Testing the `POST /users/deactivation/confirm` endpoint validates the change of user status to DEACTIVATED, email unlinking, session invalidation, identity masking in listings (rankings, submissions, groups), and blocking of all actions including authentication.

> **Important**: Upon deactivation, the email is **completely unlinked** from the account. The user **cannot log in** through any method (password or Google OAuth). The unlinked email becomes **available for registering a new account**.

**Acceptance Scenarios**:

1. **Scenario**: Successful deactivation confirmation
   - **Given** a user has a pending deactivation request with a valid code
   - **When** they enter the correct code to confirm
   - **Then** the system:
     - Changes their status to DEACTIVATED
     - Records `deactivatedAt` timestamp
     - **Unlinks the email from the account** (sets email to NULL)
     - **Anonymizes the nickname** to `user_anonimo_{10-char-uuid}` format
     - Invalidates all active sessions and tokens
     - Sends a confirmation email (to the email before unlinking)
   - **And** all historical content displays with the anonymized nickname
   - **And** the service responds 204 No Content

2. **Scenario**: Incorrect confirmation code
   - **Given** a user has a pending deactivation request
   - **When** they enter an incorrect code
   - **Then** the system rejects with 400 Bad Request (INVALID_CODE) and records a failed attempt
   - **And** the code remains valid if attempts have not been exhausted

3. **Scenario**: Expired confirmation code
   - **Given** the confirmation code has expired (after 15 minutes)
   - **When** the user attempts to confirm with the expired code
   - **Then** the system rejects with 400 Bad Request (EXPIRED_CODE)

4. **Scenario**: Confirmation attempts exhausted
   - **Given** a user has failed 4 confirmation attempts
   - **When** they fail the fifth attempt
   - **Then** the system rejects with 429 Too Many Requests (MAX_ATTEMPTS_EXCEEDED)
   - **And** blocks new attempts for 1 hour
   - **And** invalidates the current code

5. **Scenario**: Confirmation attempt during block
   - **Given** a user is blocked for having exhausted 5 attempts
   - **When** they attempt to confirm deactivation before 1 hour has passed
   - **Then** the system rejects with 429 Too Many Requests indicating remaining time

6. **Scenario**: Confirmation without pending request
   - **Given** a user has no pending deactivation request
   - **When** they attempt to confirm with a code
   - **Then** the system rejects with 404 Not Found (NO_PENDING_REQUEST)

7. **Scenario**: Email becomes available for new registration
   - **Given** a user successfully confirms deactivation
   - **When** someone attempts to register with the previously-used email
   - **Then** the system allows the registration as a new account

8. **Scenario**: Deactivated user cannot authenticate
   - **Given** a user has been deactivated
   - **When** they attempt to log in with password or Google OAuth using the old email
   - **Then** the system rejects authentication (email not found or account deactivated)

9. **Scenario**: Anonymization in rankings and submissions after confirmation
   - **Given** the user had previous participations, records, and submissions
   - **When** they successfully confirm deactivation
   - **Then** in public rankings, contest leaderboards, and submission listings, their identity is displayed as `user_anonimo`
   - **And** name, email, nickname, and institution are hidden
   - **And** scores, times, and results remain unaltered

10. **Scenario**: Blocking all actions after deactivation
    - **Given** a user is in DEACTIVATED status
    - **When** they attempt any action (login, submission, create/edit problem, create contest, comment, etc.)
    - **Then** the system rejects appropriately (authentication failure for login, 403 for authenticated attempts)

11. **Scenario**: Deactivation during an active contest
    - **Given** the user is competing in an ongoing contest
    - **When** they confirm deactivation
    - **Then** they cannot make new submissions
    - **And** their previous history remains on the leaderboard as `user_anonimo`
    - **And** the change is reflected on the leaderboard in < 60 seconds

12. **Scenario**: Coach resources after deactivation
    - **Given** a Coach who created problems confirms deactivation
    - **When** those problems are viewed
    - **Then** the problems remain fully functional and usable
    - **And** the author reference (by ID) is preserved but displayed as `user_anonimo`
    - **And** other authorized users can still modify/manage those problems

13. **Scenario**: Concurrent confirmation
    - **Given** multiple confirmation requests are sent in parallel with the same code
    - **When** one request completes first
    - **Then** subsequent requests do not alter the result and return 409 Conflict or 204 idempotent

---

### Edge Cases

- Concurrent retries and race conditions (idempotency guaranteed).
- Anonymization propagation in caches and real-time leaderboards (SLA < 60s).
- Exports or third-party APIs: must receive anonymized data for deactivated users.
- User with Coach role who is owner/admin of groups/contests: entities are preserved, author displayed as `user_anonimo`; the user cannot administer them while deactivated.
- OAuth token remains valid at the time of the call: all active sessions must be invalidated after deactivation confirmation.
- Localization of `user_anonimo` alias (business-defined constant; does not depend on client language).
- Confirmation code with leading or trailing whitespace: must be normalized before validation.
- Request for new code immediately after exhausting attempts: must wait 1 hour before being able to request a new code.
- User requests new code after failing some attempts: invalidates previous code but does not reset the attempt counter.
- Email delivery failure: code is generated but user does not receive it (must be able to request new code).
- Re-judge triggered on problems created by deactivated Coach: works normally, not dependent on author status.
- Attempting to reuse old credentials after email is unlinked: authentication fails completely.

## API Contract

### POST /users/deactivation

Request the deactivation of the authenticated user's account and send a confirmation code via email.

> **Important**: User identity is resolved exclusively from the token. Only COACH and CONTESTANT can request deactivation. ADMIN cannot self-deactivate.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Request Body**:

```json
{}
```

*(Empty body - no parameters required)*

**Responses**:

#### 200 OK
Request processed successfully. Confirmation code sent via email.

```json
{
  "message": "A confirmation code has been sent to your email"
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

#### 403 Forbidden
User role not allowed to self-deactivate (ADMIN).

```json
{
  "error": "FORBIDDEN",
  "message": "Administrators cannot deactivate their own account"
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

#### 409 Conflict
User is already deactivated.

```json
{
  "error": "ALREADY_DEACTIVATED",
  "message": "User account is already deactivated"
}
```

---

### POST /users/deactivation/confirm

Confirm account deactivation using the confirmation code received via email.

> **Important**: This operation completes the deactivation. After successful confirmation:
> - All sessions/tokens are invalidated
> - The email is **completely unlinked** from the account
> - The user **cannot authenticate** through any method
> - The email becomes **available for new registrations**
> - Global anonymization is activated

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |

**Request Body**:

```json
{
  "code": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| code | string | Yes | 6-digit confirmation code received via email |

**Responses**:

#### 204 No Content
Deactivation confirmed successfully. Sessions invalidated, email unlinked, confirmation email sent.

*(No body)*

#### 400 Bad Request
Invalid or expired code.

```json
{
  "error": "INVALID_CODE",
  "message": "The confirmation code is invalid"
}
```

```json
{
  "error": "EXPIRED_CODE",
  "message": "The confirmation code has expired. Please request a new one"
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
No pending deactivation request or user not found.

```json
{
  "error": "NOT_FOUND",
  "message": "No pending deactivation request found"
}
```

#### 409 Conflict
User already deactivated or code already used.

```json
{
  "error": "ALREADY_DEACTIVATED",
  "message": "User account is already deactivated"
}
```

#### 429 Too Many Requests
Confirmation attempts exhausted. Must wait before retrying.

```json
{
  "error": "MAX_ATTEMPTS_EXCEEDED",
  "message": "Maximum confirmation attempts exceeded. Please try again later",
  "retryAfter": 3600
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Deactivation Request**
- **FR-001**: The system MUST allow authenticated users with COACH or CONTESTANT role to request deactivation of their own account.
- **FR-002**: The system MUST NOT allow users with ADMIN role to deactivate their own account.
- **FR-003**: User identity MUST be resolved exclusively from the authentication token.
- **FR-004**: The system MUST validate that the user exists and is ACTIVE before processing the deactivation request.
- **FR-005**: When requesting deactivation, the system MUST generate a 6-digit numeric confirmation code.
- **FR-006**: The system MUST send the confirmation code to the user's email address.
- **FR-007**: The confirmation code MUST expire after 15 minutes from its generation.
- **FR-008**: When requesting a new confirmation code, the system MUST invalidate the previous code.

**Confirmation & Rate Limiting**
- **FR-009**: The system MUST allow a maximum of 5 confirmation attempts with the same code.
- **FR-010**: If the user exhausts 5 attempts, the system MUST block new attempts for 1 hour.
- **FR-011**: During the 1-hour block, the system MUST reject any confirmation attempt.
- **FR-012**: Requesting a new code after failing some attempts MUST NOT reset the attempt counter.

**Deactivation Effects**
- **FR-013**: Upon successful confirmation, the system MUST change the user's status to DEACTIVATED and record `deactivatedAt`.
- **FR-014**: The system MUST **completely unlink the email** from the deactivated account by setting it to NULL.
- **FR-014.1**: The system MUST **anonymize the nickname** to format `user_anonimo_{10-char-uuid}` (e.g., `user_anonimo_a1b2c3d4e5`).
- **FR-015**: The unlinked email MUST become available for registering a new account.
- **FR-015.1**: The original nickname MUST become available for use by other accounts after anonymization.
- **FR-016**: The system MUST invalidate all active sessions/tokens after successful deactivation confirmation.
- **FR-017**: A deactivated user MUST NOT be able to authenticate through any method (password or Google OAuth).
- **FR-018**: The system MUST send a deactivation confirmation email (to the email before unlinking).

**Content Preservation & Anonymization**
- **FR-019**: The system MUST preserve all user historical content (submissions, participations, problems, etc.) without deleting it.
- **FR-020**: Resources created by the user (problems, etc.) MUST remain linked by the author's internal ID, not by email.
- **FR-021**: The system MUST anonymize the deactivated user's identity in public views:
  - Display name/nickname: `user_anonimo`
  - Name, email, institution: hidden or null
  - Scores, times, and metrics: preserved without exposing PII
- **FR-022**: Anonymization MUST apply to: public rankings, contest leaderboards, submission listings, submission details, group member listings, and any endpoint exposing user identity.
- **FR-023**: Anonymization MUST propagate to leaderboards and caches in < 60 seconds after deactivation confirmation.

**Access Blocking**
- **FR-024**: The system MUST block ALL operations for DEACTIVATED users, including authentication.
- **FR-025**: Any authenticated request from a deactivated user MUST fail with appropriate error (401 for auth, 403 for actions).

**System Integrity**
- **FR-026**: Re-judge operations on problems created by deactivated users MUST function normally.
- **FR-027**: The system MUST be idempotent against concurrent confirmation requests.
- **FR-028**: The system MUST record an audit log of the deactivation (userId, originalEmail, originalNickname, timestamp, IP, userAgent).
- **FR-029**: The system MUST update the user's `updatedAt` when confirming deactivation.
- **FR-030**: Logs and API responses MUST not include PII of deactivated users.

### Key Entities

- **User**: Registered person in the system.  
  Relevant attributes for this feature:
  - `id` (string, UUID)
  - `email` (string, UNIQUE, nullable - set to NULL after deactivation)
  - `name` (string)
  - `institution` (string, optional)
  - `nickname` (string, UNIQUE, lowercase - anonymized to `user_anonimo_{10-char-uuid}` after deactivation)
  - `role` (enum: ADMIN | COACH | CONTESTANT)
  - `status` (enum: ACTIVE | DEACTIVATED)
  - `deactivatedAt` (timestamp, nullable)
  - `updatedAt` (timestamp)

- **DeactivationRequest**: Represents a pending deactivation request.
  - `id` (string, UUID)
  - `userId` (string, reference to User)
  - `verificationCode` (string, 6-digit numeric code)
  - `expiresAt` (timestamp, 15 minutes from creation)
  - `attempts` (integer, failed attempts count, max 5)
  - `blockedUntil` (timestamp, nullable, 1-hour block after exhausting attempts)
  - `status` (enum: PENDING, CONFIRMED, EXPIRED, BLOCKED)
  - `createdAt` (timestamp)

- **DeactivationAuditLog**: Audit record for deactivation.
  - `id` (string, UUID)
  - `userId` (string, reference to User)
  - `originalEmail` (string, email before deactivation, for audit purposes)
  - `originalNickname` (string, nickname before deactivation, for audit purposes)
  - `occurredAt` (timestamp)
  - `ip` (string, nullable)
  - `userAgent` (string, nullable)

> **Note**: Anonymization is logical (at read time). Historical objects are not rewritten; the API/query layer decides what to expose based on `status=DEACTIVATED`.

### Supported Roles

| Role | Can Self-Deactivate |
|------|---------------------|
| CONTESTANT | ✅ Yes |
| COACH | ✅ Yes |
| ADMIN | ❌ No |

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Deactivation request returns HTTP 200 with confirmation code sent via email.
- **SC-002**: ADMIN users receive HTTP 403 when attempting self-deactivation.
- **SC-003**: Successful confirmation returns HTTP 204 with sessions invalidated and email unlinked.
- **SC-004**: Confirmation code expires after 15 minutes.
- **SC-005**: The system allows a maximum of 5 confirmation attempts with the same code.
- **SC-006**: After exhausting 5 attempts, the system blocks new attempts for 1 hour.
- **SC-007**: A deactivated user cannot authenticate through any method (password or Google OAuth).
- **SC-008**: The unlinked email can be used to register a new account.
- **SC-009**: Rankings, leaderboards, and listings display `user_anonimo` for deactivated users.
- **SC-010**: No historical data (submissions, scores, problems) is deleted, only identity is anonymized.
- **SC-011**: Problems and resources created by deactivated users remain functional.
- **SC-012**: Anonymization is reflected in < 60s from deactivation confirmation.
- **SC-013**: Re-judge operations work normally regardless of author status.
- **SC-014**: Concurrent confirmation requests are handled idempotently.
- **SC-015**: Audit log records deactivation with metadata.
- **SC-016**: `updatedAt` and `deactivatedAt` are correctly recorded.

