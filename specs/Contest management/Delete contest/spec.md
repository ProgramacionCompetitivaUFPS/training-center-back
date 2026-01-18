# Feature Specification: Delete Contest

**Created**: 2026-01-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Lead deletes contest in their group (Priority: P1)

As a Lead of a group, I want to delete a contest from my group so that I can remove contests that are no longer needed or were created by mistake.

**Why this priority**: Contest deletion is essential for managing contests and cleaning up unwanted competitions. Leads need the ability to remove contests from their groups.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /api/groups/{groupId}/contests/{contestId}` endpoint with valid Lead authentication, validating that the contest and related data are deleted correctly.

**Acceptance Scenarios**:

1. **Scenario**: Successful contest deletion - SCHEDULED contest
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations
   * **And** deletes the NoSQL collection `contest_{contestId}_standings`
   * **And** deletes the final snapshot collection `contest_{contestId}_standings_final` (if exists)
   * **And** sets `contest_id` to `null` for all submissions to problems in this contest
   * **And** returns 204 No Content

2. **Scenario**: Successful contest deletion - ACTIVE contest
   * **Given** a contest exists in a group with status ACTIVE
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **And** the contest has submissions (some may be PENDING)
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations
   * **And** deletes the NoSQL collection `contest_{contestId}_standings`
   * **And** deletes the final snapshot collection `contest_{contestId}_standings_final` (if exists)
   * **And** sets `contest_id` to `null` for all submissions (including PENDING ones)
   * **And** returns 204 No Content

3. **Scenario**: Successful contest deletion - FINISHED contest
   * **Given** a contest exists in a group with status FINISHED
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations
   * **And** deletes the NoSQL collection `contest_{contestId}_standings`
   * **And** deletes the final snapshot collection `contest_{contestId}_standings_final` (if exists)
   * **And** sets `contest_id` to `null` for all submissions
   * **And** returns 204 No Content

4. **Scenario**: Successful contest deletion - contest with problems and submissions
   * **Given** a contest exists with problems P1, P2, P3
   * **And** the contest has submissions to P1 and P2
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations (P1, P2, P3)
   * **And** problems P1, P2, P3 remain in the system (not deleted)
   * **And** deletes the NoSQL collection `contest_{contestId}_standings`
   * **And** deletes the final snapshot collection `contest_{contestId}_standings_final` (if exists)
   * **And** sets `contest_id` to `null` for all submissions to P1 and P2
   * **And** submissions remain in the system (orphaned, no contest association)
   * **And** returns 204 No Content

5. **Scenario**: Delete fails - contest is locked
   * **Given** a contest exists and is locked (`locked = true`)
   * **And** the authenticated user is a Lead of the group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (CONTEST_LOCKED)
   * **And** the contest is not deleted
   * **And** indicates that the contest must be unlocked first

6. **Scenario**: Delete fails - non-Lead attempts deletion
   * **Given** a contest exists in a group
   * **And** the authenticated user is a Member (not Lead) of the group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)
   * **And** the contest is not deleted

7. **Scenario**: Delete fails - non-member attempts deletion
   * **Given** a contest exists in a group
   * **And** the authenticated user is not a member of the group
   * **And** the authenticated user is not an Admin
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)
   * **And** the contest is not deleted

8. **Scenario**: Delete fails - contest not found
   * **Given** no contest exists with the provided contestId
   * **And** the authenticated user is a Lead of the group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 404 Not Found
   * **And** indicates that the contest does not exist

9. **Scenario**: Delete fails - group not found
   * **Given** no group exists with the provided groupId
   * **And** the authenticated user is authenticated
   * **When** they attempt to delete a contest
   * **Then** the system rejects with 404 Not Found
   * **And** indicates that the group does not exist

---

### User Story 2 – Admin deletes contest in any group (Priority: P1)

As an Admin, I want to delete contests in any group so that I can assist with administrative tasks and cleanup across the platform.

**Why this priority**: Admin override capability is essential for platform management and support. Admins need the ability to delete contests when necessary.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /api/groups/{groupId}/contests/{contestId}` endpoint with Admin authentication on a group where they are not a member.

