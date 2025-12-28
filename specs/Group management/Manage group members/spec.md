# Feature Specification: Manage Group Members

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Admin adds member directly (Priority: P1)

As a Group Admin, I want to add users as members directly so that I can quickly populate groups without requiring invitation flows.

**Why this priority**: Direct member addition is essential for administrative control, bulk enrollment, and situations where invitation flows are unnecessary overhead.

**Independent Test**: Authenticated Group Admin POST `/api/groups/{g}/members` with valid user ID. Verify membership created with `role = MEMBER` and `joined_at` timestamp.

**Acceptance Scenarios**:

1. **Scenario**: Admin adds user as member

   * **Given** requesting user is admin of group `g`
   * **And** target user exists and is not already a member
   * **When** admin adds user with `role = MEMBER`
   * **Then** membership is created immediately
   * **And** `joined_at` timestamp is recorded
   * **And** audit log entry is created

2. **Scenario**: Admin adds user as admin

   * **Given** requesting user is admin of group `g`
   * **And** target user is a Coach or System Admin
   * **When** admin adds user with `role = ADMIN`
   * **Then** membership is created with admin privileges
   * **And** target user can now manage the group

3. **Scenario**: Admin attempts to add regular user as admin

   * **Given** target user has role `CONTESTANT` (not Coach/System Admin)
   * **When** admin tries to add them with `role = ADMIN`
   * **Then** system rejects with 400 (`INVALID_ADMIN_ASSIGNMENT`)

4. **Scenario**: Admin adds user who is already member

   * **Given** target user is already a member
   * **When** admin tries to add them again
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

5. **Scenario**: Non-admin attempts to add member

   * **Given** requesting user is not admin of the group
   * **When** they try to add a member
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 2 - Admin removes member (Priority: P2)

As a Group Admin, I want to remove members from the group so that I can manage group composition and remove inactive or inappropriate members.

**Why this priority**: Member removal is necessary for group management, handling policy violations, and maintaining group quality.

**Independent Test**: Admin DELETE `/api/groups/{g}/members/{userId}`. Verify membership deleted and audit log created. Ensure cannot remove last admin.

**Acceptance Scenarios**:

1. **Scenario**: Admin removes regular member

   * **Given** requesting user is admin of group `g`
   * **And** target user is a member (not admin)
   * **When** admin removes the member
   * **Then** membership is deleted
   * **And** audit log records the removal with reason

2. **Scenario**: Admin removes another admin

   * **Given** group has multiple admins
   * **And** requesting user is admin
   * **When** admin removes another admin
   * **Then** membership is deleted
   * **And** remaining admins can still manage group

3. **Scenario**: Admin attempts to remove last admin

   * **Given** only one admin exists in the group
   * **When** admin tries to remove themselves or be removed
   * **Then** system rejects with 400 (`CANNOT_REMOVE_LAST_ADMIN`)

4. **Scenario**: Admin removes member from global group

   * **Given** group is the global default group (`is_default = true`)
   * **When** admin tries to remove any member
   * **Then** system rejects with 400 (`CANNOT_REMOVE_FROM_GLOBAL_GROUP`)

5. **Scenario**: Non-admin attempts to remove member

   * **Given** requesting user is not admin
   * **When** they try to remove a member
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 3 - Change member role (Priority: P3)

As a Group Admin, I want to promote members to admin or demote admins to members so that I can adjust group leadership as needed.

**Why this priority**: Role management allows for delegation of administrative duties and adjustment of group governance structure.

**Independent Test**: Admin PATCH `/api/groups/{g}/members/{userId}` with new role. Verify role updated and constraints enforced (only Coaches can be admins).

**Acceptance Scenarios**:

1. **Scenario**: Admin promotes member to admin

   * **Given** target user is a Coach or System Admin
   * **And** currently has `role = MEMBER`
   * **When** admin promotes them to `role = ADMIN`
   * **Then** role is updated and user gains admin privileges

2. **Scenario**: Admin demotes admin to member

   * **Given** group has multiple admins
   * **And** target user is currently admin
   * **When** admin demotes them to `role = MEMBER`
   * **Then** role is updated and user loses admin privileges

