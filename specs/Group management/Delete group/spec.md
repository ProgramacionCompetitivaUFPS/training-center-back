# Feature Specification: Delete Group

**Created**: 2026-01-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Lead deletes group (Priority: P1)

As a Lead of a group, I want to delete the group so that the group and all its associated content are permanently removed from the system.

**Why this priority**: Group deletion is a critical administrative operation that requires proper safeguards to prevent accidental data loss.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /api/groups/{groupId}` endpoint with valid Lead authentication and correct confirmation, validating that the group and its associated content are deleted.

**Acceptance Scenarios**:

1. **Scenario**: Successful group deletion with correct confirmation
   * **Given** a group exists with id `g1`
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a delete request with `confirmationName` matching the group's exact name
   * **Then** the system deletes the group and all associated content
   * **And** submissions from the group's contests have their `contest_id` set to `NULL`
   * **And** returns 200 OK with deletion summary

2. **Scenario**: Deletion fails - incorrect confirmation name
   * **Given** a group exists with name "Training Camp 2025"
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a delete request with `confirmationName = "Training Camp"`
   * **Then** the system rejects with 400 Bad Request (CONFIRMATION_MISMATCH)

3. **Scenario**: Deletion fails - missing confirmation name
   * **Given** a group exists
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a delete request without `confirmationName`
   * **Then** the system rejects with 400 Bad Request (CONFIRMATION_REQUIRED)

4. **Scenario**: Lead deletes group with active contest
   * **Given** a group has a contest that is currently active (between startTime and endTime)
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a valid delete request
   * **Then** the system deletes the group including the active contest
   * **And** all submissions from that contest have their `contest_id` set to `NULL`

5. **Scenario**: Non-lead attempts to delete group
   * **Given** a group exists
   * **And** the authenticated user is a Member (not Lead) of the group
   * **When** they attempt to delete the group
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

6. **Scenario**: Non-member attempts to delete group
   * **Given** a group exists
   * **And** the authenticated user is not a member of the group
   * **When** they attempt to delete the group
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

---

### User Story 2 – Admin deletes any group (Priority: P1)

As an Admin, I want to delete any group in the system so that I can perform administrative cleanup or handle problematic groups.

**Why this priority**: Admins need full control over the system for governance and emergency situations.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /api/groups/{groupId}` endpoint with Admin authentication and correct confirmation, validating that any non-global group can be deleted.

**Acceptance Scenarios**:

1. **Scenario**: Admin deletes group they don't belong to
   * **Given** a group exists
   * **And** the authenticated user has Admin role (not a member of the group)
   * **When** they submit a valid delete request with correct `confirmationName`
   * **Then** the system deletes the group and all associated content
   * **And** returns 200 OK with deletion summary

2. **Scenario**: Admin attempts to delete global group
   * **Given** the global group exists (marked with `is_default = true`)
   * **And** the authenticated user has Admin role
   * **When** they attempt to delete the global group
   * **Then** the system rejects with 403 Forbidden (CANNOT_DELETE_GLOBAL_GROUP)

---

### User Story 3 – Contestant attempts to delete group (Priority: P3)

As a Contestant, I should not be able to delete any group regardless of my membership status.

**Why this priority**: Security validation to ensure only authorized roles can perform destructive operations.

**Independent Test**: This user story can be tested independently by attempting deletion with Contestant role, validating rejection.

**Acceptance Scenarios**:

1. **Scenario**: Contestant attempts to delete group
   * **Given** a group exists
   * **And** the authenticated user has Contestant role (even if they are a member)
   * **When** they attempt to delete the group
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

---

### Edge Cases

* Group with no contests or materials (simple deletion)
* Group with hundreds of contests and thousands of submissions (performance consideration)
* Concurrent deletion requests for the same group (idempotency)
* Deletion while a contest is being created (race condition)
* Group name with special characters requiring exact match for confirmation
* Network interruption during deletion of large group (transaction rollback)

---

## API Contract

### DELETE /api/groups/{groupId}

Delete a group and all its associated content permanently.

> **Important**: This is a destructive operation that cannot be undone. The user must provide the exact group name as confirmation. Only Leads of the group or Admins can delete groups. The global group cannot be deleted.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group to delete |

**Request Body**:

```json
{
  "confirmationName": "Training Camp 2025"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| confirmationName | string | Yes | Must exactly match the group's name to confirm deletion |

**Responses**:

#### 200 OK
Group deleted successfully.

```json
{
  "message": "Group deleted successfully",
  "deletedGroup": {
    "id": "a1b2c3d4-e5f6-7890-1234-567890123456",
    "name": "Training Camp 2025"
  },
  "deletionSummary": {
    "contestsDeleted": 5,
    "materialsDeleted": 12,
    "standingCollectionsDeleted": 10,
    "submissionsOrphaned": 1250,
    "membersRemoved": 45
  }
}
```

#### 400 Bad Request
Confirmation name missing or doesn't match.

```json
{
  "error": "CONFIRMATION_REQUIRED",
  "message": "Confirmation name is required to delete a group"
}
```

```json
{
  "error": "CONFIRMATION_MISMATCH",
  "message": "Confirmation name does not match the group name. Please enter the exact group name to confirm deletion.",
  "expected": "Training Camp 2025"
}
```

#### 401 Unauthorized
Authentication failed.

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User doesn't have permission or attempting to delete global group.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only group Leads or Admins can delete groups"
}
```

```json
{
  "error": "CANNOT_DELETE_GLOBAL_GROUP",
  "message": "The global group cannot be deleted"
}
```

