# Feature Specification: Invite to Group

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create invitation (Priority: P1)

As a Lead, I want to invite users to join my group by sending them a URL with a secure token via email so that I can proactively bring relevant people into the group.

**Why this priority**: Invitations are the primary mechanism for controlled group growth. Leads need to be able to invite specific users to build their communities and courses. The invitation is sent via email with a URL containing a JWT token that expires in 3 days.

**Independent Test**: Authenticated Lead POST `/api/groups/{g}/invitations` with valid user identifier (user_id, nickname, or email). Verify invitation created with JWT token containing user_id and group_id, and email sent with acceptance URL.

**Acceptance Scenarios**:

1. **Scenario**: Lead creates invitation for existing user by user ID

   * **Given** requesting user is lead of group `g`
   * **And** target user exists and is not already a member
   * **When** lead creates invitation with `invitee_user_id`
   * **Then** system creates JWT token with payload containing user_id, group_id, and 3-day expiry
   * **And** system sends email to user's registered email with URL: `https://training-center.com/groups/{groupId}/accept?token={jwt_token}`
   * **And** if previous invitation exists for this user-group pair, it is deleted
   * **And** returns 201 Created with invitation details (token included)

2. **Scenario**: Lead creates invitation for existing user by nickname

   * **Given** requesting user is lead of group `g`
   * **And** target nickname exists and user is not already a member
   * **When** lead creates invitation with `invitee_nickname`
   * **Then** system resolves nickname to user ID at creation time
   * **And** creates JWT token with resolved user_id in payload
   * **And** system sends email to user's registered email with acceptance URL
   * **And** if previous invitation exists for this user-group pair, it is deleted
   * **And** returns 201 Created with invitation details

3. **Scenario**: Lead creates invitation by email

   * **Given** requesting user is lead of group `g`
   * **When** lead creates invitation with `invitee_email`
   * **Then** system resolves email to user ID (if user exists)
   * **And** creates JWT token with resolved user_id in payload
   * **And** sends acceptance URL to the provided email
   * **And** if previous invitation exists for this user-group pair, it is deleted

4. **Scenario**: Lead re-invites same user (invalidates previous token)

   * **Given** an active invitation already exists for user X to group `g`
   * **When** lead creates new invitation for same user X
   * **Then** system deletes the previous invitation record
   * **And** creates new JWT token with fresh 3-day expiry
   * **And** sends new email with new URL
   * **And** previous token becomes invalid (cannot be used to accept)

5. **Scenario**: Lead invites user who is already member

   * **Given** target user is already a member of the group
   * **When** lead tries to invite them
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

6. **Scenario**: Non-lead attempts to create invitation

   * **Given** requesting user is not lead of the group
   * **When** they try to create invitation
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

7. **Scenario**: Lead invites by invalid nickname

   * **Given** lead provides `invitee_nickname` that doesn't exist
   * **When** invitation creation is attempted
   * **Then** system rejects with 404 (`NICKNAME_NOT_FOUND`)

8. **Scenario**: Lead invites by email that doesn't exist in system

   * **Given** lead provides `invitee_email` that is not registered
   * **When** invitation creation is attempted
   * **Then** system rejects with 404 (`EMAIL_NOT_FOUND`)

---

### User Story 2 - View pending invitations (Priority: P2)

As a Lead, I want to view pending invitations for my group so that I can track who has been invited and is still pending to accept.

**Why this priority**: Leads need visibility into outstanding invitations to understand who they've invited and which invitations are still valid. This helps manage group growth and follow up with invitees.

**Independent Test**: Authenticated Lead GET `/api/groups/{g}/invitations` returns paginated list of non-expired invitations with invitee details and expiration dates.

**Acceptance Scenarios**:

1. **Scenario**: Lead lists pending invitations

   * **Given** requesting user is lead of group `g`
   * **And** group has several non-expired invitations
   * **When** lead requests invitation list
   * **Then** system queries database for invitations with matching group_id
   * **And** system validates JWT token expiry for each invitation
   * **And** system returns only non-expired invitations
   * **And** response includes invitee user data and expires_at (calculated from JWT)

2. **Scenario**: Lead lists invitations with pagination

   * **Given** group has many pending invitations
   * **When** lead requests invitation list with page and size parameters
   * **Then** system returns paginated results
   * **And** response includes total count and pagination metadata

3. **Scenario**: System filters out expired invitations automatically

   * **Given** group has 5 invitations in database
   * **And** 2 of them have expired JWT tokens (>3 days old)
   * **When** lead requests invitation list
   * **Then** system returns only 3 non-expired invitations
   * **And** expired invitations are not shown (they should be cleaned up separately)

