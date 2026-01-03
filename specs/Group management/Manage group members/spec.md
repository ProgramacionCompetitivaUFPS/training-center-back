# Feature Specification: Manage Group Members

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Lead adds member directly (Priority: P1)

As a Lead, I want to add users as members directly by providing their nicknames so that I can quickly populate groups without requiring invitation flows.

**Why this priority**: Direct member addition is essential for administrative control, bulk enrollment, and situations where invitation flows are unnecessary overhead.

**Independent Test**: Authenticated Lead POST `/api/groups/{g}/members` with valid nickname. Verify membership created with `role = MEMBER` and `joined_at` timestamp.

**Acceptance Scenarios**:

1. **Scenario**: Lead adds user as member by nickname

   * **Given** requesting user is lead of group `g`
   * **And** target nickname exists and is not already a member
   * **When** lead adds user by nickname with `role = MEMBER`
   * **Then** membership is created immediately
   * **And** `joined_at` timestamp is recorded
   * **And** audit log entry is created

2. **Scenario**: Lead adds user as lead by nickname

   * **Given** requesting user is lead of group `g`
   * **And** target nickname corresponds to a Coach or Admin
   * **When** lead adds user by nickname with `role = LEAD`
   * **Then** membership is created with lead privileges
   * **And** target user can now manage the group

3. **Scenario**: Lead attempts to add regular user as lead by nickname

   * **Given** target nickname corresponds to a user with role `CONTESTANT` (not Coach/Admin)
   * **When** lead tries to add them with `role = LEAD`
   * **Then** system rejects with 400 (`INVALID_LEAD_ASSIGNMENT`)

4. **Scenario**: Lead adds user by nickname who is already member

   * **Given** target nickname is already a member
   * **When** lead tries to add them again
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

5. **Scenario**: Lead attempts to add non-existent nickname

   * **Given** target nickname doesn't exist in the system
   * **When** lead tries to add them
   * **Then** system rejects with 404 (`NICKNAME_NOT_FOUND`)

6. **Scenario**: Non-lead attempts to add member

   * **Given** requesting user is not lead of the group
   * **When** they try to add a member
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 2 - Lead removes member (Priority: P2)

As a Lead, I want to remove members from the group so that I can manage group composition and remove inactive or inappropriate members.

**Why this priority**: Member removal is necessary for group management, handling policy violations, and maintaining group quality.

**Independent Test**: Lead DELETE `/api/groups/{g}/members/{userId}`. Verify membership deleted and audit log created. Ensure cannot remove last lead.

**Acceptance Scenarios**:

1. **Scenario**: Lead removes regular member

   * **Given** requesting user is lead of group `g`
   * **And** target user is a member (not lead)
   * **When** lead removes the member
   * **Then** membership is deleted
   * **And** audit log records the removal with reason

2. **Scenario**: Lead removes another lead

   * **Given** group has multiple leads
   * **And** requesting user is lead
   * **When** lead removes another lead
   * **Then** membership is deleted
   * **And** remaining leads can still manage group

3. **Scenario**: Lead attempts to remove last lead

   * **Given** only one lead exists in the group
   * **When** lead tries to remove themselves or be removed
   * **Then** system rejects with 400 (`CANNOT_REMOVE_LAST_LEAD`)

4. **Scenario**: Lead removes member from global group

   * **Given** group is the global default group (`is_default = true`)
   * **When** lead tries to remove any member
   * **Then** system rejects with 400 (`CANNOT_REMOVE_FROM_GLOBAL_GROUP`)

5. **Scenario**: Non-lead attempts to remove member

   * **Given** requesting user is not lead
   * **When** they try to remove a member
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 3 - Change member role (Priority: P3)

As a Lead, I want to promote members to lead or demote leads to members so that I can adjust group leadership as needed.

**Why this priority**: Role management allows for delegation of administrative duties and adjustment of group governance structure.

**Independent Test**: Lead PATCH `/api/groups/{g}/members/{userId}` with new role. Verify role updated and constraints enforced (only Coaches can be leads).

**Acceptance Scenarios**:

1. **Scenario**: Lead promotes member to lead

   * **Given** target user is a Coach or Admin
   * **And** currently has `role = MEMBER`
   * **When** lead promotes them to `role = LEAD`
   * **Then** role is updated and user gains lead privileges

