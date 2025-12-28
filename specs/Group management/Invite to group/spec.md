# Feature Specification: Invite to Group

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create invitation (Priority: P1)

As a Group Admin, I want to invite users to join my group so that I can proactively bring relevant people into the group.

**Why this priority**: Invitations are the primary mechanism for controlled group growth. Admins need to be able to invite specific users to build their communities and courses.

**Independent Test**: Authenticated Group Admin POST `/api/groups/{g}/invitations` with valid user. Verify invitation created with secure token and `status = PENDING`.

**Acceptance Scenarios**:

1. **Scenario**: Admin creates invitation for existing user by user ID

   * **Given** requesting user is admin of group `g`
   * **And** target user exists and is not already a member
   * **When** admin creates invitation with `invitee_user_id`
   * **Then** system creates invitation with UUID token and `status = PENDING`
   * **And** returns 201 Created with invitation details (token included)

2. **Scenario**: Admin creates invitation for existing user by username

   * **Given** requesting user is admin of group `g`
   * **And** target username exists and user is not already a member
   * **When** admin creates invitation with `invitee_username`
   * **Then** system resolves username to user ID and creates invitation
   * **And** returns 201 Created with invitation details

3. **Scenario**: Admin creates invitation by email

   * **Given** requesting user is admin of group `g`
   * **When** admin creates invitation with `invitee_email`
   * **Then** system creates invitation for email address
   * **And** invitation can be claimed when user with that email logs in

3. **Scenario**: Admin invites user who is already member

   * **Given** target user is already a member of the group
   * **When** admin tries to invite them
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

4. **Scenario**: Non-admin attempts to create invitation

   * **Given** requesting user is not admin of the group
   * **When** they try to create invitation
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

5. **Scenario**: Admin creates invitation with expiry

   * **Given** admin provides `expires_at` timestamp
   * **When** invitation is created
   * **Then** system stores expiry and will reject acceptance after that time

6. **Scenario**: Admin invites by invalid username

   * **Given** admin provides `invitee_username` that doesn't exist
   * **When** invitation creation is attempted
   * **Then** system rejects with 404 (`USER_NOT_FOUND`)

---

### User Story 2 - Manage pending invitations (Priority: P2)

As a Group Admin, I want to view and manage pending invitations so that I can track who has been invited and cancel invitations if needed.

**Why this priority**: Admins need visibility into outstanding invitations to manage group growth and clean up unused invitations.

**Independent Test**: Admin GET `/api/groups/{g}/invitations` returns list of pending invitations. Admin DELETE `/api/groups/{g}/invitations/{id}` cancels invitation.

**Acceptance Scenarios**:

1. **Scenario**: Admin lists pending invitations

   * **Given** group has several invitations in different states
   * **When** admin requests invitation list
   * **Then** system returns invitations with status, created date, invitee info
   * **And** expired invitations are marked appropriately

2. **Scenario**: Admin cancels pending invitation

   * **Given** invitation exists with `status = PENDING`
   * **When** admin cancels the invitation
   * **Then** invitation status becomes `CANCELLED`
   * **And** token becomes invalid for acceptance

3. **Scenario**: Admin attempts to cancel accepted invitation

   * **Given** invitation has `status = ACCEPTED`
   * **When** admin tries to cancel it
   * **Then** system rejects with 400 (`INVITATION_ALREADY_PROCESSED`)

4. **Scenario**: Non-admin attempts to view invitations

   * **Given** requesting user is not admin of the group
   * **When** they try to list invitations
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 3 - Handle invitation expiry (Priority: P3)

As a System, I want to automatically handle expired invitations so that old invitations don't remain valid indefinitely.

**Why this priority**: Expired invitations should be cleaned up automatically to maintain security and prevent confusion.

**Independent Test**: Create invitation with short expiry, wait for expiry, verify acceptance fails with `INVITATION_EXPIRED`.

**Acceptance Scenarios**:

