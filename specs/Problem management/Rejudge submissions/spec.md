# Feature Specification: Rejudge Submissions

**Created**: 2025-12-23

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Contest owner rejudges submissions in active contest (Priority: P1)

As a contest owner (the user who created the contest), I want to rejudge all submissions affected by test case changes in my active contest so that the rankings reflect the correct verdicts with the updated test cases.

**Why this priority**: When test cases are updated during an active contest, maintaining fair and accurate rankings is critical. This is the primary use case for rejudging - ensuring all contestants are evaluated fairly with the same test cases.

**Independent Test**: This user story can be tested independently by consuming the `POST /contests/{contestId}/problems/{slug}/rejudge` endpoint during an active contest, validating that affected submissions are rejudged and the Standing is recalculated.

**Acceptance Scenarios**:

1. **Scenario**: Successful rejudge in active contest
   - **Given** a contest is active (current time is between startTime and endTime)
   - **And** the authenticated user is the contest owner
   - **And** a problem in the contest has updated test cases (testCasesUpdatedAt changed)
   - **And** there are submissions where `submittedAt < testCasesUpdatedAt`
   - **When** the owner triggers rejudge for that problem
   - **Then** the system queues all affected submissions for rejudging (verdict → PENDING)
   - **And** rejudges them asynchronously
   - **And** updates the Standing automatically after each submission is rejudged
   - **And** returns confirmation with count of submissions queued

2. **Scenario**: Rejudge with no affected submissions
   - **Given** a contest is active
   - **And** the authenticated user is the contest owner
   - **And** all submissions for the problem were submitted after testCasesUpdatedAt
   - **When** the owner triggers rejudge
   - **Then** the system returns success with 0 submissions queued

3. **Scenario**: Non-owner attempts to rejudge in contest
   - **Given** a contest is active
   - **And** the authenticated user is NOT the contest owner
   - **When** they attempt to rejudge a problem in that contest
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

4. **Scenario**: Contest not found
   - **Given** the contestId does not exist
   - **When** rejudge is attempted
   - **Then** the system rejects with 404 Not Found

5. **Scenario**: Problem not in contest
   - **Given** a contest exists
   - **And** the authenticated user is the contest owner
   - **And** the problem is not part of that contest
   - **When** rejudge is attempted
   - **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_IN_CONTEST)

6. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a rejudge request is submitted
   - **Then** the system rejects with 401 Unauthorized

---

### User Story 2 – Contestant manually rejudges own submission (Priority: P1)

As a contestant, I want to manually rejudge my submission when I notice the problem's test cases have been updated so that I can see my verdict with the new test cases.

**Why this priority**: Contestants need the ability to update their submission results when test cases change, especially for practice submissions or submissions in finished contests where the standing is frozen.

**Independent Test**: This user story can be tested independently by consuming the `POST /submissions/{submissionId}/rejudge` endpoint, validating that the contestant's own submission is rejudged when test cases have been updated.

**Acceptance Scenarios**:

1. **Scenario**: Successful manual rejudge by contestant
   - **Given** a submission exists where `submittedAt < testCasesUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge for their submission
   - **Then** the system changes verdict to PENDING
   - **And** queues the submission for asynchronous rejudging
   - **And** returns confirmation

2. **Scenario**: Rejudge submission that doesn't need rejudging
   - **Given** a submission exists where `submittedAt >= testCasesUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they attempt to rejudge
   - **Then** the system rejects with 400 Bad Request (NO_REJUDGE_NEEDED)

3. **Scenario**: Rejudge submission from finished contest
   - **Given** a submission exists in a finished contest
   - **And** `submittedAt < testCasesUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge
   - **Then** the system rejudges the submission
   - **And** does NOT update the contest Standing (standing is frozen)

4. **Scenario**: Rejudge submission from active contest
   - **Given** a submission exists in an active contest
   - **And** `submittedAt < testCasesUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge
   - **Then** the system rejects with 403 Forbidden (USE_CONTEST_REJUDGE)
   - **And** suggests using the contest owner's rejudge endpoint

5. **Scenario**: User attempts to rejudge another user's submission
   - **Given** a submission exists
   - **And** the authenticated user is NOT the submission owner
   - **And** the authenticated user is NOT an admin
   - **When** they attempt to rejudge
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

