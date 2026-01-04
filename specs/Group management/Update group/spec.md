# Feature Specification: Update Group

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Update group metadata (Priority: P1)

As a Lead, I want to update my group's name and description so that I can keep the information current and relevant.

**Why this priority**: Basic metadata updates are the most common operation and have minimal side effects.

**Independent Test**: Authenticated Lead PATCH `/api/groups/{groupId}` with new name/description. Verify changes persisted and audit logged.

**Acceptance Scenarios**:

1. **Scenario**: Lead updates group description

   * **Given** requesting user is lead of group `g`
   * **When** lead updates only the `description`
   * **Then** description is updated
   * **And** `updated_at` timestamp is set
   * **And** other fields remain unchanged
   * **And** audit log is created

2. **Scenario**: Lead updates group name

   * **Given** requesting user is lead of group `g`
   * **And** new name is unique (case-insensitive)
   * **When** lead updates the `name`
   * **Then** name is updated
   * **And** `updated_at` timestamp is set
   * **And** audit log is created

3. **Scenario**: Lead updates name to existing name

   * **Given** another group already has the desired name
   * **When** lead tries to update name
   * **Then** system rejects with 409 (`NAME_ALREADY_EXISTS`)

4. **Scenario**: Lead updates name to reserved name

   * **Given** lead tries to rename to "Global" or other reserved name
   * **When** update is processed
   * **Then** system rejects with 400 (`RESERVED_NAME`)

5. **Scenario**: Non-lead attempts to update group

   * **Given** requesting user is not lead of the group
   * **When** they try to update
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

6. **Scenario**: Admin updates any group

   * **Given** requesting user is Admin (implicit permissions)
   * **When** Admin updates any group
   * **Then** update succeeds

7. **Scenario**: Lead updates multiple fields at once

   * **Given** requesting user is lead of group `g`
   * **When** lead updates `name` and `description` together
   * **Then** both fields are updated atomically
   * **And** single audit log entry is created

---

### User Story 2 - Update group policies (Priority: P2)

As a Lead, I want to change my group's visibility and join policy so that I can adjust how users discover and join my group.

**Why this priority**: Policy changes are less frequent but important for group lifecycle management.

**Independent Test**: Authenticated Lead PATCH `/api/groups/{groupId}` with new visibility/join_policy. Verify policy constraints enforced and pending requests handled correctly.

**Acceptance Scenarios**:

1. **Scenario**: Lead changes join_policy from REQUEST to OPEN

   * **Given** group has `visibility = VISIBLE` and `join_policy = REQUEST`
   * **And** group has pending join requests
   * **When** lead changes `join_policy` to `OPEN`
   * **Then** policy is updated
   * **And** all pending requests are **approved automatically** (users become members)
   * **And** audit log records the policy change and auto-approved requests

2. **Scenario**: Lead changes join_policy from REQUEST to INVITE

   * **Given** group has `visibility = VISIBLE` and `join_policy = REQUEST`
   * **And** group has pending join requests
   * **When** lead changes `join_policy` to `INVITE`
   * **Then** policy is updated
   * **And** all pending requests are **rejected automatically**
   * **And** audit log records the policy change and auto-rejected requests

3. **Scenario**: Lead changes join_policy from OPEN to REQUEST

   * **Given** group has `visibility = VISIBLE` and `join_policy = OPEN`
   * **When** lead changes `join_policy` to `REQUEST`
   * **Then** policy is updated
   * **And** new users must now request to join (no pending requests affected)

4. **Scenario**: Lead changes visibility from VISIBLE to NOT_VISIBLE

   * **Given** group has `visibility = VISIBLE` and any `join_policy`
   * **And** group may have pending join requests
   * **When** lead changes to `NOT_VISIBLE`
   * **Then** visibility is updated to `NOT_VISIBLE`
   * **And** `join_policy` is **automatically set to `INVITE`**
   * **And** all pending requests are **rejected automatically**
   * **And** audit log records all changes