2. **Scenario**: Lead demotes lead to member

   * **Given** group has multiple leads
   * **And** target user is currently lead
   * **When** lead demotes them to `role = MEMBER`
   * **Then** role is updated and user loses lead privileges

3. **Scenario**: Lead attempts to promote non-coach to lead

   * **Given** target user is `CONTESTANT` (not Coach)
   * **When** lead tries to promote them to lead
   * **Then** system rejects with 400 (`INVALID_LEAD_ASSIGNMENT`)

4. **Scenario**: Lead attempts to demote last lead

   * **Given** only one lead exists
   * **When** lead tries to demote themselves to member
   * **Then** system rejects with 400 (`CANNOT_REMOVE_LAST_LEAD`)

---

### User Story 4 - Member leaves group voluntarily (Priority: P2)

As a Group Member, I want to leave a group voluntarily so that I can stop participating in groups I'm no longer interested in.

**Why this priority**: Voluntary leaving respects user autonomy and helps maintain engaged group membership.

**Independent Test**: Member DELETE `/api/groups/{g}/members/me`. Verify membership removed and audit logged. Ensure cannot leave global group.

**Acceptance Scenarios**:

1. **Scenario**: Member leaves regular group

   * **Given** user is member of non-global group
   * **And** user is not the last lead (if lead)
   * **When** user leaves the group
   * **Then** membership is removed
   * **And** user loses access to group content

2. **Scenario**: Member attempts to leave global group

   * **Given** group is the global default group
   * **When** user tries to leave
   * **Then** system rejects with 400 (`CANNOT_LEAVE_GLOBAL_GROUP`)

3. **Scenario**: Last lead attempts to leave

   * **Given** user is the only lead of the group
   * **When** user tries to leave
   * **Then** system rejects with 400 (`CANNOT_LEAVE_AS_LAST_LEAD`)

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

* **FR-M-001**: System MUST allow Leads to add users as members directly.
* **FR-M-002**: System MUST allow Leads to assign `LEAD` role only to Coaches and Admins.
* **FR-M-003**: System MUST allow Leads to remove members except from global group.
* **FR-M-004**: System MUST prevent removal of the last lead from any group.
* **FR-M-005**: System MUST allow Leads to change member roles (promote/demote).
* **FR-M-006**: System MUST allow members to leave groups voluntarily except global group.
* **FR-M-007**: System MUST prevent members from leaving if they are the last lead.
* **FR-M-008**: System MUST record `joined_at` timestamp for direct additions.
* **FR-M-009**: System MUST create audit logs for critical membership changes (add, remove, role changes, not routine queries).
* **FR-M-010**: System MUST preserve historical references when members are removed.
* **FR-M-011**: System MUST validate nickname existence before membership operations.
* **FR-M-012**: System MUST enforce that global group membership cannot be modified.
* **FR-M-013**: System MUST handle concurrent membership changes safely using transactions.
* **FR-M-014**: System MUST ensure audit logs capture actor, target, action, timestamp, and reason for critical operations only.
* **FR-M-015**: System MUST ensure only Coaches and Admins can be assigned Lead role.
* **FR-M-016**: System MUST ensure global group membership is immutable - users cannot be added/removed manually.
* **FR-M-017**: System MUST ensure every group has at least one lead at all times.
* **FR-M-018**: System MUST preserve historical data when members are removed.
* **FR-M-019**: System MUST ensure member removal does not delete user account, only group membership.
* **FR-M-020**: System MUST ensure role changes are immediate and affect user permissions instantly.

### Key Entities *(include if feature involves data)*

* **GroupMember** (full lifecycle)
  * **Description**: Complete membership record with role management.
  * **Core attributes**:
    * `group_id` (UUID)
    * `user_id` (UUID)
    * `role` (enum: `LEAD`, `MEMBER`)
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

* **SC-M-001**: Leads can add members directly and membership is created with proper timestamps.
* **SC-M-002**: Only Coaches/Admins can be assigned Lead role; others are rejected.
* **SC-M-003**: Leads can remove members except when it would leave zero leads.
* **SC-M-004**: Members can leave groups except global group and when they're last lead.
* **SC-M-005**: Role changes are applied immediately and validated against business rules.
* **SC-M-006**: Critical membership operations are recorded in audit logs with complete information (routine queries are not logged).
* **SC-M-007**: Concurrent membership operations don't create inconsistent states.
* **SC-M-008**: Historical data is preserved when members are removed from groups.