1. **Scenario**: System marks expired invitations

   * **Given** invitations exist with `expires_at` in the past
   * **When** system cleanup process runs
   * **Then** expired invitations are marked with `status = EXPIRED`

2. **Scenario**: User attempts to accept expired invitation

   * **Given** invitation token exists but has expired
   * **When** user tries to accept
   * **Then** system rejects with 400 (`INVITATION_EXPIRED`)

3. **Scenario**: Admin views expired invitations

   * **Given** group has expired invitations
   * **When** admin lists invitations
   * **Then** expired invitations are shown with `status = EXPIRED`

---

### Edge Cases

* Admin invites user who gets deleted before acceptance
* Invitation created for email that matches multiple users
* User accepts invitation after admin cancels it (race condition)
* Multiple invitations for same user (should prevent or return existing)
* Invitation token leaked/shared - single use enforcement
* Very long expiry times (years in future)
* System clock changes affecting expiry calculations

## Requirements *(mandatory)*

### Functional Requirements

* **FR-I-001**: System MUST allow Group Admins to create invitations for users by user ID, username, or email address.
* **FR-I-002**: System MUST restrict invitation creation to Group Admins and System Admins.
* **FR-I-003**: System MUST generate UUID tokens for invitations (simple, debuggable, sufficient entropy).
* **FR-I-004**: System MUST prevent inviting users who are already group members.
* **FR-I-005**: System MUST support optional expiry dates for invitations.
* **FR-I-006**: System MUST allow admins to list pending invitations for their groups.
* **FR-I-007**: System MUST allow admins to cancel pending invitations.
* **FR-I-008**: System MUST prevent cancellation of already-processed invitations.
* **FR-I-009**: System MUST automatically mark expired invitations as `EXPIRED`.
* **FR-I-010**: System MUST make invitation tokens single-use (consumed on acceptance).
* **FR-I-011**: System MUST record audit logs for critical invitation operations (create, cancel, not routine queries).
* **FR-I-012**: System MUST generate invitation tokens using standard UUID generation.
* **FR-I-013**: System MUST handle concurrent invitation operations safely.
* **FR-I-014**: System MUST ensure only Group Admins and System Admins can create invitations.
* **FR-I-015**: System MUST make invitations scoped to single group and single invitee.
* **FR-I-016**: System MUST ensure invitation tokens are single-use and become invalid after acceptance.
* **FR-I-017**: System MUST ensure expired invitations cannot be accepted but remain for audit.
* **FR-I-018**: System MUST ensure cancelled invitations cannot be accepted and tokens become invalid.
* **FR-I-019**: System MUST allow email-based invitations to be claimed by any user with matching email.
* **FR-I-020**: System MUST resolve usernames to user IDs when creating invitations by username.

### Key Entities *(include if feature involves data)*

* **GroupInvitation**
  * **Description**: Invitation created by admin for user or email to join group.
  * **Core attributes**:
    * `id` (UUID)
    * `group_id` (UUID)
    * `invitee_user_id` (UUID, optional if invite by email or username)
    * `invitee_username` (string, optional, resolved to user_id)
    * `invitee_email` (string, optional)
    * `token` (UUID, simple and debuggable)
    * `created_by` (user id)
    * `created_at` (timestamp)
    * `expires_at` (timestamp, optional)
    * `status` (enum: `PENDING`, `ACCEPTED`, `REJECTED`, `EXPIRED`, `CANCELLED`)
    * `metadata` (JSON: reason, notes)