6. **Scenario**: Submission not found
   - **Given** the submissionId does not exist
   - **When** rejudge is attempted
   - **Then** the system rejects with 404 Not Found

---

### User Story 3 – Admin rejudges all affected submissions for a problem (Priority: P1)

As an Admin, I want to rejudge all submissions affected by test case changes for a specific problem (across all contexts) so that I can ensure data consistency across the entire platform.

**Why this priority**: Admins need comprehensive control to fix issues when test cases are updated, affecting submissions across multiple contests and practice sessions.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/problems/{slug}/rejudge` endpoint, validating that all submissions where `submittedAt < testCasesUpdatedAt` are rejudged regardless of context.

**Acceptance Scenarios**:

1. **Scenario**: Admin rejudges all affected submissions
   - **Given** the authenticated user has ADMIN role
   - **And** a problem has submissions where `submittedAt < testCasesUpdatedAt`
   - **When** the admin triggers global rejudge for the problem
   - **Then** the system queues ALL affected submissions (from all contests and practice)
   - **And** rejudges them asynchronously
   - **And** updates Standing only for active contests
   - **And** does NOT update Standing for finished contests
   - **And** returns count of submissions queued

2. **Scenario**: Non-admin attempts admin rejudge
   - **Given** the authenticated user does NOT have ADMIN role
   - **When** they attempt to use admin rejudge endpoint
   - **Then** the system rejects with 403 Forbidden (ADMIN_REQUIRED)

3. **Scenario**: Problem not found
   - **Given** the problem slug does not exist
   - **When** admin rejudge is attempted
   - **Then** the system rejects with 404 Not Found

4. **Scenario**: No submissions need rejudging
   - **Given** all submissions for the problem have `submittedAt >= testCasesUpdatedAt`
   - **When** admin triggers rejudge
   - **Then** the system returns success with 0 submissions queued

---

### User Story 4 – Admin rejudges submissions for specific contest (Priority: P2)

As an Admin, I want to rejudge all affected submissions for a problem within a specific contest so that I can fix issues in a targeted way without affecting other contests.

**Why this priority**: Provides more granular control for admins when issues are isolated to a specific contest. Lower priority than global rejudge as it's a more specific use case.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/problems/{slug}/rejudge?contestId={contestId}` endpoint, validating that only submissions from the specified contest are rejudged.

**Acceptance Scenarios**:

1. **Scenario**: Admin rejudges problem in specific contest
   - **Given** the authenticated user has ADMIN role
   - **And** a problem has submissions in a specific contest
   - **And** those submissions have `submittedAt < testCasesUpdatedAt`
   - **When** the admin triggers rejudge with contestId filter
   - **Then** the system queues only submissions from that contest
   - **And** rejudges them asynchronously
   - **And** updates Standing if contest is active
   - **And** does NOT update Standing if contest is finished
   - **And** returns count of submissions queued

2. **Scenario**: Contest not found
   - **Given** the authenticated user has ADMIN role
   - **And** the contestId does not exist
   - **When** admin rejudge with contestId is attempted
   - **Then** the system rejects with 404 Not Found (CONTEST_NOT_FOUND)

3. **Scenario**: Problem not in specified contest
   - **Given** the authenticated user has ADMIN role
   - **And** the problem is not part of the specified contest
   - **When** admin rejudge with contestId is attempted
   - **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_IN_CONTEST)

---

### User Story 5 – Admin rejudges specific submission (Priority: P2)

As an Admin, I want to rejudge a specific submission regardless of test case updates so that I can fix individual issues or investigate problems.

**Why this priority**: Provides fine-grained control for debugging and fixing individual submission issues. Lower priority as it's an edge case for troubleshooting.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/submissions/{submissionId}/rejudge` endpoint, validating that the specific submission is rejudged regardless of testCasesUpdatedAt.

**Acceptance Scenarios**:

1. **Scenario**: Admin rejudges specific submission
   - **Given** the authenticated user has ADMIN role
   - **And** a submission exists
   - **When** the admin triggers rejudge for that submission
   - **Then** the system changes verdict to PENDING
   - **And** queues the submission for rejudging (even if `submittedAt >= testCasesUpdatedAt`)
   - **And** updates Standing if submission is in active contest
   - **And** returns confirmation