## API Contract

### POST /api/groups/{groupId}/members

Add a user as a member or lead to the group.

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
  "nickname": "string",
  "role": "string",
  "reason": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| nickname | string | Yes | Nickname of the user to add |
| role | string | Yes | Role for the user: `MEMBER` or `LEAD` |
| reason | string | No | Reason for adding the user |

**Responses**:

#### 201 Created
Member added successfully.

```json
{
  "group_id": "string",
  "user_id": "string",
  "nickname": "string",
  "role": "string",
  "joined_at": "string",
  "added_by": "string",
  "join_method": "DIRECT_ADD"
}
```

#### 400 Bad Request
Validation error in the request.

```json
{
  "error": "INVALID_LEAD_ASSIGNMENT",
  "message": "Only Coaches and Admins can be assigned as leads"
}
```

#### 403 Forbidden
User does not have permission to add members.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can add members to the group"
}
```

#### 404 Not Found
Nickname not found.

```json
{
  "error": "NICKNAME_NOT_FOUND",
  "message": "The specified nickname does not exist"
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

### DELETE /api/groups/{groupId}/members/{nickname}

Remove a member from the group.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |
| nickname | string | Yes | Nickname of the user to remove |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| reason | string | No | Reason for removal |

**Responses**:

#### 204 No Content
Member removed successfully.

#### 400 Bad Request
Cannot remove member due to business rule violation.

```json
{
  "error": "CANNOT_REMOVE_LAST_LEAD",
  "message": "Cannot remove the last lead from the group"
}
```

```json
{
  "error": "CANNOT_REMOVE_FROM_GLOBAL_GROUP",
  "message": "Cannot remove members from the global group"
}
```

#### 403 Forbidden
User does not have permission to remove members.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can remove members from the group"
}
```

#### 404 Not Found
Member not found in group.

```json
{
  "error": "NOT_A_MEMBER",
  "message": "User is not a member of this group"
}
```

---

### PATCH /api/groups/{groupId}/members/{nickname}

Change a member's role in the group.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |
| nickname | string | Yes | Nickname of the user |

**Request Body**:
```json
{
  "role": "string",
  "reason": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| role | string | Yes | New role for the user: `MEMBER` or `LEAD` |
| reason | string | No | Reason for the role change |

**Responses**:

#### 200 OK
Role changed successfully.

```json
{
  "group_id": "string",
  "user_id": "string",
  "nickname": "string",
  "role": "string",
  "joined_at": "string",
  "role_changed_at": "string"
}
```

#### 400 Bad Request
Cannot change role due to business rule violation.

```json
{
  "error": "INVALID_LEAD_ASSIGNMENT",
  "message": "Only Coaches and Admins can be assigned as leads"
}
```

```json
{
  "error": "CANNOT_REMOVE_LAST_LEAD",
  "message": "Cannot demote the last lead of the group"
}
```

#### 403 Forbidden
User does not have permission to change roles.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can change member roles"
}
```

#### 404 Not Found
Member not found in group.

```json
{
  "error": "NOT_A_MEMBER",
  "message": "User is not a member of this group"
}
```

---

### DELETE /api/groups/{groupId}/members/me

Leave a group voluntarily.

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
Successfully left the group.

#### 400 Bad Request
Cannot leave group due to business rule violation.

```json
{
  "error": "CANNOT_LEAVE_GLOBAL_GROUP",
  "message": "Cannot leave the global group"
}
```

```json
{
  "error": "CANNOT_LEAVE_AS_LAST_LEAD",
  "message": "Cannot leave as the last lead of the group"
}
```

#### 404 Not Found
User is not a member of the group.

```json
{
  "error": "NOT_A_MEMBER",
  "message": "You are not a member of this group"
}
```

## Notes / Implementation hints

* Use database transactions for membership operations to maintain consistency
* Implement soft deletion for audit purposes - keep membership records with `removed_at` timestamp
* Consider batch operations for bulk member addition/removal
* Validate user system roles before assigning group lead role
* Preserve foreign key references in historical data when members are removed
* Rate limit membership operations to prevent abuse
* Consider notification system for membership changes
* Implement member search/filtering for large groups
* Support bulk role changes for administrative efficiency
* Ensure cascade handling when users are deleted/anonymized system-wide
* Nickname validation should use the unique `nickname` field from User entity
* All member operations use nicknames as identifiers for better user experience and consistency

---