**Acceptance Scenarios**:

1. **Scenario**: Admin deletes contest in group they don't belong to
   * **Given** a contest exists in a group
   * **And** the authenticated user has Admin role (not a member of the group)
   * **And** the contest is not locked
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** performs all deletion operations (Contest_Problem, NoSQL collection, submissions `contest_id = null`)
   * **And** returns 204 No Content

2. **Scenario**: Admin cannot delete locked contest
   * **Given** a contest exists and is locked (`locked = true`)
   * **And** the authenticated user has Admin role
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (CONTEST_LOCKED)
   * **And** the contest is not deleted
   * **Note**: Even Admins cannot delete locked contests - must be unlocked first

---

### User Story 3 – Lead deletes contest in global group (Priority: P1)

As a Lead of the global group (Admin or assigned Coach), I want to delete contests in the global group so that I can manage public competitions effectively.

**Why this priority**: Global contests need deletion capabilities. Leads of the global group should have full management permissions including deletion.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /api/groups/{globalGroupId}/contests/{contestId}` endpoint with Lead authentication (Admin or Coach who is Lead of global group).

**Acceptance Scenarios**:

1. **Scenario**: Lead (Coach) deletes contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Coach role and is a Lead of the global group
   * **And** the contest is not locked
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** performs all deletion operations
   * **And** returns 204 No Content

2. **Scenario**: Admin (Lead) deletes contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Admin role (automatically Lead of global group)
   * **And** the contest is not locked
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** performs all deletion operations
   * **And** returns 204 No Content

3. **Scenario**: Coach (non-Lead) attempts to delete contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Coach role but is NOT a Lead of the global group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)
   * **And** the contest is not deleted

4. **Scenario**: Contestant attempts to delete contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Contestant role
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)
   * **And** the contest is not deleted

---

### Edge Cases

* Deleting contest with no problems (should succeed).
* Deleting contest with no submissions (should succeed).
* Deleting contest with no registrations (should succeed).
* Deleting contest with no standings (should succeed).
* Deleting contest while submissions are being processed (submissions become orphaned).
* Deleting contest while standings are being calculated (should handle gracefully).
* Concurrent delete requests for the same contest (should be idempotent or handle race conditions).
* Deleting contest when group is being deleted simultaneously.
* Deleting contest that was just created (no data yet).
* Deleting contest with very large number of submissions (performance consideration).

---

## API Contract

### DELETE /api/groups/{groupId}/contests/{contestId}

Delete an existing contest and all associated data.

> **Important**: 
> - For regular groups: only Leads and Admin can delete contests
> - For global group: only Leads of the global group (Admin and assigned Coaches) can delete contests
> - Locked contests cannot be deleted (must be unlocked first)
> - Contest can be deleted in any status (SCHEDULED, ACTIVE, FINISHED)
> - Problems are NOT deleted (they are global entities)
> - Submissions are NOT deleted, but `contest_id` is set to `null` (become orphaned)
> - Contest_Problem associations are deleted
> - NoSQL collection `contest_{contestId}_standings` is deleted
> - Final snapshot `contest_{contestId}_standings_final` is deleted if it exists

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Responses**:

#### 204 No Content
Contest deleted successfully. All associated data has been removed or updated.

(No body)

#### 401 Unauthorized
Authentication failed.

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User doesn't have permission or contest is locked.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only group Leads and Admin can delete contests in this group"
}
```

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only Leads of the global group can delete contests"
}
```

```json
{
  "error": "CONTEST_LOCKED",
  "message": "Contest is locked and cannot be deleted. Please unlock the contest first."
}
```

#### 404 Not Found
Group or contest not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Group not found"
}
```

