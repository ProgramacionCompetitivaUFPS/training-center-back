# Feature Specification: Admin Deactivate User

**Created**: 2026-01-17

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Admin deactivates a user account (Priority: P1)

As a system administrator, I want to deactivate a user's account so that I can ban problematic users, handle compromised accounts, or enforce platform policies.

**Why this priority**: Administrators need the ability to remove users from the platform for security and governance reasons. Unlike self-deactivation, this does not require user consent or confirmation codes.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/users/{id}/deactivate` endpoint with valid admin authentication, validating that the target user's status changes to DEACTIVATED and all associated effects are applied.

**Acceptance Scenarios**:

1. **Scenario**: Successful user deactivation by admin
   - **Given** an authenticated user has the ADMIN role
   - **And** a target user exists with status ACTIVE and role CONTESTANT or COACH
   - **When** the admin requests to deactivate the user
   - **Then** the system:
     - Changes the user's status to DEACTIVATED
     - Records `deactivatedAt` timestamp
     - Unlinks the email from the account (sets to NULL)
     - Anonymizes the nickname to `user_anonimo_{10-char-uuid}`
     - Invalidates all active sessions and tokens for the target user
   - **And** responds with 204 No Content

2. **Scenario**: Admin attempts to deactivate another admin
   - **Given** an authenticated user has the ADMIN role
   - **And** the target user also has the ADMIN role
   - **When** the admin requests to deactivate the target user
   - **Then** the system rejects with 403 Forbidden (CANNOT_DEACTIVATE_ADMIN)

3. **Scenario**: Admin attempts to deactivate themselves
   - **Given** an authenticated user has the ADMIN role
   - **When** the admin attempts to deactivate their own account
   - **Then** the system rejects with 403 Forbidden (CANNOT_SELF_DEACTIVATE)

4. **Scenario**: Target user already deactivated
   - **Given** an authenticated user has the ADMIN role
   - **And** the target user has status DEACTIVATED
   - **When** the admin requests to deactivate the user
   - **Then** the system responds with 204 No Content (idempotent)

5. **Scenario**: Target user not found
   - **Given** an authenticated user has the ADMIN role
   - **And** no user exists with the provided ID
   - **When** the admin requests to deactivate the user
   - **Then** the system rejects with 404 Not Found

6. **Scenario**: Non-admin attempts to deactivate user
   - **Given** the authenticated user has role CONTESTANT or COACH
   - **When** they attempt to deactivate another user
   - **Then** the system rejects with 403 Forbidden (ADMIN_REQUIRED)

7. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a deactivation request is submitted
   - **Then** the system rejects with 401 Unauthorized

---

### Edge Cases

- Admin deactivates a user who is currently competing in an active contest (submissions stop, standings anonymized).
- Admin deactivates a Coach who owns groups or created problems (resources preserved, author anonymized).
- Concurrent deactivation requests for the same user (handled idempotently).
- User has pending email change or password recovery requests (all invalidated).
- Deactivated user's email becomes immediately available for new registrations.

## API Contract

### POST /admin/users/{id}/deactivate

Deactivate a user account as an administrator.

> **Important**: 
> - Only ADMIN users can access this endpoint
> - Cannot deactivate other ADMIN users
> - Cannot self-deactivate (use this endpoint on own account)
> - Operation is idempotent (deactivating already deactivated user returns 204)

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for admin authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string (UUID) | Yes | The unique identifier of the user to deactivate |

**Request Body**:

```json
{}
```

*(Empty body - no parameters required)*

**Responses**:

#### 204 No Content
User deactivated successfully (or was already deactivated).

*(No body)*

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
Permission denied.

```json
{
  "error": "ADMIN_REQUIRED",
  "message": "Admin privileges required"
}
```

```json
{
  "error": "CANNOT_DEACTIVATE_ADMIN",
  "message": "Cannot deactivate another administrator"
}
```

```json
{
  "error": "CANNOT_SELF_DEACTIVATE",
  "message": "Administrators cannot deactivate their own account"
}
```

#### 404 Not Found
Target user not found.

```json
{
  "error": "NOT_FOUND",
  "message": "User not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Authorization**
- **FR-001**: The system MUST allow only users with ADMIN role to access this endpoint.
- **FR-002**: The system MUST NOT allow admins to deactivate other users with ADMIN role.
- **FR-003**: The system MUST NOT allow admins to deactivate their own account via this endpoint.

**Deactivation Effects**
- **FR-004**: Upon deactivation, the system MUST change the user's status to DEACTIVATED.
- **FR-005**: The system MUST record `deactivatedAt` timestamp.
- **FR-006**: The system MUST unlink the email from the account (set to NULL).
- **FR-007**: The system MUST anonymize the nickname to format `user_anonimo_{10-char-uuid}`.
- **FR-008**: The system MUST invalidate all active sessions and tokens for the deactivated user.
- **FR-009**: The unlinked email MUST become available for new account registrations.
- **FR-010**: The original nickname MUST become available for use by other accounts.

**Content Preservation**
- **FR-011**: The system MUST preserve all user historical content (submissions, participations, problems, etc.).
- **FR-012**: The system MUST anonymize the deactivated user's identity in public views (rankings, leaderboards, submissions).
- **FR-013**: Resources created by the user (problems, groups) MUST remain functional with author displayed as `user_anonimo`.

**System Behavior**
- **FR-014**: The operation MUST be idempotent (deactivating already deactivated user returns 204).
- **FR-015**: The system MUST update the user's `updatedAt` timestamp.
- **FR-016**: The system MUST record an audit log of the deactivation (adminId, targetUserId, timestamp).
- **FR-017**: The system MUST invalidate any pending email change or password recovery requests.

### Key Entities

- **User**: Registered person in the system.  
  Relevant attributes:
  - `id` (string, UUID)
  - `email` (string, UNIQUE, nullable - set to NULL after deactivation)
  - `nickname` (string, UNIQUE, lowercase - anonymized after deactivation)
  - `role` (enum: ADMIN | COACH | CONTESTANT)
  - `status` (enum: ACTIVE | DEACTIVATED)
  - `deactivatedAt` (timestamp, nullable)
  - `updatedAt` (timestamp)

- **AdminDeactivationAuditLog**: Audit record for admin-initiated deactivations.
  - `id` (string, UUID)
  - `adminId` (string, reference to Admin User)
  - `targetUserId` (string, reference to deactivated User)
  - `originalEmail` (string, email before deactivation)
  - `originalNickname` (string, nickname before deactivation)
  - `occurredAt` (timestamp)
  - `reason` (string, optional - for future enhancement)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Admin can deactivate CONTESTANT or COACH users via `POST /admin/users/{id}/deactivate` with HTTP 204.
- **SC-002**: Admin cannot deactivate other ADMIN users - HTTP 403.
- **SC-003**: Admin cannot self-deactivate via this endpoint - HTTP 403.
- **SC-004**: Non-admin users receive HTTP 403 when attempting to use this endpoint.
- **SC-005**: Deactivated user cannot authenticate through any method.
- **SC-006**: Deactivated user's email becomes available for new registrations.
- **SC-007**: Rankings and leaderboards display `user_anonimo` for deactivated users.
- **SC-008**: Operation is idempotent - deactivating already deactivated user returns HTTP 204.
- **SC-009**: Audit log records admin deactivations with metadata.
- **SC-010**: `updatedAt` and `deactivatedAt` are correctly recorded.