3. **Scenario**: Admin attempts to promote non-coach to admin

   * **Given** target user is `CONTESTANT` (not Coach)
   * **When** admin tries to promote them to admin
   * **Then** system rejects with 400 (`INVALID_ADMIN_ASSIGNMENT`)

4. **Scenario**: Admin attempts to demote last admin

   * **Given** only one admin exists
   * **When** admin tries to demote themselves to member
   * **Then** system rejects with 400 (`CANNOT_REMOVE_LAST_ADMIN`)

---

### User Story 4 - Member leaves group voluntarily (Priority: P2)

As a Group Member, I want to leave a group voluntarily so that I can stop participating in groups I'm no longer interested in.

**Why this priority**: Voluntary leaving respects user autonomy and helps maintain engaged group membership.

**Independent Test**: Member DELETE `/api/groups/{g}/members/me`. Verify membership removed and audit logged. Ensure cannot leave global group.

**Acceptance Scenarios**:

1. **Scenario**: Member leaves regular group

   * **Given** user is member of non-global group
   * **And** user is not the last admin (if admin)
   * **When** user leaves the group
   * **Then** membership is removed
   * **And** user loses access to group content

2. **Scenario**: Member attempts to leave global group

   * **Given** group is the global default group
   * **When** user tries to leave
   * **Then** system rejects with 400 (`CANNOT_LEAVE_GLOBAL_GROUP`)

3. **Scenario**: Last admin attempts to leave

   * **Given** user is the only admin of the group
   * **When** user tries to leave
   * **Then** system rejects with 400 (`CANNOT_LEAVE_AS_LAST_ADMIN`)

4. **Scenario**: Non-member attempts to leave

   * **Given** user is not a member of the group
   * **When** user tries to leave
   * **Then** system rejects with 404 (`NOT_A_MEMBER`)

---

### Edge Cases

* Admin removes member who has active contest submissions (preserve historical data)
* Concurrent role changes by multiple admins
* Member leaves while having pending join requests in other groups
* Admin removes member who was added via invitation (invitation status handling)
* Role change for user whose system role changes (Coach becomes Contestant)
* Bulk member operations (add/remove multiple users)
* Member removal cascading effects on group-specific data

## Requirements *(mandatory)*

### Functional Requirements

* **FR-M-001**: System MUST allow Group Admins to add users as members directly.
* **FR-M-002**: System MUST allow Group Admins to assign `ADMIN` role only to Coaches and System Admins.
* **FR-M-003**: System MUST allow Group Admins to remove members except from global group.
* **FR-M-004**: System MUST prevent removal of the last admin from any group.
* **FR-M-005**: System MUST allow Group Admins to change member roles (promote/demote).
* **FR-M-006**: System MUST allow members to leave groups voluntarily except global group.
* **FR-M-007**: System MUST prevent members from leaving if they are the last admin.
* **FR-M-008**: System MUST record `joined_at` timestamp for direct additions.
* **FR-M-009**: System MUST create audit logs for critical membership changes (add, remove, role changes, not routine queries).
* **FR-M-010**: System MUST preserve historical references when members are removed.
* **FR-M-011**: System MUST validate user existence before membership operations.
* **FR-M-012**: System MUST enforce that global group membership cannot be modified.
* **FR-M-013**: System MUST handle concurrent membership changes safely using transactions.
* **FR-M-014**: System MUST ensure audit logs capture actor, target, action, timestamp, and reason for critical operations only.
* **FR-M-015**: System MUST ensure only Coaches and System Admins can be assigned Group Admin role.
* **FR-M-016**: System MUST ensure global group membership is immutable - users cannot be added/removed manually.
* **FR-M-017**: System MUST ensure every group has at least one admin at all times.
* **FR-M-018**: System MUST preserve historical data when members are removed.
* **FR-M-019**: System MUST ensure member removal does not delete user account, only group membership.
* **FR-M-020**: System MUST ensure role changes are immediate and affect user permissions instantly.