```json
{
  "error": "NOT_FOUND",
  "message": "Contest not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Contest Deletion**
* **FR-DC-001**: The system MUST allow Leads to delete contests in their groups.
* **FR-DC-002**: The system MUST allow Admin to delete contests in any group (has implicit Lead permissions).
* **FR-DC-003**: The system MUST allow only Leads of the global group to delete contests in the global group (Admin is automatically Lead, and Admin can assign other Coaches as Leads).
* **FR-DC-005**: The system MUST allow deletion of contests in any status (SCHEDULED, ACTIVE, FINISHED).
* **FR-DC-006**: The system MUST prevent deletion of locked contests (must be unlocked first).

**Data Deletion**
* **FR-DC-007**: The system MUST delete the Contest record when deletion is requested.
* **FR-DC-008**: The system MUST delete all Contest_Problem associations (cascade delete).
* **FR-DC-009**: The system MUST delete the NoSQL collection `contest_{contestId}_standings` when contest is deleted.
* **FR-DC-010**: The system MUST delete the final snapshot collection `contest_{contestId}_standings_final` when contest is deleted (if it exists).

**Submissions Handling**
* **FR-DC-011**: The system MUST NOT delete submissions when a contest is deleted.
* **FR-DC-012**: The system MUST set `contest_id` to `null` for all submissions to problems in the deleted contest.
* **FR-DC-013**: The system MUST handle submissions in PENDING state (set `contest_id` to `null` even if still being judged).

**Problems Handling**
* **FR-DC-014**: The system MUST NOT delete problems when a contest is deleted (problems are global entities).
* **FR-DC-015**: Problems remain available for use in other contests.

**Permissions**
* **FR-DC-016**: For regular groups, only Leads and Admin can delete contests.
* **FR-DC-017**: For the global group, only Leads of the global group can delete contests.
* **FR-DC-018**: Members who are not Leads MUST NOT be able to delete contests in any group.
* **FR-DC-019**: Contestants MUST NOT be able to delete contests in any group.
* **FR-DC-020**: Coaches who are not Leads of the global group MUST NOT be able to delete contests in the global group.

**Validation**
* **FR-DC-021**: The system MUST validate that the contest exists before attempting deletion.
* **FR-DC-022**: The system MUST validate that the group exists before attempting deletion.
* **FR-DC-023**: The system MUST validate that the contest is not locked before allowing deletion.
* **FR-DC-024**: The system MUST validate user permissions before allowing deletion.

**Response**
* **FR-DC-025**: The system MUST return 204 No Content on successful deletion.
* **FR-DC-026**: The system MUST return appropriate error codes for validation and authorization failures.

### Key Entities

* **Contest**: Represents a programming competition.
  * `id` (string, UUID, PK)
  * Other attributes as defined in Create Contest spec
  * **Deletion**: Hard delete - record is permanently removed

* **Contest_Problem**: Links problems to contests with ordering.
  * `id` (string, UUID, PK)
  * `contest_id` (string, UUID, FK to Contest)
  * `problem_id` (string, UUID, FK to Problem)
  * `order` (integer)
  * **Deletion**: Cascade delete when contest is deleted

* **Standing/Register Collection**: NoSQL collection containing registration and standing data.
  * Collection name: `contest_{contestId}_standings`
  * Document structure: See Register to Contest spec for full document schema
  * **Deletion**: Entire collection deleted when contest is deleted
  * **Snapshot**: Final snapshot created in `contest_{contestId}_standings_final` when contest ends (deleted when contest is deleted)

* **Submission**: Represents a contestant's solution attempt.
  * `id` (string, UUID, PK)
  * `contest_id` (string, UUID, FK to Contest, nullable)
  * `problem_id` (string, UUID, FK to Problem)
  * `contestant_id` (string, UUID, FK to User)
  * Other attributes as defined in Submission spec
  * **Deletion**: NOT deleted, but `contest_id` is set to `null` when contest is deleted

* **Problem**: Represents a programming problem.
  * `id` (string, UUID, PK)
  * Other attributes as defined in Problem spec
  * **Deletion**: NOT deleted (problems are global entities)

> **Deletion Behavior Summary**:
> * **Deleted**: Contest, Contest_Problem, NoSQL collection `contest_{contestId}_standings`
> * **Deleted**: Final snapshot `contest_{contestId}_standings_final` (if exists)
> * **Not Deleted**: Problem, Submission
> * **Updated**: Submission.`contest_id` → `null` (orphaned submissions)

### Permission Matrix

| Role | Regular Group (as Lead) | Regular Group (as Member) | Global Group (as Lead) | Global Group (as Member) |
|------|------------------------|--------------------------|----------------------|------------------------|
| Admin | ✅ | ✅ (implicit Lead) | ✅ (auto Lead) | N/A |
| Coach | ✅ | ❌ | ✅ | ❌ |
| Contestant | ❌ | ❌ | ❌ | ❌ |

### Deletion Flow

```
DELETE /api/groups/{groupId}/contests/{contestId}
    ↓