2. **Scenario**: Admin rejudges submission from finished contest
   - **Given** the authenticated user has ADMIN role
   - **And** a submission exists in a finished contest
   - **When** the admin triggers rejudge
   - **Then** the system rejudges the submission
   - **And** does NOT update the contest Standing

3. **Scenario**: Non-admin attempts admin submission rejudge
   - **Given** the authenticated user does NOT have ADMIN role
   - **When** they attempt to use admin submission rejudge endpoint
   - **Then** the system rejects with 403 Forbidden (ADMIN_REQUIRED)

---

### Edge Cases

- Concurrent rejudge requests for the same submission (should be idempotent or queued).
- Submission already in PENDING state when rejudge is requested.
- Very large number of submissions to rejudge (e.g., 10,000+).
- Submission was made during active contest, but contest finishes before rejudge completes (Standing should NOT update since contest is now finished).
- Test cases updated multiple times in quick succession.
- Submission code reference is invalid or missing (should fail gracefully).
- Standing calculation fails during rejudge (should log error, not block rejudge).
- Admin rejudges while contest owner simultaneously rejudges same submissions.
- Rejudge request for a problem with status NOT_VISIBLE (should be allowed - submissions exist from when problem was VISIBLE).
- Network interruption during asynchronous rejudging.

---

## API Contract

### POST /contests/{contestId}/problems/{slug}/rejudge

Rejudge all affected submissions for a problem within a specific contest. Only the contest owner can trigger this.

> **Important**: This endpoint only works during active contests. Only submissions where `submittedAt < testCasesUpdatedAt` are rejudged. The Standing is automatically updated after each submission is rejudged. The authenticated user must be the contest owner.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| contestId | string (UUID) | Yes | The unique identifier of the contest |
| slug | string | Yes | The unique slug of the problem |

**Responses**:

#### 200 OK
Rejudge initiated successfully.

```json
{
  "message": "Rejudge initiated successfully",
  "contestId": "a1b2c3d4-e5f6-7890-1234-567890123456",
  "problemSlug": "sum-of-two-numbers",
  "submissionsQueued": 15,
  "contestStatus": "ACTIVE",
  "standingWillUpdate": true
}
```

#### 400 Bad Request
Problem not in contest or no submissions need rejudging.

```json
{
  "error": "PROBLEM_NOT_IN_CONTEST",
  "message": "The specified problem is not part of this contest"
}
```

```json
{
  "error": "NO_SUBMISSIONS_TO_REJUDGE",
  "message": "No submissions need rejudging for this problem"
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
User is not the contest owner.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only the contest owner can rejudge submissions in this contest"
}
```

#### 404 Not Found
Contest or problem not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Contest not found"
}
```

```json
{
  "error": "NOT_FOUND",
  "message": "Problem not found"
}
```

---

### POST /submissions/{submissionId}/rejudge

Manually rejudge a specific submission. Contestant can rejudge their own submissions outside of active contests.

> **Important**: Only works for submissions outside active contests or in finished contests. For active contests, use the contest owner's rejudge endpoint. Only the submission owner can rejudge their submission. Only works when `submittedAt < testCasesUpdatedAt`.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| submissionId | string (UUID) | Yes | The unique identifier of the submission |

**Responses**:

#### 200 OK
Submission rejudge initiated successfully.

```json
{
  "message": "Submission rejudge initiated successfully",
  "submissionId": "a1b2c3d4-e5f6-7890-1234-567890123456",
  "previousVerdict": "ACCEPTED",
  "currentStatus": "PENDING",
  "problemSlug": "sum-of-two-numbers"
}
```

#### 400 Bad Request
Submission doesn't need rejudging.

```json
{
  "error": "NO_REJUDGE_NEEDED",
  "message": "This submission does not need rejudging. Test cases have not been updated since submission."
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
User doesn't have permission or submission is in active contest.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "You can only rejudge your own submissions"
}
```

```json
{
  "error": "USE_CONTEST_REJUDGE",
  "message": "This submission is part of an active contest. Please contact the contest owner to rejudge all affected submissions in the contest."
}
```

#### 404 Not Found
Submission not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Submission not found"
}
```

---