5. **Scenario**: Lead changes visibility from NOT_VISIBLE to VISIBLE

   * **Given** group has `visibility = NOT_VISIBLE` and `join_policy = INVITE`
   * **When** lead changes to `VISIBLE`
   * **Then** visibility is updated
   * **And** `join_policy` remains `INVITE` (lead can change it separately)

6. **Scenario**: Lead attempts invalid policy combination

   * **Given** lead tries to set `visibility = NOT_VISIBLE` with `join_policy = OPEN`
   * **When** update is processed
   * **Then** system rejects with 400 (`INVALID_POLICY_COMBINATION`)

7. **Scenario**: Lead attempts invalid policy combination (REQUEST)

   * **Given** lead tries to set `visibility = NOT_VISIBLE` with `join_policy = REQUEST`
   * **When** update is processed
   * **Then** system rejects with 400 (`INVALID_POLICY_COMBINATION`)

---

### User Story 3 - Global group restrictions (Priority: P3)

As a System, I want to restrict modifications to the global group so that the system's default behavior is preserved.

**Why this priority**: Edge case protection for the special global group.

**Acceptance Scenarios**:

1. **Scenario**: Attempt to modify global group

   * **Given** group is the global default group (`is_default = true`)
   * **When** anyone (including Admin) tries to update any field
   * **Then** system rejects with 400 (`CANNOT_MODIFY_GLOBAL_GROUP`)

---

### Edge Cases

* Lead updates to same values (no-op) → Accept, update `updated_at`, minimal audit log
* Concurrent updates by multiple leads → Last write wins, validate constraints before commit
* Group name with special characters or very long name → Apply same validation as create
* Lead changes from INVITE to REQUEST/OPEN → No pending requests to handle (invitations are separate)
* Pending invitations when changing policies → Invitations remain valid (they use JWT tokens)

---

## Requirements *(mandatory)*

### Functional Requirements

**Authorization**
* **FR-UG-001**: System MUST allow Leads to update groups they lead.
* **FR-UG-002**: System MUST allow Admins to update any group (implicit permissions).
* **FR-UG-003**: System MUST reject update attempts by non-leads/non-admins with 403.

**Field Updates**
* **FR-UG-004**: System MUST allow updating `name`, `description`, `visibility`, and `join_policy`.
* **FR-UG-005**: System MUST NOT allow updating `id`, `created_by`, `created_at`, `is_default`, `members_count`.
* **FR-UG-006**: System MUST validate `name` uniqueness (case-insensitive) when updating.
* **FR-UG-007**: System MUST reject reserved names when updating.
* **FR-UG-008**: System MUST update `updated_at` timestamp on every successful update.

**Policy Constraints**
* **FR-UG-009**: System MUST enforce valid visibility + join_policy combinations:
  * `VISIBLE` allows `OPEN`, `REQUEST`, or `INVITE`
  * `NOT_VISIBLE` only allows `INVITE`
* **FR-UG-010**: System MUST automatically set `join_policy = INVITE` when `visibility` changes to `NOT_VISIBLE`.
* **FR-UG-011**: System MUST reject attempts to set invalid combinations with 400 (`INVALID_POLICY_COMBINATION`).

**Pending Requests Handling**
* **FR-UG-012**: When `join_policy` changes from `REQUEST` to `OPEN`, system MUST automatically approve all pending requests (create memberships).
* **FR-UG-013**: When `join_policy` changes from `REQUEST` to `INVITE`, system MUST automatically reject all pending requests.
* **FR-UG-014**: When `visibility` changes to `NOT_VISIBLE`, system MUST automatically reject all pending requests.

**Global Group Protection**
* **FR-UG-015**: System MUST reject ALL modifications to the global group (`is_default = true`) with 400 (`CANNOT_MODIFY_GLOBAL_GROUP`).

**Audit**
* **FR-UG-016**: System MUST record audit logs for all group updates including: fields changed, old values, new values, auto-approved/rejected requests count.

---

### Key Entities *(include if feature involves data)*