### Key Entities *(include if feature involves data)*

* **GroupMember** (full lifecycle)
  * **Description**: Complete membership record with role management.
  * **Core attributes**:
    * `group_id` (UUID)
    * `user_id` (UUID)
    * `role` (enum: `ADMIN`, `MEMBER`)
    * `joined_at` (timestamp)
    * `added_by` (UUID, who added them)
    * `join_method` (enum: `DIRECT_ADD`, `INVITATION`, `REQUEST_APPROVED`, `OPEN_JOIN`)

* **MembershipAuditLog**
  * **Description**: Audit trail for membership changes.
  * **Core attributes**:
    * `id` (UUID)
    * `group_id` (UUID)
    * `target_user_id` (UUID)
    * `actor_user_id` (UUID)
    * `action` (enum: `ADDED`, `REMOVED`, `ROLE_CHANGED`, `LEFT`)
    * `old_role`, `new_role` (for role changes)
    * `reason` (text, optional)
    * `created_at` (timestamp)

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-M-001**: Group Admins can add members directly and membership is created with proper timestamps.
* **SC-M-002**: Only Coaches/System Admins can be assigned Group Admin role; others are rejected.
* **SC-M-003**: Admins can remove members except when it would leave zero admins.
* **SC-M-004**: Members can leave groups except global group and when they're last admin.
* **SC-M-005**: Role changes are applied immediately and validated against business rules.
* **SC-M-006**: Critical membership operations are recorded in audit logs with complete information (routine queries are not logged).
* **SC-M-007**: Concurrent membership operations don't create inconsistent states.
* **SC-M-008**: Historical data is preserved when members are removed from groups.

## Example API (informational, optional)

**Add Member** — `POST /api/groups/{groupId}/members`

```json
{
  "user_id": "user-uuid-123",
  "role": "MEMBER",
  "reason": "Course enrollment"
}
```

**Success Response** (201 Created)
```json
{
  "group_id": "group-uuid-456",
  "user_id": "user-uuid-123",
  "role": "MEMBER",
  "joined_at": "2025-12-28T12:00:00Z",
  "added_by": "admin-uuid-789",
  "join_method": "DIRECT_ADD"
}
```

**Remove Member** — `DELETE /api/groups/{groupId}/members/{userId}`

Query parameters:
- `reason` (optional): Reason for removal

**Success Response** (204 No Content)

**Change Role** — `PATCH /api/groups/{groupId}/members/{userId}`

```json
{
  "role": "ADMIN",
  "reason": "Promoting to co-instructor"
}
```

**Success Response** (200 OK)
```json
{
  "group_id": "group-uuid-456",
  "user_id": "user-uuid-123",
  "role": "ADMIN",
  "joined_at": "2025-12-20T10:00:00Z",
  "role_changed_at": "2025-12-28T12:00:00Z"
}
```

**Leave Group** — `DELETE /api/groups/{groupId}/members/me`

**Success Response** (204 No Content)

**Error Responses**
* `403 INSUFFICIENT_PERMISSIONS` — non-admin attempting admin operation
* `400 INVALID_ADMIN_ASSIGNMENT` — trying to make non-coach an admin
* `400 CANNOT_REMOVE_LAST_ADMIN` — removing last admin
* `400 CANNOT_LEAVE_GLOBAL_GROUP` — trying to leave global group
* `400 CANNOT_REMOVE_FROM_GLOBAL_GROUP` — admin trying to remove from global
* `409 ALREADY_MEMBER` — adding existing member
* `404 NOT_A_MEMBER` — operating on non-member

## Notes / Implementation hints

* Use database transactions for membership operations to maintain consistency
* Implement soft deletion for audit purposes - keep membership records with `removed_at` timestamp
* Consider batch operations for bulk member addition/removal
* Validate user system roles before assigning group admin role
* Preserve foreign key references in historical data when members are removed
* Rate limit membership operations to prevent abuse
* Consider notification system for membership changes
* Implement member search/filtering for large groups
* Support bulk role changes for administrative efficiency
* Ensure cascade handling when users are deleted/anonymized system-wide

---