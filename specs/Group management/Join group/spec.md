# Feature Specification: Join Group

**Created**: 2025-12-28

> **Related Spec**: This spec covers the user perspective for joining groups. For the Lead/Admin perspective (managing requests), see [Manage Join Requests](../Manage%20join%20requests/spec.md).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Request to join a group (Priority: P1)

As a user, I want to request to join a group with REQUEST policy so that leads can review and approve my membership.

**Why this priority**: Request-to-join is the primary mechanism for users to express interest in joining controlled groups. This enables organic growth while maintaining lead control over membership.

**Independent Test**: Authenticated user POST `/api/groups/{g}/requests` for a group with `join_policy = REQUEST`. Verify request record created with `status = PENDING`.

**Acceptance Scenarios**:

1. **Scenario**: User creates join request for REQUEST group

   * **Given** group `g` has `join_policy = REQUEST` and `visibility = VISIBLE`
   * **And** user is authenticated and not already a member
   * **When** user submits join request with optional message
   * **Then** system creates request with `status = PENDING`
   * **And** returns 201 Created with request details

2. **Scenario**: User attempts to request join for INVITE-only group

   * **Given** group has `join_policy = INVITE`
   * **When** user tries to create join request
   * **Then** system rejects with 400 (`INVALID_JOIN_POLICY`)

3. **Scenario**: User attempts to request join for OPEN group

   * **Given** group has `join_policy = OPEN`
   * **When** user tries to create join request
   * **Then** system rejects with 400 (`INVALID_JOIN_POLICY`) - should use direct join instead

4. **Scenario**: User requests to join group they're already member of

   * **Given** user is already a member of the group
   * **When** user tries to create join request
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

5. **Scenario**: User requests to join non-visible group

   * **Given** group has `visibility = NOT_VISIBLE`
   * **When** user tries to create join request
   * **Then** system rejects with 404 (`GROUP_NOT_FOUND`)
   * **Note**: Non-visible groups only allow `INVITE` policy, so requests are not applicable

6. **Scenario**: User already has pending request for the group

   * **Given** user has an existing join request with `status = PENDING`
   * **When** user tries to create another join request
   * **Then** system rejects with 409 (`REQUEST_ALREADY_PENDING`)

---

### User Story 2 - Join open group directly (Priority: P2)

As a user, I want to join an open group immediately so that I can access group content without waiting for approval.

**Why this priority**: Open groups should allow frictionless joining for public communities and courses. This improves user experience for non-restricted groups.

**Independent Test**: Authenticated user POST `/api/groups/{g}/join` for group with `join_policy = OPEN`. Verify immediate membership creation with `role = MEMBER`.

**Acceptance Scenarios**:

1. **Scenario**: User joins open group directly

   * **Given** group has `join_policy = OPEN` and `visibility = VISIBLE`
   * **And** user is authenticated and not already a member
   * **When** user calls join endpoint
   * **Then** membership is created immediately with `role = MEMBER`
   * **And** `joined_at` timestamp is recorded

2. **Scenario**: User attempts direct join on non-open group

   * **Given** group has `join_policy = REQUEST` or `join_policy = INVITE`
   * **When** user tries to join directly
   * **Then** system rejects with 400 (`DIRECT_JOIN_NOT_ALLOWED`)

3. **Scenario**: User joins open group they're already member of

   * **Given** user is already a member
   * **When** user tries to join again
   * **Then** system returns 409 (`ALREADY_MEMBER`)

---

### User Story 3 - Accept invitation (Priority: P2)

As an invited user, I want to accept an invitation using the URL sent to my email so I can join the group that invited me.

**Why this priority**: Invitation acceptance completes the invitation flow and enables consent-based membership for private groups.

**Independent Test**: Use valid JWT token from invitation URL to accept invitation. Verify membership created and invitation record deleted.

**Acceptance Scenarios**:

1. **Scenario**: User accepts valid invitation

   * **Given** user clicks invitation URL with valid JWT token
   * **And** JWT token is not expired (within 3 days)
   * **And** user_id in JWT payload matches authenticated user
   * **When** invitee accepts invitation using token
   * **Then** membership is created with `role = MEMBER`
   * **And** invitation record is deleted from database
   * **And** `joined_at` timestamp is recorded