Validate permissions (Lead/Admin)
    ↓
Validate contest exists and is not locked
    ↓
Delete Contest record
    ↓
Cascade delete Contest_Problem records
    ↓
Delete NoSQL collection contest_{contestId}_standings
    ↓
Delete final snapshot collection contest_{contestId}_standings_final (if exists)
    ↓
Update Submission records (SET contest_id = NULL WHERE contest_id = contestId)
    ↓
Return 204 No Content
```

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-DC-001**: Leads can delete contests in their groups via `DELETE /api/groups/{groupId}/contests/{contestId}` with HTTP 204.
* **SC-DC-002**: Admin can delete contests in any group with HTTP 204.
* **SC-DC-003**: Leads of the global group can delete contests in global group with HTTP 204.
* **SC-DC-004**: Locked contests cannot be deleted - HTTP 403.
* **SC-DC-005**: Non-Lead non-admin users cannot delete contests in regular groups - HTTP 403.
* **SC-DC-006**: Non-Lead users cannot delete contests in global group - HTTP 403.
* **SC-DC-007**: Contest can be deleted in any status (SCHEDULED, ACTIVE, FINISHED).
* **SC-DC-008**: Contest record is permanently deleted from database.
* **SC-DC-009**: All Contest_Problem associations are deleted.
* **SC-DC-010**: NoSQL collection `contest_{contestId}_standings` is deleted.
* **SC-DC-011**: Final snapshot collection `contest_{contestId}_standings_final` is deleted (if exists).
* **SC-DC-012**: Problems are NOT deleted (remain in system).
* **SC-DC-013**: Submissions are NOT deleted but have `contest_id` set to `null`.
* **SC-DC-014**: Submissions in PENDING state are handled correctly (`contest_id` set to `null`).
* **SC-DC-015**: Non-existent contests return HTTP 404.
* **SC-DC-016**: Non-existent groups return HTTP 404.
* **SC-DC-017**: Unauthorized requests return HTTP 401.
* **SC-DC-018**: Deletion is idempotent (deleting already-deleted contest returns 404).

---

## Optional Notes

* **Idempotency**: Deleting a non-existent contest returns 404, making the operation idempotent.
* **Cascade Deletion**: Database-level cascade constraints may be used for Contest_Problem deletion.
* **Performance**: For contests with many submissions, consider batch updates for setting `contest_id` to `null`.
* **Audit Trail**: Consider logging contest deletions for administrative purposes (future enhancement).
* **Soft Delete**: Current implementation uses hard delete. Soft delete could be considered for future versions to allow recovery.
* **Related specs**:
  * Create Contest: Initial contest creation
  * Update Contest: Modifying contest details
  * View Contest: Contest details and problem list
  * Contest Standings: Standing calculation details