### POST /admin/problems/{slug}/rejudge

Admin rejudges all affected submissions for a problem across all contexts (all contests and practice).

> **Important**: Only users with ADMIN role can use this endpoint. Rejudges all submissions where `submittedAt < testCasesUpdatedAt`. Standing is updated only for active contests, not for finished contests.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for admin authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Responses**:

#### 200 OK
Admin rejudge initiated successfully.

```json
{
  "message": "Admin rejudge initiated successfully",
  "problemSlug": "sum-of-two-numbers",
  "submissionsQueued": 45,
  "breakdown": {
    "activeContests": 12,
    "finishedContests": 18,
    "practice": 15
  },
  "standingUpdates": {
    "willUpdate": ["contest-a1b2c3d4", "contest-e5f6g7h8"],
    "frozen": ["contest-i9j0k1l2"]
  }
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
User does not have admin privileges.

```json
{
  "error": "ADMIN_REQUIRED",
  "message": "Admin privileges required for this operation"
}
```

#### 404 Not Found
Problem not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Problem not found"
}
```

---

### POST /admin/problems/{slug}/rejudge?contestId={contestId}

Admin rejudges all affected submissions for a problem within a specific contest.

> **Important**: Only users with ADMIN role can use this endpoint. Allows targeted rejudging for a specific contest. Standing updated only if contest is active.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for admin authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Query Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Responses**:

#### 200 OK
Admin rejudge initiated successfully for specific contest.

```json
{
  "message": "Admin rejudge initiated successfully",
  "problemSlug": "sum-of-two-numbers",
  "contestId": "a1b2c3d4-e5f6-7890-1234-567890123456",
  "submissionsQueued": 12,
  "contestStatus": "ACTIVE",
  "standingWillUpdate": true
}
```

#### 400 Bad Request
Problem not in contest.

```json
{
  "error": "PROBLEM_NOT_IN_CONTEST",
  "message": "The specified problem is not part of this contest"
}
```

#### 401 Unauthorized
Authentication failed.

#### 403 Forbidden
User does not have admin privileges.

```json
{
  "error": "ADMIN_REQUIRED",
  "message": "Admin privileges required for this operation"
}
```