2. **Scenario**: User accepts expired invitation

   * **Given** JWT token exists but expiry has passed (>3 days old)
   * **When** user tries to accept
   * **Then** system rejects with 400 (`INVITATION_EXPIRED`)

3. **Scenario**: User accepts invalid or tampered token

   * **Given** JWT token is malformed, has invalid signature, or doesn't exist
   * **When** user tries to accept
   * **Then** system rejects with 400 (`INVALID_TOKEN`)

4. **Scenario**: User accepts invitation when already member

   * **Given** user is already a member of the group (race condition)
   * **When** user tries to accept invitation
   * **Then** system returns 409 (`ALREADY_MEMBER`)

5. **Scenario**: User accepts old invitation after new one was sent

   * **Given** lead re-invited user (old invitation record was deleted)
   * **And** user clicks old invitation URL with old JWT token
   * **When** user tries to accept with old token
   * **Then** system rejects with 400 (`INVALID_TOKEN`) - invitation record doesn't exist

---

### User Story 4 - User views own request status (Priority: P3)

As a User, I want to view the status of my join request so that I know if it's still pending, approved, or rejected.

**Why this priority**: Users need feedback on their requests to understand their membership status and whether to take other actions.

**Independent Test**: Authenticated user GET `/api/groups/{g}/requests/me`. Verify user can see their own request status for the group.

**Acceptance Scenarios**:

1. **Scenario**: User views pending request

   * **Given** user has a pending join request for group `g`
   * **When** user queries their request status
   * **Then** system returns request details with `status = PENDING`

2. **Scenario**: User views approved request

   * **Given** user's request was approved
   * **When** user queries their request status
   * **Then** system returns request with `status = APPROVED`

3. **Scenario**: User views rejected request

   * **Given** user's request was rejected
   * **When** user queries their request status
   * **Then** system returns request with `status = REJECTED`

4. **Scenario**: User has no request for group

   * **Given** user never requested to join group `g`
   * **When** user queries their request status
   * **Then** system returns 404 (`REQUEST_NOT_FOUND`)

---

### User Story 5 - User cancels pending request (Priority: P3)

As a User, I want to cancel my pending join request so that I can withdraw my request if I change my mind.

**Why this priority**: Users should have control over their pending requests and be able to withdraw them.

**Independent Test**: Authenticated user DELETE `/api/groups/{g}/requests/me`. Verify request is removed and user can submit new request.

**Acceptance Scenarios**:

1. **Scenario**: User cancels pending request

   * **Given** user has a pending join request
   * **When** user deletes their request
   * **Then** request is removed
   * **And** user can submit a new request later

2. **Scenario**: User tries to cancel already processed request

   * **Given** user's request was already approved or rejected
   * **When** user tries to cancel it
   * **Then** system rejects with 400 (`REQUEST_ALREADY_PROCESSED`)

3. **Scenario**: User tries to cancel non-existent request

   * **Given** user has no request for the group
   * **When** user tries to cancel
   * **Then** system returns 404 (`REQUEST_NOT_FOUND`)

---

### Edge Cases

* User requests to join group that gets deleted before approval → Request orphaned, handle gracefully
* Multiple concurrent join requests from same user → Prevent duplicates with DB constraint
* User accepts invitation after being added directly by lead → Return `ALREADY_MEMBER`
* Join request for group that changes join policy before approval → Still process pending requests
* User tries to join global group (should be automatic, not manual) → Reject with `CANNOT_JOIN_GLOBAL_GROUP`
* Rate limiting on join requests to prevent spam → Max 5 requests per user per hour
* User clicks old invitation URL after lead re-sent new invitation → Return `INVALID_TOKEN`
* JWT token is manually modified or tampered with → Return `INVALID_TOKEN`
* User submits new request after previous was rejected → Allow, but consider cooldown period

## Requirements *(mandatory)*

### Functional Requirements

**Join Request Creation**
* **FR-J-001**: System MUST allow users to create join requests for groups with `join_policy = REQUEST`.
* **FR-J-002**: System MUST prevent duplicate pending requests from same user for same group.
* **FR-J-003**: System MUST allow users to submit new request after previous was rejected (consider cooldown).