* **InvitationToken**
  * **Description**: Simple UUID token for invitation acceptance.
  * **Properties**:
    * Standard UUID format
    * URL-safe
    * Single-use (invalidated on acceptance)
    * Stored directly in database (no hashing needed)

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-I-001**: Group Admins can create invitations and receive secure tokens with 201 Created.
* **SC-I-002**: Non-admins attempting to create invitations receive 403 Forbidden.
* **SC-I-003**: Invitations for existing members are rejected with 409 Already Member.
* **SC-I-004**: Admins can list and filter pending invitations for their groups.
* **SC-I-005**: Admins can cancel pending invitations, making tokens invalid.
* **SC-I-006**: Expired invitations are automatically marked and cannot be accepted.
* **SC-I-007**: Invitation tokens are simple UUIDs that can be easily debugged and validated.
* **SC-I-008**: All invitation operations are recorded in audit logs.

## Example API (informational, optional)

**Create Invitation by User ID** — `POST /api/groups/{groupId}/invitations`

```json
{
  "invitee_user_id": "user-uuid-123",
  "expires_at": "2026-01-15T00:00:00Z",
  "metadata": {
    "reason": "Course enrollment",
    "notes": "Student from previous semester"
  }
}
```

**Create Invitation by Username** — `POST /api/groups/{groupId}/invitations`

```json
{
  "invitee_username": "student_john",
  "expires_at": "2026-01-15T00:00:00Z",
  "metadata": {
    "reason": "Course enrollment"
  }
}
```

**Create Email Invitation** — `POST /api/groups/{groupId}/invitations`

```json
{
  "invitee_email": "student@university.edu",
  "expires_at": "2026-01-15T00:00:00Z"
}
```

**Success Response** (201 Created)
```json
{
  "id": "invite-uuid-456",
  "group_id": "group-uuid-789",
  "token": "550e8400-e29b-41d4-a716-446655440000",
  "invitee_user_id": "user-uuid-123",
  "status": "PENDING",
  "created_by": "admin-uuid-001",
  "created_at": "2025-12-28T12:00:00Z",
  "expires_at": "2026-01-15T00:00:00Z"
}
```

**Create Email Invitation** — `POST /api/groups/{groupId}/invitations`

```json
{
  "invitee_email": "student@university.edu",
  "expires_at": "2026-01-15T00:00:00Z"
}
```

**List Invitations** — `GET /api/groups/{groupId}/invitations`

**Success Response** (200 OK)
```json
{
  "invitations": [
    {
      "id": "invite-uuid-456",
      "invitee_user_id": "user-uuid-123",
      "invitee_email": null,
      "status": "PENDING",
      "created_at": "2025-12-28T12:00:00Z",
      "expires_at": "2026-01-15T00:00:00Z"
    },
    {
      "id": "invite-uuid-789",
      "invitee_user_id": null,
      "invitee_email": "student@university.edu",
      "status": "EXPIRED",
      "created_at": "2025-11-01T10:00:00Z",
      "expires_at": "2025-12-01T00:00:00Z"
    }
  ]
}
```

**Cancel Invitation** — `DELETE /api/groups/{groupId}/invitations/{invitationId}`

**Success Response** (204 No Content)

**Error Responses**
* `403 INSUFFICIENT_PERMISSIONS` — non-admin attempting operation
* `404 USER_NOT_FOUND` — invalid username provided
* `409 ALREADY_MEMBER` — inviting existing member
* `400 INVITATION_ALREADY_PROCESSED` — trying to cancel accepted/rejected invitation
* `400 INVITATION_EXPIRED` — trying to accept expired invitation
* `404 INVITATION_NOT_FOUND` — invalid invitation ID or token

## Notes / Implementation hints

* Store invitation tokens as UUIDs directly in database (no hashing needed)
* Use standard UUID validation for token verification
* Consider rate limiting invitation creation (e.g., max 50 invitations per admin per day)
* Email-based invitations should be claimed automatically when user with matching email logs in
* Username-based invitations resolve to user ID at creation time
* Provide admin UI to bulk-invite users from CSV or user list
* Consider notification system to alert invitees via email
* Cleanup job should run periodically to mark expired invitations
* Token format: standard UUID (e.g., 550e8400-e29b-41d4-a716-446655440000)
* Support invitation templates with pre-filled reason/message for common use cases

---