#### 404 Not Found
Problem or contest not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Contest not found"
}
```

---

### POST /admin/submissions/{submissionId}/rejudge

Admin rejudges a specific submission regardless of whether test cases have been updated.

> **Important**: Only users with ADMIN role can use this endpoint. Allows rejudging any submission for debugging purposes. Standing updated only if submission is in active contest.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for admin authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| submissionId | string (UUID) | Yes | The unique identifier of the submission |

**Responses**:

#### 200 OK
Admin submission rejudge initiated successfully.

```json
{
  "message": "Admin submission rejudge initiated successfully",
  "submissionId": "a1b2c3d4-e5f6-7890-1234-567890123456",
  "previousVerdict": "ACCEPTED",
  "currentStatus": "PENDING",
  "problemSlug": "sum-of-two-numbers",
  "contestId": "c1d2e3f4-g5h6-7890-1234-567890123456",
  "contestStatus": "FINISHED",
  "standingWillUpdate": false
}
```

#### 401 Unauthorized
Authentication failed.

#### 403 Forbidden
User does not have admin privileges.

```json
{
  "error": "ADMIN_REQUIRED",
  "message": "Admin privileges required for this operation"
}
```

#### 404 Not Found
Submission not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Submission not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Detection and Triggering**
- **FR-001**: The system MUST determine if a submission needs rejudging by comparing `submittedAt < testCasesUpdatedAt`.
- **FR-002**: The system MUST automatically update `testCasesUpdatedAt` timestamp whenever test cases are uploaded via `POST /problems/{slug}/files` with fileType=testCases.
- **FR-003**: The system MUST allow contest owners to rejudge all affected submissions in their active contests.
- **FR-004**: The system MUST allow contestants to manually rejudge their own submissions outside of active contests.
- **FR-005**: The system MUST allow admins to rejudge submissions at global, contest, or individual submission level.

**Rejudging Process**
- **FR-006**: The system MUST change submission verdict to PENDING when rejudging is initiated.
- **FR-007**: The system MUST process all rejudging asynchronously (non-blocking).
- **FR-008**: The system MUST execute the submission code against all current test cases during rejudging.
- **FR-009**: The system MUST update the submission verdict based on rejudging results (ACCEPTED, WRONG_ANSWER, TIME_LIMIT_EXCEEDED, MEMORY_LIMIT_EXCEEDED, RUNTIME_ERROR, etc.).
- **FR-010**: The system MUST NOT maintain a history of previous verdicts.

**Standing Updates**
- **FR-011**: The system MUST automatically recalculate and update Standing after each submission is rejudged in an ACTIVE contest.
- **FR-012**: The system MUST NOT update Standing for rejudged submissions in FINISHED contests (standing is frozen).
- **FR-013**: The system MUST update Standing only for contests where `currentTime` is between `startTime` and `endTime` at the moment the rejudge completes.
- **FR-014**: If a submission was made during an active contest but the contest finishes before rejudging completes, the Standing MUST NOT be updated.

**Permissions**
- **FR-015**: The system MUST allow only the contest owner to rejudge submissions in their contest.
- **FR-016**: The system MUST allow contestants to rejudge only their own submissions.
- **FR-017**: The system MUST prevent contestants from rejudging submissions in active contests (must use contest owner's rejudge).
- **FR-018**: The system MUST allow only users with ADMIN role to use admin rejudge endpoints.
- **FR-019**: The system MUST allow admins to rejudge any submission regardless of owner or context.

**Validation**
- **FR-020**: The system MUST validate that the problem exists and is part of the specified contest (for contest-scoped rejudge).
- **FR-021**: The system MUST validate that the contest owner is the authenticated user (for contest rejudge).
- **FR-022**: The system MUST validate that submissions belong to the authenticated user (for contestant manual rejudge).
- **FR-023**: The system MUST reject contestant manual rejudge requests for submissions in active contests.
- **FR-024**: The system MUST allow rejudging submissions of problems with status NOT_VISIBLE (submissions may exist from when problem was VISIBLE).

**General**
- **FR-025**: The system MUST NOT return internal IDs in responses (except where needed as identifiers).
- **FR-026**: The system MUST return the count of submissions queued for rejudging.
- **FR-027**: The system MUST return validation and authorization errors with consistent structure.

### Key Entities

- **Problem**: Extended with rejudging-related timestamp.  
  Key attributes:
  - `slug` (string, unique)
  - `testCasesUpdatedAt` (timestamp, updated when test cases are uploaded)
  - `status` (enum: `NOT_VISIBLE` | `VISIBLE`)
  - Other attributes from Create Problem spec

- **Submission**: Represents a contestant's solution attempt.  
  Key attributes:
  - `id` (string, UUID)
  - `contestant_id` (string, UUID, FK to User)
  - `problem_id` (string, UUID, FK to Problem)
  - `contest_id` (string, UUID, FK to Contest, nullable - null for practice submissions)
  - `codeReference` (string, reference to stored code, **pending definition**)
  - `verdict` (enum: `PENDING` | `ACCEPTED` | `WRONG_ANSWER` | `TIME_LIMIT_EXCEEDED` | `MEMORY_LIMIT_EXCEEDED` | `RUNTIME_ERROR` | `COMPILATION_ERROR` | etc.)
  - `submittedAt` (timestamp)

- **Contest**: Represents a programming competition.  
  Key attributes:
  - `id` (string, UUID)
  - `ownerId` (string, UUID, FK to User, the creator of the contest)
  - `startTime` (timestamp)
  - `endTime` (timestamp)
  - `group_id` (string, UUID, FK to Group, nullable)
  - Other attributes to be defined in Contest spec

- **Standing**: Tracks contestant rankings within a contest.  
  Key attributes:
  - `id` (string, UUID)
  - `contest_id` (string, UUID, FK to Contest)
  - `contestant_id` (string, UUID, FK to User)
  - `problemsSolved` (integer)
  - `totalAttempts` (integer)
  - Other ranking-related attributes to be defined in Standing spec

> **Contest States (for rejudging logic)**:
> - `ACTIVE`: Current time is between startTime and endTime → Standing WILL be updated
> - `FINISHED`: Current time is after endTime → Standing is FROZEN, will NOT be updated

### Submission Verdicts

| Verdict | Description |
|---------|-------------|
| PENDING | Submission is being judged or rejudged |
| ACCEPTED | Solution passed all test cases |
| WRONG_ANSWER | Solution produced incorrect output |
| TIME_LIMIT_EXCEEDED | Solution exceeded time limit |
| MEMORY_LIMIT_EXCEEDED | Solution exceeded memory limit |
| RUNTIME_ERROR | Solution crashed during execution |
| COMPILATION_ERROR | Solution failed to compile |

### Permission Matrix

| Action | Contest Owner | Contestant (own) | Admin | Other Users |
|--------|---------------|------------------|-------|-------------|
| Rejudge all in active contest | ✅ (own contest) | ❌ | ✅ | ❌ |
| Rejudge own submission (outside active contest) | ✅ | ✅ | ✅ | ❌ |
| Rejudge all for problem (global) | ❌ | ❌ | ✅ | ❌ |
| Rejudge all in specific contest | ❌ | ❌ | ✅ | ❌ |
| Rejudge specific submission | ✅ (own only) | ✅ (own only) | ✅ | ❌ |

### Rejudge Decision Logic

```
Function: needsRejudge(submission, problem)
  return submission.submittedAt < problem.testCasesUpdatedAt