4. **Scenario**: Non-lead attempts to view invitations

   * **Given** requesting user is not lead of the group
   * **When** they try to list invitations
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

5. **Scenario**: Admin views invitations

   * **Given** requesting user is Admin (has implicit permissions on all groups)
   * **When** Admin requests invitation list for any group
   * **Then** system returns pending invitations successfully

6. **Scenario**: Group has no pending invitations

   * **Given** group has no invitations or all are expired
   * **When** lead requests invitation list
   * **Then** system returns empty list with 200 OK

---

### Edge Cases

* User attempts to accept invitation after JWT token expires (3 days)
* User clicks old invitation URL after lead re-sends invitation (old token deleted)
* Multiple concurrent invitation creations for same user (race condition)
* User accepts invitation after being added directly by lead
* Invitation sent to email that gets changed before acceptance
* User deletes account before accepting invitation
* JWT token is tampered with or manually modified
* System clock changes affecting JWT expiry validation

## Requirements *(mandatory)*

### Functional Requirements

* **FR-I-001**: System MUST allow Leads to create invitations for users by user ID, nickname, or email address.
* **FR-I-002**: System MUST restrict invitation creation to Leads and Admins.
* **FR-I-003**: System MUST generate JWT tokens for invitations containing user_id, group_id, and expiry in payload.
* **FR-I-004**: System MUST set JWT token expiry to exactly 3 days (72 hours) from creation time.
* **FR-I-005**: System MUST prevent inviting users who are already group members.
* **FR-I-006**: System MUST resolve nicknames to user IDs at invitation creation time.
* **FR-I-007**: System MUST resolve email addresses to user IDs at invitation creation time.
* **FR-I-008**: System MUST reject invitations for non-existent nicknames with 404 (`NICKNAME_NOT_FOUND`).
* **FR-I-009**: System MUST reject invitations for non-existent emails with 404 (`EMAIL_NOT_FOUND`).
* **FR-I-010**: System MUST send invitation email with acceptance URL to user's registered email address.
* **FR-I-011**: System MUST format invitation URL as: `https://training-center.com/groups/{groupId}/accept?token={jwt_token}`.
* **FR-I-012**: System MUST delete any existing invitation record for same user-group pair when creating new invitation.
* **FR-I-013**: System MUST make invitation tokens single-use (consumed on acceptance or deleted on re-invite).
* **FR-I-014**: System MUST record audit logs for invitation creation (not routine queries).
* **FR-I-015**: System MUST handle concurrent invitation operations safely.
* **FR-I-016**: System MUST ensure invitations are scoped to single group and single invitee.
* **FR-I-017**: System MUST validate JWT signature and expiry before accepting invitation.
* **FR-I-018**: System MUST reject expired JWT tokens with 400 (`INVITATION_EXPIRED`).
* **FR-I-019**: System MUST reject tampered or invalid JWT tokens with 400 (`INVALID_TOKEN`).
* **FR-I-020**: System MUST allow Leads to list pending invitations for their groups.
* **FR-I-021**: System MUST restrict invitation listing to Leads and Admins.
* **FR-I-022**: System MUST filter out expired invitations when listing (validate JWT expiry).
* **FR-I-023**: System MUST support pagination for invitation listings.
* **FR-I-024**: System MUST include invitee user data and expiration info in listing response.
* **FR-I-025**: System MUST calculate expires_at from JWT token payload when listing invitations.

### Key Entities *(include if feature involves data)*

* **GroupInvitation**
  * **Description**: Invitation created by lead for user to join group. Record is deleted when new invitation is created for same user-group pair.
  * **Core attributes**:
    * `id` (UUID)
    * `group_id` (UUID)
    * `invitee_user_id` (UUID, always resolved at creation time)
    * `token` (JWT string containing payload: {user_id, group_id, exp})

* **JWT Token Structure**
  * **Description**: JWT token for invitation acceptance.
  * **Payload**:
    * `user_id` (UUID of invitee)
    * `group_id` (UUID of group)
    * `exp` (expiry timestamp, 3 days from creation)
    * `iat` (issued at timestamp)
  * **Properties**:
    * Signed with application secret key
    * Fixed TTL of 72 hours (3 days)
    * URL-safe format
    * Single-use (invitation record deleted on acceptance or re-invite)

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-I-001**: Leads can create invitations and receive JWT tokens with 201 Created.
* **SC-I-002**: Non-leads attempting to create invitations receive 403 Forbidden.
* **SC-I-003**: Invitations for existing members are rejected with 409 Already Member.
* **SC-I-004**: System sends email with invitation URL to user's registered email address.
* **SC-I-005**: JWT tokens expire exactly 3 days after creation.
* **SC-I-006**: Previous invitation records are deleted when new invitation is created for same user-group pair.
* **SC-I-007**: Expired JWT tokens are rejected with 400 Invitation Expired.
* **SC-I-008**: Invalid or tampered JWT tokens are rejected with 400 Invalid Token.
* **SC-I-009**: All invitation creations are recorded in audit logs.
* **SC-I-010**: Leads can list pending invitations and receive paginated results with invitee data.
* **SC-I-011**: Non-leads attempting to list invitations receive 403 Forbidden.
* **SC-I-012**: Invitation listings only include non-expired invitations (JWT expiry validated).
* **SC-I-013**: Admins can list invitations for any group.

