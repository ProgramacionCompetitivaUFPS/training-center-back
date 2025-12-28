3# Feature Specification: Join Group

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Request to join a group (Priority: P1)

As a user, I want to request to join a group with REQUEST policy so that admins can review and approve my membership.

**Why this priority**: Request-to-join is the primary mechanism for users to express interest in joining controlled groups. This enables organic growth while maintaining admin control over membership.

**Independent Test**: Authenticated user POST `/api/groups/{g}/requests` for a group with `join_policy = REQUEST`. Verify request record created with `status = PENDING` and admins can view/approve it.

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

3. **Scenario**: User requests to join group they're already member of

   * **Given** user is already a member of the group
   * **When** user tries to create join request
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

4. **Scenario**: User requests to join non-visible group

   * **Given** group has `visibility = NOT_VISIBLE`
   * **When** user tries to create join request
   * **Then** system rejects with 404 (`GROUP_NOT_FOUND`)
   * **Note**: Non-visible groups only allow `INVITE` policy, so requests are not applicable

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

### User Story 3 - Accept invitation (Priority: P3)

As an invited user, I want to accept an invitation so I can join the group that invited me.

**Why this priority**: Invitation acceptance completes the invitation flow and enables consent-based membership for private groups.

**Independent Test**: Use valid invitation token to accept invitation. Verify membership created and invitation status updated to `ACCEPTED`.

**Acceptance Scenarios**:

1. **Scenario**: User accepts valid invitation

   * **Given** invitation exists with valid token and `status = PENDING`
   * **And** invitation is not expired
   * **When** invitee accepts invitation using token
   * **Then** membership is created with `role = MEMBER`
   * **And** invitation `status` becomes `ACCEPTED`
   * **And** `joined_at` timestamp is recorded

2. **Scenario**: User accepts expired invitation

   * **Given** invitation token exists but `expires_at` has passed
   * **When** user tries to accept
   * **Then** system rejects with 400 (`INVITATION_EXPIRED`)

3. **Scenario**: User accepts invalid token

   * **Given** token is malformed or doesn't exist
   * **When** user tries to accept
   * **Then** system rejects with 404 (`INVITATION_NOT_FOUND`)

4. **Scenario**: User accepts invitation when already member

   * **Given** user is already a member of the group (race condition)
   * **When** user tries to accept invitation
   * **Then** system returns 409 (`ALREADY_MEMBER`)
   * **And** invitation status becomes `CONSUMED`

---

### Edge Cases

* User requests to join group that gets deleted before approval
* Multiple concurrent join requests from same user
* User accepts invitation after being added directly by admin
* Join request for group that changes join policy before approval
* User tries to join global group (should be automatic, not manual)
* Rate limiting on join requests to prevent spam

## Requirements *(mandatory)*

### Functional Requirements

* **FR-J-001**: System MUST allow users to create join requests for groups with `join_policy = REQUEST`.
* **FR-J-002**: System MUST allow users to join groups directly when `join_policy = OPEN`.
* **FR-J-003**: System MUST allow users to accept invitations using valid tokens.
* **FR-J-004**: System MUST restrict join operations to visible groups (`visibility = VISIBLE`) unless user has invitation.
* **FR-J-005**: System MUST prevent duplicate memberships - reject if user already member.
* **FR-J-006**: System MUST create GroupMember records with `role = MEMBER` and `joined_at` timestamp.
* **FR-J-007**: System MUST validate invitation tokens and expiry before acceptance.
* **FR-J-008**: System MUST update invitation status to `ACCEPTED` or `CONSUMED` appropriately.
* **FR-J-009**: System MUST enforce join policy restrictions - reject inappropriate join methods.
* **FR-J-010**: System MUST record audit logs for critical join operations (membership creation, not routine queries).
* **FR-J-011**: System MUST handle concurrent join attempts gracefully using DB constraints.
* **FR-J-012**: System MUST validate invitation tokens using standard UUID validation (no timing attack considerations needed for UUID tokens).
* **FR-J-013**: System MUST ensure users can only join visible groups unless they have valid invitation.
* **FR-J-014**: System MUST prevent manual joining of global group (membership is automatic).

### Key Entities *(include if feature involves data)*

* **JoinRequest**
  * **Description**: User-created request to join a group with REQUEST policy.
  * **Core attributes**:
    * `id` (UUID)
    * `group_id` (UUID)
    * `requester_user_id` (UUID)
    * `message` (text, optional)
    * `status` (enum: `PENDING`, `APPROVED`, `REJECTED`)
    * `created_at`, `handled_by`, `handled_at`

* **GroupMember** (creation aspect)
  * **Description**: Membership record created when user joins group.
  * **Core attributes**:
    * `group_id` (UUID)
    * `user_id` (UUID)
    * `role` (enum: `MEMBER` for joins, `ADMIN` assigned separately)
    * `joined_at` (timestamp)

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-J-001**: Users can create join requests for REQUEST groups and receive 201 Created response.
* **SC-J-002**: Users can join OPEN groups directly and membership is created immediately.
* **SC-J-003**: Users can accept valid invitations and membership is created with proper timestamps.
* **SC-J-004**: Invalid join attempts (wrong policy, already member, etc.) are rejected with appropriate error codes.
* **SC-J-005**: All join operations are logged in audit trail with actor and timestamp.
* **SC-J-006**: Concurrent join attempts are handled without creating duplicate memberships.

## Example API (informational, optional)

**Create Join Request** — `POST /api/groups/{groupId}/requests`

```json
{
  "message": "I'm interested in joining this algorithms course"
}
```

**Success Response** (201 Created)
```json
{
  "id": "request-uuid-123",
  "group_id": "group-uuid-456",
  "status": "PENDING",
  "message": "I'm interested in joining this algorithms course",
  "created_at": "2025-12-28T12:00:00Z"
}
```

**Join Open Group** — `POST /api/groups/{groupId}/join`

**Success Response** (201 Created)
```json
{
  "group_id": "group-uuid-456",
  "user_id": "user-uuid-789",
  "role": "MEMBER",
  "joined_at": "2025-12-28T12:00:00Z"
}
```

**Accept Invitation** — `POST /api/groups/{groupId}/invitations/{token}/accept`

**Success Response** (200 OK)
```json
{
  "group_id": "group-uuid-456",
  "user_id": "user-uuid-789",
  "role": "MEMBER",
  "joined_at": "2025-12-28T12:00:00Z"
}
```

**Error Responses**
* `400 INVALID_JOIN_POLICY` — wrong join method for group policy
* `400 INVITATION_EXPIRED` — invitation token expired
* `404 GROUP_NOT_FOUND` — group not visible or doesn't exist
* `404 INVITATION_NOT_FOUND` — invalid invitation token
* `409 ALREADY_MEMBER` — user already member of group

## Notes / Implementation hints

* Join requests should include rate limiting (e.g., max 5 requests per user per hour)
* Invitation tokens are simple UUIDs - standard validation, no timing attack considerations needed
* Use database constraints to prevent duplicate memberships on concurrent requests
* Consider soft-deleting join requests rather than hard deletion for audit purposes
* Global group membership should be handled by user creation flow, not join endpoints
* Audit logs should capture the join method (request approval, direct join, invitation) for critical operations only
* Join requests are only valid for groups with `REQUEST` policy
* Direct joining is only valid for groups with `OPEN` policy
* Invitation acceptance works regardless of group join policy
* Non-visible groups only support `INVITE` policy, so join requests and direct joins are not applicable

---