**Direct Join (OPEN policy)**
* **FR-J-004**: System MUST allow users to join groups directly when `join_policy = OPEN`.
* **FR-J-005**: System MUST restrict join operations to visible groups (`visibility = VISIBLE`).

**Invitation Acceptance**
* **FR-J-006**: System MUST allow users to accept invitations using valid JWT tokens.
* **FR-J-007**: System MUST validate JWT signature, expiry (3 days), and payload before acceptance.
* **FR-J-008**: System MUST verify user_id in JWT payload matches authenticated user accepting invitation.
* **FR-J-009**: System MUST delete invitation record after successful acceptance (single-use enforcement).

**Membership Creation**
* **FR-J-010**: System MUST prevent duplicate memberships - reject if user already member.
* **FR-J-011**: System MUST create GroupMember records with `role = MEMBER` and `joined_at` timestamp.
* **FR-J-012**: System MUST enforce join policy restrictions - reject inappropriate join methods.

**User Request Management**
* **FR-J-013**: System MUST allow users to view status of their own join request for a group.
* **FR-J-014**: System MUST allow users to cancel their pending join requests.

**General**
* **FR-J-015**: System MUST record audit logs for critical join operations (membership creation).
* **FR-J-016**: System MUST handle concurrent join attempts gracefully using DB constraints.
* **FR-J-017**: System MUST prevent manual joining of global group (membership is automatic).

### Key Entities *(include if feature involves data)*

* **JoinRequest**
  * **Description**: User-created request to join a group with REQUEST policy.
  * **Core attributes**:
    * `id` (UUID)
    * `group_id` (UUID)
    * `requester_user_id` (UUID)
    * `message` (text, optional) - User's message explaining why they want to join
    * `status` (enum: `PENDING`, `APPROVED`, `REJECTED`)
    * `created_at` (timestamp)
  * **Constraints**:
    * Unique constraint on (`group_id`, `requester_user_id`) for pending requests

* **GroupMember** (creation aspect)
  * **Description**: Membership record created when user joins group.
  * **Core attributes**:
    * `group_id` (UUID)
    * `user_id` (UUID)
    * `role` (enum: `MEMBER` for joins, `LEAD` assigned separately)
    * `joined_at` (timestamp)

## Success Criteria *(mandatory)*

### Measurable Outcomes

**Join Request Creation**
* **SC-J-001**: Users can create join requests for REQUEST groups and receive 201 Created response.
* **SC-J-002**: Duplicate pending requests from same user are prevented (409 `REQUEST_ALREADY_PENDING`).

**Direct Join**
* **SC-J-003**: Users can join OPEN groups directly and membership is created immediately.
* **SC-J-004**: Direct join attempts on non-OPEN groups are rejected (400 `DIRECT_JOIN_NOT_ALLOWED`).

**Invitation Acceptance**
* **SC-J-005**: Users can accept valid JWT invitations and membership is created with proper timestamps.
* **SC-J-006**: Expired or tampered JWT tokens are rejected appropriately.

**User Request Management**
* **SC-J-007**: Users can view the status of their own join request for a group.
* **SC-J-008**: Users can cancel their pending requests.

**General**
* **SC-J-009**: Invalid join attempts (wrong policy, already member, etc.) are rejected with appropriate error codes.
* **SC-J-010**: All critical join operations are logged in audit trail with actor and timestamp.
* **SC-J-011**: Concurrent join attempts are handled without creating duplicate memberships.

## API Contract

### POST /api/groups/{groupId}/requests

