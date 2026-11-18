# Feature Specification: Delete Contest

**Created**: 2026-01-03
**Synced with the implementation**: 2026-11-18

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Lead deletes contest in their group (Priority: P1)

As a Lead of a group, I want to delete a contest from my group so that I can remove contests that are no longer needed or were created by mistake.

**Why this priority**: Contest deletion is essential for managing contests and cleaning up unwanted competitions. Leads need the ability to remove contests from their groups.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /groups/{groupId}/contests/{contestId}` endpoint with valid Lead authentication, validating that the contest and related data are deleted correctly.

**Acceptance Scenarios**:

1. **Scenario**: Successful contest deletion - SCHEDULED contest
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the contest is not locked
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations
   * **And** deletes the contest's registrations and team participations
   * **And** attempts to remove the cached standings of the contest (a failure to remove them does not fail the request)
   * **And** sets `contest_id` to `null` for all submissions of the contest
   * **And** returns 204 No Content

2. **Scenario**: Delete fails - contest is ACTIVE
   * **Given** a contest exists in a group with status ACTIVE
   * **And** the authenticated user is a Lead of the group (or an Admin)
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 409 Conflict (`CONTEST_IS_ACTIVE`)
   * **And** the contest and all its data are left untouched

3. **Scenario**: Delete fails - contest is locked
   * **Given** a contest exists and is locked (`locked = true`)
   * **And** the authenticated user is a Lead of the group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (`CONTEST_LOCKED`)
   * **And** the contest and all its data are left untouched
   * **And** the contest must be unlocked first

4. **Scenario**: Successful contest deletion - FINISHED contest
   * **Given** a contest exists in a group with status FINISHED
   * **And** the contest is not locked
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations, registrations and team participations
   * **And** attempts to remove the cached standings of the contest (a failure to remove them does not fail the request)
   * **And** sets `contest_id` to `null` for all submissions (including postcompetition ones)
   * **And** returns 204 No Content

5. **Scenario**: Successful contest deletion - contest with problems and submissions
   * **Given** a contest with status SCHEDULED or FINISHED exists with problems P1, P2, P3
   * **And** the contest is not locked
   * **And** the contest has submissions to P1 and P2
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** deletes all Contest_Problem associations (P1, P2, P3)
   * **And** problems P1, P2, P3 remain in the system (not deleted)
   * **And** sets `contest_id` to `null` for all submissions to P1 and P2
   * **And** submissions remain in the system (orphaned, no contest association)
   * **And** returns 204 No Content

6. **Scenario**: Delete fails - non-Lead attempts deletion
   * **Given** a contest exists in a group
   * **And** the authenticated user is a Member (not Lead) of the group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)
   * **And** the contest is not deleted

7. **Scenario**: Delete fails - non-member attempts deletion
   * **Given** a contest exists in a group
   * **And** the authenticated user is not a member of the group
   * **And** the authenticated user is not an Admin
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)
   * **And** the contest is not deleted

8. **Scenario**: Delete fails - contest not found
   * **Given** no contest exists with the provided contestId
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 404 Not Found (`CONTEST_NOT_FOUND`)

9. **Scenario**: Delete fails - contest belongs to another group
   * **Given** the contest exists but in a group different from `groupId` in the path
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 404 Not Found (`CONTEST_NOT_FOUND`)
   * **And** the same response is returned when the group does not exist

---

### User Story 2 – Admin deletes contest in any group (Priority: P1)

As an Admin, I want to delete contests in any group so that I can assist with administrative tasks and cleanup across the platform.

**Why this priority**: Admin override capability is essential for platform management and support.

**Independent Test**: Consume the `DELETE /groups/{groupId}/contests/{contestId}` endpoint with Admin authentication on a group where they are not a member.

**Acceptance Scenarios**:

1. **Scenario**: Admin deletes contest in group they don't belong to
   * **Given** a contest with status SCHEDULED or FINISHED exists in a group
   * **And** the contest is not locked
   * **And** the authenticated user has Admin role (not a member of the group)
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** performs all deletion operations (Contest_Problem, registrations, team participations, attempted cached-standings removal, submissions `contest_id = null`)
   * **And** returns 204 No Content

2. **Scenario**: Admin cannot delete a locked contest
   * **Given** a contest exists and is locked (`locked = true`)
   * **And** the authenticated user has Admin role
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (`CONTEST_LOCKED`)
   * **Note**: Even Admins cannot delete locked contests - they must unlock them first

3. **Scenario**: Admin cannot delete an ACTIVE contest
   * **Given** a contest exists with status ACTIVE
   * **And** the authenticated user has Admin role
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 409 Conflict (`CONTEST_IS_ACTIVE`)

---

### User Story 3 – Lead deletes contest in global group (Priority: P1)

As a Lead of the global group (Admin or assigned Coach), I want to delete contests in the global group so that I can manage public competitions effectively.

**Why this priority**: Global contests need deletion capabilities. Leads of the global group should have full management permissions including deletion.

**Independent Test**: Consume the `DELETE /groups/{globalGroupId}/contests/{contestId}` endpoint with Lead authentication (Admin or Coach who is Lead of global group).

**Acceptance Scenarios**:

1. **Scenario**: Lead (Coach) deletes contest in global group
   * **Given** a contest with status SCHEDULED or FINISHED exists in the global group (`is_default = true`)
   * **And** the contest is not locked
   * **And** the authenticated user has Coach role and is a Lead of the global group
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** returns 204 No Content

2. **Scenario**: Admin deletes contest in global group
   * **Given** a contest with status SCHEDULED or FINISHED exists in the global group (`is_default = true`)
   * **And** the contest is not locked
   * **And** the authenticated user has Admin role
   * **When** they submit a delete request
   * **Then** the system deletes the contest
   * **And** returns 204 No Content

3. **Scenario**: Coach (non-Lead) attempts to delete contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Coach role but is NOT a Lead of the global group
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)
   * **And** the contest is not deleted

4. **Scenario**: Contestant attempts to delete contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Contestant role
   * **When** they attempt to delete the contest
   * **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)
   * **And** the contest is not deleted

---

### Edge Cases

* Deleting contest with no problems, submissions or registrations (should succeed).
* Deleting a contest whose standings were never requested (nothing cached; should succeed).
* Deleting a contest that has submissions still being judged (their `contest_id` becomes `null`).
* Concurrent delete requests for the same contest (the second one returns 404).
* The contest is locked after it was loaded but before it is deleted: it is not deleted and the request returns 403 `CONTEST_LOCKED`.
* The contest is locked and then unlocked again within the same instant the delete is refused: nothing is deleted and the request returns 409 `CONTEST_CONFLICT`; the client can retry.
* Deleting a contest that was just created (no data yet).
* Deleting a contest with a very large number of submissions (performance consideration).
* A locked contest that is also ACTIVE is rejected as locked (403) before the active check (409).

---

## API Contract

### DELETE /groups/{groupId}/contests/{contestId}

Delete an existing contest and its associated data.

> **Important**:
> - For regular groups and the global group: only Leads of the group and Admins can delete contests (an Admin needs no membership)
> - Locked contests cannot be deleted (must be unlocked first), not even by an Admin or the owner
> - Contest can be deleted when SCHEDULED or FINISHED; an ACTIVE contest cannot be deleted
> - Problems are NOT deleted (they are global entities)
> - Submissions are NOT deleted, but `contest_id` is set to `null` (become orphaned)
> - Contest_Problem associations, registrations and team participations are deleted (cascade)
> - Removal of the cached standings (`standings:{contestId}`) is attempted on a best-effort basis: if Redis fails the contest is still deleted, the request returns 204 and the cache entry may remain

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
User is not a Lead of the group and is not an Admin, or the contest is locked.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "only leads and admins can delete contests"
}
```