#### 404 Not Found
Group not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Group not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Deletion Authorization**
* **FR-DG-001**: The system MUST only allow group Leads or Admins to delete groups.
* **FR-DG-002**: The system MUST NOT allow the global group (`is_default = true`) to be deleted.
* **FR-DG-003**: Admin MUST be able to delete any non-global group without being a member.
* **FR-DG-004**: Members who are not Leads MUST NOT be able to delete groups.
* **FR-DG-005**: Users with Contestant role MUST NOT be able to delete any group.

**Confirmation Mechanism**
* **FR-DG-006**: The system MUST require `confirmationName` in the request body.
* **FR-DG-007**: The `confirmationName` MUST exactly match the group's name (case-sensitive).
* **FR-DG-008**: The system MUST reject deletion if `confirmationName` is missing or doesn't match.

**Data Deletion**
* **FR-DG-009**: The system MUST perform hard delete on the group entity.
* **FR-DG-010**: The system MUST delete all contests associated with the group (hard delete).
* **FR-DG-011**: The system MUST delete all materials associated with the group (hard delete).
* **FR-DG-012**: The system MUST delete all standings associated with the group's contests (hard delete).
* **FR-DG-012.1**: For each deleted contest, the system MUST delete the NoSQL collection `contest_{contestId}_standings` (active standings).
* **FR-DG-012.2**: For each deleted contest, the system MUST delete the NoSQL collection `contest_{contestId}_standings_final` (final snapshot) if it exists.
* **FR-DG-013**: The system MUST delete all group memberships (`GroupMember` records).
* **FR-DG-014**: The system MUST delete all pending join requests for the group.
* **FR-DG-015**: The system MUST delete all pending invitations for the group.

**Submission Preservation**
* **FR-DG-016**: The system MUST preserve all submissions from the group's contests.
* **FR-DG-017**: The system MUST set `contest_id = NULL` for all submissions from deleted contests.
* **FR-DG-018**: Orphaned submissions MUST remain in the user's submission history.

**Active Contest Handling**
* **FR-DG-019**: The system MUST allow deletion of groups with active contests.
* **FR-DG-020**: Active contests MUST be deleted along with the group (no special handling).

**Response**
* **FR-DG-021**: The system MUST return a deletion summary with counts of deleted entities.
* **FR-DG-022**: The system MUST NOT return internal IDs except for the deleted group ID.

**Atomicity**
* **FR-DG-023**: The entire deletion operation MUST be atomic (all-or-nothing transaction).
* **FR-DG-024**: If any part of the deletion fails, the system MUST rollback all changes.

### Key Entities Affected

* **Group**: Hard deleted
  * All attributes removed from database
  
* **Contest**: Hard deleted (cascade from Group)
  * `contest.group_id` references deleted group
  
* **Material**: Hard deleted (cascade from Group)
  * `material.group_id` references deleted group
  
* **Standing (NoSQL Collections)**: Hard deleted (cascade from Contest)
  * For each contest: `contest_{contestId}_standings` collection deleted
  * For each contest: `contest_{contestId}_standings_final` collection deleted (if exists)
  * All participant registration and standing documents removed
  
* **Submission**: Preserved but orphaned
  * `submission.contest_id` set to `NULL`
  * `submission.problem_id` remains valid (problems are global)
  * `submission.contestant_id` remains valid
  
* **GroupMember**: Hard deleted
  * All membership records for the group removed
  
* **JoinRequest**: Hard deleted
  * All pending requests for the group removed
  
* **GroupInvitation**: Hard deleted
  * All pending invitations for the group removed

### Permission Matrix

| Role | Can Delete Own Group | Can Delete Any Group | Can Delete Global Group |
|------|---------------------|---------------------|------------------------|
| Admin | ✅ | ✅ | ❌ |
| Coach (Lead) | ✅ | ❌ | ❌ |
| Coach (Member) | ❌ | ❌ | ❌ |
| Contestant | ❌ | ❌ | ❌ |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-DG-001**: Leads can delete their groups via `DELETE /api/groups/{groupId}` with HTTP 200.
* **SC-DG-002**: Admins can delete any non-global group via `DELETE /api/groups/{groupId}` with HTTP 200.
* **SC-DG-003**: Deletion requires exact group name confirmation (case-sensitive).
* **SC-DG-004**: Incorrect or missing confirmation returns HTTP 400.
* **SC-DG-005**: Attempting to delete global group returns HTTP 403 (CANNOT_DELETE_GLOBAL_GROUP).
* **SC-DG-006**: Non-leads and non-admins receive HTTP 403 (INSUFFICIENT_PERMISSIONS).
* **SC-DG-007**: All contests and materials are deleted with the group.
* **SC-DG-008**: All standings are deleted with the contests (NoSQL collections `contest_{contestId}_standings` and `contest_{contestId}_standings_final` for each contest).
* **SC-DG-009**: Submissions are preserved with `contest_id = NULL`.
* **SC-DG-010**: Deletion summary includes counts of all affected entities.
* **SC-DG-011**: Groups with active contests can be deleted.
* **SC-DG-012**: Deletion is atomic - partial failures result in complete rollback.

---

## Optional Notes

* **Performance**: For groups with large amounts of data, consider implementing deletion in background with status tracking.
* **Audit log**: Group deletion should be logged for compliance (who deleted, when, what was in the group).
* **Soft delete alternative**: Future enhancement could add a "trash" period before permanent deletion.
* **Notification**: Currently no notification to members - could be added as future enhancement.
* **Related specs**:
  * Create Group: Group creation
  * Update Group: Group modification
  * Manage Group Members: Member management before deletion

