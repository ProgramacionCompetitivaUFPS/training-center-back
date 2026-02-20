# Feature Specification: Delete Problem

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Delete problem (Priority: P3)

As a problem author or Admin, I want to permanently delete a problem from the system so that it is no longer available, while preserving historical submission data for users.

**Why this priority**: Problem deletion is an administrative operation that is infrequently needed. Most problems remain in the system indefinitely. However, it's useful for removing duplicate, test, or inappropriate problems. It's P3 because the system functions normally without this feature.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /problems/{slug}` endpoint with proper confirmation, validating that the problem and its associations are deleted while submissions are preserved.

**Acceptance Scenarios**:

1. **Scenario**: Successful deletion with confirmation
   - **Given** a problem exists (any status: DRAFT or PUBLISHED)
   - **And** the authenticated user is the author or Admin
   - **When** they request DELETE /problems/{slug} with correct slug confirmation in body
   - **Then** the system performs hard delete of:
     - Problem record
     - All problem files (test cases, solutions, checker, validator)
     - All Contest_Problem associations (problem removed from all contests)
   - **And** preserves all Submission records with problem_id intact
   - **And** returns HTTP 204 No Content

2. **Scenario**: Deletion without confirmation
   - **Given** a problem exists
   - **And** the authenticated user is the author or Admin
   - **When** they request DELETE /problems/{slug} without confirmation in body
   - **Then** the system rejects with HTTP 400 Bad Request
   - **And** returns error indicating confirmation is required

3. **Scenario**: Deletion with incorrect slug confirmation
   - **Given** a problem exists with slug "binary-search"
   - **And** the authenticated user is the author or Admin
   - **When** they request DELETE with confirmation slug "binary-tree" (incorrect)
   - **Then** the system rejects with HTTP 400 Bad Request
   - **And** returns error indicating slug confirmation doesn't match

4. **Scenario**: Delete problem in active contest
   - **Given** a problem exists and is included in an ACTIVE contest
   - **And** the authenticated user is the author or Admin
   - **When** they delete the problem with correct confirmation
   - **Then** the system deletes the problem and Contest_Problem association
   - **And** the problem disappears from the contest immediately
   - **And** contest standings are recalculated if necessary
   - **And** existing submissions remain visible in standings with preserved problem title

5. **Scenario**: Delete problem in scheduled contest
   - **Given** a problem exists and is included in a SCHEDULED (future) contest
   - **And** the authenticated user is the author or Admin
   - **When** they delete the problem with correct confirmation
   - **Then** the system deletes the problem and Contest_Problem association
   - **And** the problem is removed from the scheduled contest
   - **And** no notification is sent (contest hasn't started yet)

6. **Scenario**: Delete problem in finished contest
   - **Given** a problem exists and was included in a FINISHED contest
   - **And** the authenticated user is the author or Admin
   - **When** they delete the problem with correct confirmation
   - **Then** the system deletes the problem and Contest_Problem association
   - **And** existing submissions remain accessible with preserved problem title
   - **And** historical standings remain intact

7. **Scenario**: Delete problem with no submissions
   - **Given** a problem exists with zero submissions
   - **And** the authenticated user is the author or Admin
   - **When** they delete the problem with correct confirmation
   - **Then** the system deletes the problem successfully
   - **And** returns HTTP 204 No Content

8. **Scenario**: Delete problem not in any contest
   - **Given** a problem exists but is not included in any contest
   - **And** the authenticated user is the author or Admin
   - **When** they delete the problem with correct confirmation
   - **Then** the system deletes the problem successfully
   - **And** returns HTTP 204 No Content

9. **Scenario**: Non-author, non-Admin attempts deletion
   - **Given** a problem exists
   - **And** the authenticated user is a modifier (not author) or regular user
   - **When** they attempt to delete the problem
   - **Then** the system rejects with HTTP 403 Forbidden
   - **And** returns error code INSUFFICIENT_PERMISSIONS

10. **Scenario**: Unauthenticated deletion attempt
    - **Given** a problem exists
    - **When** a request is made without valid authentication credentials
    - **Then** the system rejects with HTTP 401 Unauthorized

11. **Scenario**: Problem not found
    - **Given** no problem exists with the provided slug
    - **When** a deletion request is submitted
    - **Then** the system returns HTTP 404 Not Found

12. **Scenario**: Access deleted problem
    - **Given** a problem was successfully deleted
    - **When** any user attempts to view the problem via GET /problems/{slug}
    - **Then** the system returns HTTP 404 Not Found

13. **Scenario**: Access statistics of deleted problem
    - **Given** a problem was successfully deleted
    - **When** any user attempts to view statistics via GET /problems/{slug}/statistics
    - **Then** the system returns HTTP 404 Not Found

14. **Scenario**: View submission after problem deletion
    - **Given** a problem was deleted
    - **And** submissions exist for that problem
    - **When** a user views a submission via GET /submissions/{id}
    - **Then** the system returns the submission with preserved problem title
    - **And** problem_id remains intact in the submission record

---

### Edge Cases

- Delete problem with hundreds of submissions (performance consideration)
- Delete problem that is in multiple contests simultaneously
- Delete problem while submissions are being judged
- Concurrent deletion attempts for same problem
- Delete problem with very large files (200MB test cases)
- Delete problem immediately after creation (no submissions, no contests)
- Attempt to delete same problem twice (idempotency)
- Delete problem with special characters in slug
- Standing recalculation when problem deleted from active contest
- Contest with only one problem, and that problem is deleted

## API Contract

### DELETE /problems/{slug}

Permanently delete a problem from the system, including all files and contest associations, while preserving submission history.

> **Important**: This is a destructive operation that cannot be undone. Requires explicit confirmation by providing the problem slug in the request body. Only the problem author or Admin can delete problems. Submissions are preserved to maintain user history.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | Unique problem identifier (3-70 chars, lowercase alphanumeric with hyphens) |

**Request Body**:

```json
{
  "confirmSlug": "binary-search"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| confirmSlug | string | Yes | Must exactly match the problem slug in the URL path for confirmation |

**Responses**:

#### 204 No Content
Problem successfully deleted. No response body.

#### 400 Bad Request
Invalid request (missing confirmation or slug mismatch).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Slug confirmation is required to delete this problem",
  "details": [
    {
      "field": "confirmSlug",
      "message": "Must match the problem slug exactly"
    }
  ]
}
```

**When slug doesn't match**:
```json
{
  "error": "SLUG_MISMATCH",
  "message": "Confirmation slug does not match the problem slug"
}
```

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission to delete this problem.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only the problem author or Admin can delete this problem"
}
```