```json
{
  "error": "CONTEST_LOCKED",
  "message": "contest is locked and cannot be deleted; unlock it first"
}
```

#### 404 Not Found
Contest not found, or it does not belong to the group in the path.

```json
{
  "error": "CONTEST_NOT_FOUND",
  "message": "contest not found"
}
```

#### 409 Conflict
The contest is ACTIVE, or it changed while the delete was being refused (retry the request).

```json
{
  "error": "CONTEST_IS_ACTIVE",
  "message": "cannot delete an active contest"
}
```

```json
{
  "error": "CONTEST_CONFLICT",
  "message": "the contest changed while it was being deleted; try again"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Contest Deletion**
* **FR-DC-001**: The system MUST allow Leads to delete contests in their groups.
* **FR-DC-002**: The system MUST allow Admin to delete contests in any group (has implicit Lead permissions).
* **FR-DC-003**: The system MUST allow only Leads of the global group to delete contests in the global group (Admin is automatically Lead, and Admin can assign other Coaches as Leads).
* **FR-DC-005**: The system MUST allow deletion of contests in status SCHEDULED or FINISHED.
* **FR-DC-006**: The system MUST reject the deletion of an ACTIVE contest with 409 `CONTEST_IS_ACTIVE`.
* **FR-DC-006.1**: The system MUST reject the deletion of a locked contest with 403 `CONTEST_LOCKED`, for every user including Admins and the owner. Permissions are checked first, and the lock before the contest status.
* **FR-DC-006.2**: The deletion MUST be atomic with the lock check: the contest is removed only if it is unlocked at the moment of the delete, so a lock set after the contest was loaded still prevents it (403 `CONTEST_LOCKED`). If the delete is refused but the contest turns out to be unlocked, nothing is deleted and the system MUST answer 409 `CONTEST_CONFLICT`.

**Data Deletion**
* **FR-DC-007**: The system MUST delete the Contest record when deletion is requested.
* **FR-DC-008**: The system MUST delete all Contest_Problem associations, registrations and team participations (cascade delete).
* **FR-DC-009**: The system MUST attempt to remove the cached standings projection (`standings:{contestId}`) of the contest. A failure to remove it MUST NOT fail the request, so the request may return 204 while the cache entry remains.

**Submissions Handling**
* **FR-DC-011**: The system MUST NOT delete submissions when a contest is deleted.
* **FR-DC-012**: The system MUST set `contest_id` to `null` for all submissions of the deleted contest.
* **FR-DC-013**: The system MUST handle submissions in PENDING state (set `contest_id` to `null` even if still being judged).

**Problems Handling**
* **FR-DC-014**: The system MUST NOT delete problems when a contest is deleted (problems are global entities).
* **FR-DC-015**: Problems remain available for use in other contests.

**Permissions**
* **FR-DC-016**: Only Leads of the contest's group and Admin can delete contests.
* **FR-DC-018**: Members who are not Leads MUST NOT be able to delete contests in any group.
* **FR-DC-019**: Contestants MUST NOT be able to delete contests in any group.
* **FR-DC-020**: Coaches who are not Leads of the global group MUST NOT be able to delete contests in the global group.

**Validation**
* **FR-DC-021**: The system MUST validate that the contest exists and belongs to the group in the path before attempting deletion; otherwise it returns 404 `CONTEST_NOT_FOUND`.
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
  * `contest_id` (FK to Contest, `ON DELETE CASCADE`)
  * **Deletion**: Cascade delete when contest is deleted

* **Contest registrations and team participations**: Who is registered to the contest (individually or as a team).
  * `contest_id` (FK to Contest, `ON DELETE CASCADE`)
  * **Deletion**: Cascade delete when contest is deleted

* **Standings cache**: Redis key `standings:{contestId}` (see View contest standings spec).
  * **Deletion**: Removal is attempted best-effort and can fail silently, leaving the key behind; the key is then harmless because standings are never served for a deleted contest

* **Submission**: Represents a contestant's solution attempt.
  * `contest_id` (FK to Contest, nullable, `ON DELETE SET NULL`)
  * **Deletion**: NOT deleted, but `contest_id` is set to `null` when contest is deleted

* **Problem**: Represents a programming problem.
  * **Deletion**: NOT deleted (problems are global entities)

> **Deletion Behavior Summary**:
> * **Deleted**: Contest, Contest_Problem, contest registrations, team participations
> * **Removal attempted (best effort)**: cached standings
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
DELETE /groups/{groupId}/contests/{contestId}
    ↓
Load contest (404 CONTEST_NOT_FOUND if missing or in another group)
    ↓
Validate permissions (Lead of the group, or Admin) → 403
    ↓
Reject if the contest is locked → 403 CONTEST_LOCKED
    ↓
Reject if the contest is ACTIVE → 409 CONTEST_IS_ACTIVE
    ↓
Delete Contest record only if it is still unlocked (one atomic statement)
(refused: 404 missing, 403 CONTEST_LOCKED, 409 CONTEST_CONFLICT)
(cascade: Contest_Problem, registrations, team participations;
 submissions.contest_id → NULL)
    ↓
Attempt to remove cached standings (best effort; a failure is ignored)
    ↓
Return 204 No Content
```

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-DC-001**: Leads can delete contests in their groups via `DELETE /groups/{groupId}/contests/{contestId}` with HTTP 204.
* **SC-DC-002**: Admin can delete contests in any group with HTTP 204.
* **SC-DC-003**: Leads of the global group can delete contests in global group with HTTP 204.
* **SC-DC-004**: Locked contests cannot be deleted - HTTP 403 `CONTEST_LOCKED`.
* **SC-DC-004.1**: ACTIVE contests cannot be deleted - HTTP 409 `CONTEST_IS_ACTIVE`.
* **SC-DC-004.2**: A contest locked while a delete is in flight is not deleted.
* **SC-DC-005**: Non-Lead non-admin users cannot delete contests in regular groups - HTTP 403.
* **SC-DC-006**: Non-Lead users cannot delete contests in global group - HTTP 403.
* **SC-DC-007**: A SCHEDULED or FINISHED contest can be deleted.
* **SC-DC-008**: Contest record is permanently deleted from database.
* **SC-DC-009**: All Contest_Problem associations, registrations and team participations are deleted.
* **SC-DC-010**: Removal of the cached standings of the contest is attempted; if it fails the request still returns HTTP 204.
* **SC-DC-012**: Problems are NOT deleted (remain in system).
* **SC-DC-013**: Submissions are NOT deleted but have `contest_id` set to `null`, including those still PENDING.
* **SC-DC-015**: Non-existent contests, or contests of another group, return HTTP 404.
* **SC-DC-017**: Unauthorized requests return HTTP 401.
* **SC-DC-018**: Deletion is idempotent in outcome (deleting an already-deleted contest returns 404).

---

## Not Enforced

* **Distinct "group not found"**: the response for a missing group is the same 404 `CONTEST_NOT_FOUND` as for a missing contest.

---

## Optional Notes

* **Idempotency**: Deleting a non-existent contest returns 404.
* **Cascade Deletion**: Contest_Problem, registrations and team participations are removed by database-level `ON DELETE CASCADE`; submissions are detached by `ON DELETE SET NULL`.
* **Performance**: For contests with many submissions, consider batch updates for setting `contest_id` to `null`.
* **Audit Trail**: Consider logging contest deletions for administrative purposes (future enhancement).
* **Soft Delete**: Current implementation uses hard delete. Soft delete could be considered for future versions to allow recovery.
* **Related specs**:
  * Create Contest: Initial contest creation
  * Update Contest: Modifying contest details
  * View Contest: Contest details and problem list
  * View Contest Standings: Standing calculation details