Create a join request for a group with REQUEST policy.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Request Body**:
```json
{
  "message": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| message | string | No | Optional message explaining why user wants to join |

**Responses**:

#### 201 Created
Join request created successfully.

```json
{
  "id": "string",
  "group_id": "string",
  "status": "PENDING",
  "message": "string",
  "created_at": "string"
}
```

#### 400 Bad Request
Cannot create join request due to policy violation.

```json
{
  "error": "INVALID_JOIN_POLICY",
  "message": "This group does not accept join requests"
}
```

#### 404 Not Found
Group not found or not visible.

```json
{
  "error": "GROUP_NOT_FOUND",
  "message": "Group not found or not visible"
}
```

#### 409 Conflict
User is already a member or has pending request.

```json
{
  "error": "ALREADY_MEMBER",
  "message": "You are already a member of this group"
}
```

```json
{
  "error": "REQUEST_ALREADY_PENDING",
  "message": "You already have a pending request for this group"
}
```

---

### POST /api/groups/{groupId}/join

Join an open group directly without approval.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Responses**:

#### 201 Created
Successfully joined the group.

```json
{
  "group_id": "string",
  "user_id": "string",
  "role": "MEMBER",
  "joined_at": "string"
}
```

#### 400 Bad Request
Cannot join group directly.

```json
{
  "error": "DIRECT_JOIN_NOT_ALLOWED",
  "message": "This group does not allow direct joining"
}
```

#### 404 Not Found
Group not found or not visible.

```json
{
  "error": "GROUP_NOT_FOUND",
  "message": "Group not found or not visible"
}
```

#### 409 Conflict
User is already a member.

```json
{
  "error": "ALREADY_MEMBER",
  "message": "You are already a member of this group"
}
```

---

### POST /api/groups/{groupId}/accept

Accept an invitation using JWT token from email URL.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| token | string | Yes | JWT invitation token from email URL |

**Responses**:

#### 200 OK
Invitation accepted successfully and membership created.

```json
{
  "group_id": "string",
  "user_id": "string",
  "role": "MEMBER",
  "joined_at": "string"
}
```

#### 400 Bad Request
Invitation cannot be accepted due to expiry or invalid token.

```json
{
  "error": "INVITATION_EXPIRED",
  "message": "The invitation has expired (>3 days old)"
}
```

```json
{
  "error": "INVALID_TOKEN",
  "message": "Invalid, tampered, or non-existent invitation token"
}
```

#### 409 Conflict
User is already a member.

```json
{
  "error": "ALREADY_MEMBER",
  "message": "You are already a member of this group"
}
```

---

### GET /api/groups/{groupId}/requests/me

Get the current user's join request status for a group.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Responses**:

#### 200 OK
Request found.

```json
{
  "id": "string",
  "group_id": "string",
  "message": "string",
  "status": "PENDING",
  "created_at": "string"
}
```

#### 404 Not Found
No request found for this user and group.

```json
{
  "error": "REQUEST_NOT_FOUND",
  "message": "You have not requested to join this group"
}
```

---

### DELETE /api/groups/{groupId}/requests/me

Cancel the current user's pending join request.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Responses**:

#### 204 No Content
Request cancelled successfully.

#### 400 Bad Request
Request cannot be cancelled.

```json
{
  "error": "REQUEST_ALREADY_PROCESSED",
  "message": "Cannot cancel a request that has already been processed"
}
```

#### 404 Not Found
No pending request found.

```json
{
  "error": "REQUEST_NOT_FOUND",
  "message": "You have no pending request for this group"
}
```

---

## Notes / Implementation hints

**General Join Flow**
* Join requests should include rate limiting (e.g., max 5 requests per user per hour)
* Use database constraints to prevent duplicate memberships on concurrent requests
* Global group membership should be handled by user creation flow, not join endpoints
* Audit logs should capture the join method (request approval, direct join, invitation) for critical operations only
* Join requests are only valid for groups with `REQUEST` policy
* Direct joining is only valid for groups with `OPEN` policy
* Non-visible groups only support `INVITE` policy, so join requests and direct joins are not applicable

**Invitation Handling**
* Use JWT library to validate invitation tokens - check signature, expiry, and payload
* JWT tokens for invitations have fixed 3-day TTL and contain user_id and group_id in payload
* Verify user_id in JWT payload matches authenticated user accepting invitation
* Delete invitation record after successful acceptance (single-use enforcement)
* Invitation acceptance works regardless of group join policy
* Invitation URL format: `https://training-center.com/groups/{groupId}/accept?token={jwt_token}`
* Handle edge case where user clicks old invitation URL after lead re-sent new invitation (invitation record deleted)

**User Request Management**
* Users can only have one pending request per group (unique constraint)
* Allow users to cancel only PENDING requests (not already processed)
* Consider cooldown period before allowing new request after rejection (e.g., 24 hours)

**Performance Considerations**
* Index on (`group_id`, `requester_user_id`) for duplicate prevention
* Index on (`group_id`, `status`) for efficient request queries

---