## API Contract

### POST /api/groups/{groupId}/invitations

Create an invitation for a user to join the group. Sends email with invitation URL containing JWT token.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Request Body**:
```json
{
  "invitee_user_id": "string",
  "invitee_nickname": "string",
  "invitee_email": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| invitee_user_id | string | No | User ID of the invitee (mutually exclusive with nickname/email) |
| invitee_nickname | string | No | Nickname of the invitee (mutually exclusive with user_id/email) - resolved to user_id at creation |
| invitee_email | string | No | Email of the invitee (mutually exclusive with user_id/nickname) - resolved to user_id at creation |

**Responses**:

#### 201 Created
Invitation created successfully and email sent.

```json
{
  "id": "string",
  "group_id": "string",
  "invitee_user_id": "string",
  "invitation_url": "https://training-center.com/groups/{groupId}/accept?token={jwt_token}",
  "expires_at": "string"
}
```

#### 403 Forbidden
User does not have permission to create invitations.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can create invitations"
}
```

#### 404 Not Found
Nickname or email not found.

```json
{
  "error": "NICKNAME_NOT_FOUND",
  "message": "The specified nickname does not exist"
}
```

```json
{
  "error": "EMAIL_NOT_FOUND",
  "message": "The specified email is not registered"
}
```

#### 409 Conflict
User is already a member.

```json
{
  "error": "ALREADY_MEMBER",
  "message": "User is already a member of this group"
}
```

---

### GET /api/groups/{groupId}/invitations

List all pending (non-expired) invitations for the group.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| page | integer | No | Page number for pagination (default: 1) |
| size | integer | No | Number of items per page (default: 20, max: 100) |

**Responses**:

#### 200 OK
Invitations retrieved successfully.

```json
{
  "invitations": [
    {
      "id": "string",
      "group_id": "string",
      "invitee": {
        "user_id": "string",
        "nickname": "string",
        "email": "string",
        "full_name": "string"
      },
      "expires_at": "string"
    }
  ],
  "pagination": {
    "page": 1,
    "size": 20,
    "total_items": 45,
    "total_pages": 3
  }
}
```

#### 403 Forbidden
User does not have permission to view invitations.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can view invitations"
}
```

## Notes / Implementation hints

* Use JWT library (e.g., PyJWT, jsonwebtoken) to generate and verify tokens
* JWT payload MUST include: `user_id`, `group_id`, `exp` (3 days from creation), `iat`
* Sign JWT tokens with application secret key (environment variable)
* TTL is fixed at 72 hours (3 days) - no custom expiry allowed
* Delete existing invitation record for same user-group pair before creating new one (prevents token accumulation)
* Email service should send invitation with URL format: `https://training-center.com/groups/{groupId}/accept?token={jwt_token}`
* Email template should include group name, inviter name, and clear call-to-action button
* Resolve nicknames and emails to user_id at invitation creation time - reject if not found
* Consider rate limiting invitation creation (e.g., max 50 invitations per lead per day)
* JWT validation on acceptance should check: signature, expiry, user_id matches authenticated user
* On successful acceptance, delete the invitation record (single-use enforcement)
* Audit logs should capture: invitation creation (who invited whom to which group), acceptance events
* URL should be compatible with frontend routing and mobile deep linking
* Consider notification preferences - user may opt out of invitation emails
* Handle edge case: user changes email between invitation and acceptance
* JWT tokens are inherently tamper-proof due to signature verification

**For listing invitations**:
* Query database for invitations with matching `group_id`
* For each invitation, decode JWT token (without full validation) to extract `exp` field
* Filter out invitations where `exp` < current timestamp
* Join with User table to get invitee details (nickname, email, full_name)
* Implement pagination to handle large numbers of invitations
* Consider caching decoded JWT expiry times to avoid repeated decoding
* Expired invitations in DB should be cleaned up periodically by background job (optional optimization)
* Admin check: verify if requesting user has Admin role (system-level), which grants implicit permissions on all groups without requiring membership

---