#### 404 Not Found
Problem with the specified slug does not exist.

```json
{
  "error": "NOT_FOUND",
  "message": "Problem not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Access Control**
- **FR-001**: The system MUST allow only the problem author or Admin to delete problems.
- **FR-002**: The system MUST reject deletion attempts from modifiers (who are not the author) with HTTP 403 Forbidden.
- **FR-003**: The system MUST reject deletion attempts from unauthenticated users with HTTP 401 Unauthorized.
- **FR-004**: The system MUST return HTTP 404 Not Found for non-existent problem slugs.

**Confirmation Requirement**
- **FR-005**: The system MUST require explicit confirmation by providing the problem slug in the request body.
- **FR-006**: The system MUST validate that confirmSlug exactly matches the problem slug in the URL path.
- **FR-007**: The system MUST reject deletion requests without confirmSlug with HTTP 400 Bad Request.
- **FR-008**: The system MUST reject deletion requests with mismatched confirmSlug with HTTP 400 Bad Request.

**Deletion Scope**
- **FR-009**: The system MUST perform hard delete of the Problem record.
- **FR-010**: The system MUST delete all problem files from storage (test cases, solutions, checker, validator).
- **FR-011**: The system MUST delete all Contest_Problem associations (remove problem from all contests).
- **FR-012**: The system MUST preserve all Submission records with problem_id intact.
- **FR-013**: The system MUST preserve the problem title in submission records for historical reference.

**Contest Impact**
- **FR-014**: The system MUST allow deletion of problems that are in ACTIVE contests.
- **FR-015**: The system MUST allow deletion of problems that are in SCHEDULED contests.
- **FR-016**: The system MUST allow deletion of problems that are in FINISHED contests.
- **FR-017**: When a problem is deleted from an ACTIVE contest, the system MUST recalculate standings if necessary.
- **FR-018**: When a problem is deleted from a SCHEDULED contest, the system MUST remove it without notification.
- **FR-019**: When a problem is deleted from a FINISHED contest, the system MUST preserve historical standings.

**Submission Preservation**
- **FR-020**: The system MUST NOT delete Submission records when a problem is deleted.
- **FR-021**: Submissions MUST remain accessible via View submission endpoint after problem deletion.
- **FR-022**: Submissions MUST display the preserved problem title after problem deletion.
- **FR-023**: The problem_id field in submissions MUST remain intact after problem deletion.

**Post-Deletion Behavior**
- **FR-024**: After deletion, GET /problems/{slug} MUST return HTTP 404 Not Found.
- **FR-025**: After deletion, GET /problems/{slug}/statistics MUST return HTTP 404 Not Found.
- **FR-026**: After deletion, the problem slug MUST become available for reuse by new problems.
- **FR-027**: The system MUST return HTTP 204 No Content for successful deletions.

**Edge Cases**
- **FR-028**: The system MUST handle deletion of problems with zero submissions.
- **FR-029**: The system MUST handle deletion of problems not included in any contest.
- **FR-030**: The system MUST handle deletion of problems with large files (up to 200MB test cases).
- **FR-031**: The system MUST handle concurrent deletion attempts gracefully (idempotency).
- **FR-032**: The system MUST handle deletion while submissions are being judged.

### Key Entities

- **Problem**: Represents a programming problem.  
  Relevant attributes:
  - `id` (string, UUID, internal only)
  - `slug` (string, unique, 3-70 chars)
  - `title` (string, preserved in submissions)
  - `status` (enum: DRAFT | PUBLISHED)
  - `authorId` (string, UUID, FK to User)
  - `testCasesFileKey` (string, nullable)
  - `solutionFileKeys` (array of strings)
  - `checkerFileKey` (string, nullable)
  - `validatorFileKey` (string, nullable)

- **Contest_Problem**: Association between contest and problem.  
  Relevant attributes:
  - `id` (string, UUID, internal only)
  - `contestId` (string, UUID, FK to Contest)
  - `problemId` (string, UUID, FK to Problem)
  - `position` (integer)

- **Submission**: Code submission for a problem.  
  Relevant attributes:
  - `id` (string, UUID, internal only)
  - `problemId` (string, UUID, FK to Problem - preserved after deletion)
  - `problemTitle` (string, preserved for display)
  - `contestId` (string, UUID, FK to Contest, nullable)

- **User**: Represents a user.  
  Relevant attributes:
  - `id` (string, UUID, internal only)
  - `role` (enum: ADMIN | COACH | CONTESTANT)

> **Note on Deletion**: Problem deletion is a hard delete operation. The Problem record and all associated files are permanently removed. Contest_Problem associations are deleted to remove the problem from all contests. Submissions are preserved with problem_id and problemTitle intact for historical reference.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Problem author can successfully delete their own problems with correct confirmation.
- **SC-002**: Admin can successfully delete any problem with correct confirmation.
- **SC-003**: Modifiers (non-authors) cannot delete problems and receive HTTP 403 Forbidden.
- **SC-004**: Deletion requests without confirmSlug are rejected with HTTP 400 Bad Request.
- **SC-005**: Deletion requests with mismatched confirmSlug are rejected with HTTP 400 Bad Request.
- **SC-006**: Problem record is permanently deleted from database.
- **SC-007**: All problem files are deleted from storage (test cases, solutions, checker, validator).
- **SC-008**: All Contest_Problem associations are deleted (problem removed from all contests).
- **SC-009**: All Submission records are preserved with problem_id intact.
- **SC-010**: Submissions display preserved problem title after deletion.
- **SC-011**: Problems can be deleted from ACTIVE contests with standings recalculation.
- **SC-012**: Problems can be deleted from SCHEDULED contests without notification.
- **SC-013**: Problems can be deleted from FINISHED contests with preserved standings.
- **SC-014**: Deleted problems return HTTP 404 Not Found on subsequent access attempts.
- **SC-015**: Statistics for deleted problems return HTTP 404 Not Found.
- **SC-016**: Submissions remain accessible after problem deletion.
- **SC-017**: Deleted problem slugs become available for reuse.
- **SC-018**: Successful deletions return HTTP 204 No Content.
- **SC-019**: Unauthenticated requests receive HTTP 401 Unauthorized.
- **SC-020**: Non-existent problem slugs return HTTP 404 Not Found.