Function: shouldUpdateStanding(submission, contest)
  if contest is null:
    return false
  currentTime = now()
  isActive = currentTime >= contest.startTime AND currentTime <= contest.endTime
  return isActive
```

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Contest owners can rejudge all affected submissions in their active contests via `POST /contests/{contestId}/problems/{slug}/rejudge` with HTTP 200.
- **SC-002**: Contestants can manually rejudge their own submissions outside active contests via `POST /submissions/{submissionId}/rejudge` with HTTP 200.
- **SC-003**: Admins can rejudge all affected submissions globally via `POST /admin/problems/{slug}/rejudge` with HTTP 200.
- **SC-004**: Admins can rejudge all affected submissions in a specific contest via `POST /admin/problems/{slug}/rejudge?contestId={contestId}` with HTTP 200.
- **SC-005**: Admins can rejudge a specific submission via `POST /admin/submissions/{submissionId}/rejudge` with HTTP 200.
- **SC-006**: Submissions are correctly identified for rejudging when `submittedAt < testCasesUpdatedAt`.
- **SC-007**: Submission verdict changes to PENDING when rejudging is initiated.
- **SC-008**: Standing is automatically updated after rejudging for submissions in active contests.
- **SC-009**: Standing is NOT updated after rejudging for submissions in finished contests.
- **SC-010**: Contestants cannot rejudge submissions in active contests (must go through contest owner).
- **SC-011**: Only contest owners can rejudge submissions in their contests.
- **SC-012**: Only admins can use admin rejudge endpoints (HTTP 403 for non-admins).
- **SC-013**: Users cannot rejudge other users' submissions (except admins).
- **SC-014**: All rejudging is processed asynchronously.
- **SC-015**: `testCasesUpdatedAt` is automatically updated when test cases are uploaded.

---

## Optional Notes

- **Asynchronous Processing**: All rejudging should be processed through a queue system to handle large volumes without blocking API responses.
- **Idempotency**: Consider making rejudge requests idempotent - if a submission is already queued or being rejudged, subsequent requests should not create duplicate jobs.
- **Batch Operations**: For admin operations affecting many submissions, consider implementing progress tracking or webhook notifications.
- **Standing Calculation**: The detailed algorithm for Standing calculation will be defined in a separate Standing/Ranking spec.
- **Code Reference**: The mechanism for storing and retrieving submission code (`codeReference`) is pending definition and will be specified in the Submission spec.
- **Monitoring**: Consider adding logging and metrics for rejudge operations to track performance and identify issues.
- **Rate Limiting**: Consider rate limiting rejudge requests to prevent abuse, especially for the manual contestant rejudge endpoint.
- **Related specs**:
  - Create Problem: Initial problem creation
  - Update Problem: Uploading test cases (triggers testCasesUpdatedAt update)
  - Publish Problem: Problem validation and publication
  - Submit Solution: Creating submissions (to be defined)
  - Contest Management: Creating and managing contests (to be defined)
  - Standing/Ranking: Calculating and displaying contest rankings (to be defined)