* **Group** (update aspect)
  * **Modifiable attributes**:
    * `name` (string, unique, case-insensitive)
    * `description` (text, optional)
    * `visibility` (enum: `VISIBLE`, `NOT_VISIBLE`)
    * `join_policy` (enum: `INVITE`, `REQUEST`, `OPEN`)
    * `updated_at` (timestamp) - set automatically on update

  * **Non-modifiable attributes**:
    * `id` (UUID)
    * `created_by` (user id)
    * `created_at` (timestamp)
    * `is_default` (bool)
    * `members_count` (derived)

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-UG-001**: Leads can update name and description successfully.
* **SC-UG-002**: Name uniqueness is enforced on updates (409 for duplicates).
* **SC-UG-003**: Policy combinations are validated (400 for invalid combinations).
* **SC-UG-004**: Non-leads receive 403 when attempting updates.
* **SC-UG-005**: Global group modifications are rejected with 400.
* **SC-UG-006**: When changing REQUEST → OPEN, pending requests are auto-approved.
* **SC-UG-007**: When changing REQUEST → INVITE, pending requests are auto-rejected.
* **SC-UG-008**: When changing to NOT_VISIBLE, join_policy is forced to INVITE and pending requests are auto-rejected.
* **SC-UG-009**: All updates are recorded in audit logs with complete change information.

---

## API Contract

### PATCH /api/groups/{groupId}

Update group information. Only provided fields are updated (partial update).

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead/Admin authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Request Body**:
```json
{
  "name": "string",
  "description": "string",
  "visibility": "string",
  "join_policy": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | No | New name (must be unique, case-insensitive) |
| description | string | No | New description |
| visibility | string | No | New visibility: `VISIBLE` or `NOT_VISIBLE` |
| join_policy | string | No | New join policy: `INVITE`, `REQUEST`, or `OPEN` |

> **Note**: At least one field must be provided. If `visibility` is set to `NOT_VISIBLE`, `join_policy` will be automatically set to `INVITE` regardless of what is provided.

**Responses**:

#### 200 OK
Group updated successfully.

```json
{
  "id": "string",
  "name": "string",
  "description": "string",
  "visibility": "string",
  "join_policy": "string",
  "created_by": "string",
  "created_at": "string",
  "updated_at": "string",
  "members_count": 0,
  "policy_change_effects": {
    "requests_auto_approved": 0,
    "requests_auto_rejected": 0
  }
}
```

> **Note**: `policy_change_effects` is only included when pending requests were affected by the update.

#### 400 Bad Request
Validation error.

```json
{
  "error": "INVALID_POLICY_COMBINATION",
  "message": "NOT_VISIBLE groups can only have INVITE join policy"
}
```

```json
{
  "error": "RESERVED_NAME",
  "message": "The group name is reserved"
}
```

```json
{
  "error": "CANNOT_MODIFY_GLOBAL_GROUP",
  "message": "The global group cannot be modified"
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "At least one field must be provided"
}
```

#### 403 Forbidden
User does not have permission.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can update this group"
}
```

#### 404 Not Found
Group not found.

```json
{
  "error": "GROUP_NOT_FOUND",
  "message": "Group not found"
}
```

#### 409 Conflict
Name already exists.

```json
{
  "error": "NAME_ALREADY_EXISTS",
  "message": "A group with this name already exists"
}
```

---

## Notes / Implementation hints

**General**
* Use PATCH semantics - only update fields that are provided in the request
* Validate all constraints before applying any changes (atomic update)
* Update `updated_at` timestamp on every successful update

**Policy Changes with Pending Requests**
* When handling pending requests, use database transaction to ensure atomicity
* For auto-approval (REQUEST → OPEN): Create GroupMember records for all pending requesters with `role = MEMBER`
* For auto-rejection: Update request status to `REJECTED`
* Record count of affected requests in audit log

**Visibility Change to NOT_VISIBLE**
* This is a compound operation: change visibility + force join_policy + reject pending requests
* Process in order: validate → reject requests → update visibility → update join_policy

**Global Group Check**
* Check `is_default = true` early in the flow to fail fast
* No modifications allowed, including by Admin

**Name Validation**
* Apply same rules as group creation: unique (case-insensitive), not reserved, trimmed whitespace

**Concurrency**
* Use optimistic locking or database constraints to handle concurrent updates
* Last write wins, but all constraint validations must pass

---
