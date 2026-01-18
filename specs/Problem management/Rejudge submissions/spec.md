# Feature Specification: Rejudge Submissions

**Created**: 2025-12-23

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Contest owner or Lead rejudges submissions in active contest (Priority: P1)

As a contest owner (the user who created the contest) or a Lead of the group, I want to rejudge all submissions affected by judging component changes (test cases, checker, or validator) in an active contest so that the rankings reflect the correct verdicts with the updated components.

**Why this priority**: When judging components (test cases, checker, or validator) are updated during an active contest, maintaining fair and accurate rankings is critical. This is the primary use case for rejudging - ensuring all contestants are evaluated fairly with the same judging criteria. Leads need this capability to manage contests effectively.

**Independent Test**: This user story can be tested independently by consuming the `POST /contests/{contestId}/problems/{slug}/rejudge` endpoint during an active contest, validating that affected submissions are rejudged and the Standing is recalculated.

**Acceptance Scenarios**:

1. **Scenario**: Successful rejudge in active contest by owner
   - **Given** a contest is active (current time is between startTime and endTime)
   - **And** the authenticated user is the contest owner
   - **And** a problem in the contest has updated judging components (problemJudgingUpdatedAt changed)
   - **And** there are submissions where `submittedAt < problemJudgingUpdatedAt`
   - **When** the owner triggers rejudge for that problem
   - **Then** the system queues all affected submissions for rejudging (verdict → PENDING)
   - **And** rejudges them asynchronously
   - **And** updates the Standing automatically after each submission is rejudged
   - **And** returns confirmation with count of submissions queued

2. **Scenario**: Successful rejudge in active contest by Lead
   - **Given** a contest is active (current time is between startTime and endTime)
   - **And** the authenticated user is a Lead of the group
   - **And** a problem in the contest has updated judging components (problemJudgingUpdatedAt changed)
   - **And** there are submissions where `submittedAt < problemJudgingUpdatedAt`
   - **When** the Lead triggers rejudge for that problem
   - **Then** the system queues all affected submissions for rejudging (verdict → PENDING)
   - **And** rejudges them asynchronously
   - **And** updates the Standing automatically after each submission is rejudged
   - **And** returns confirmation with count of submissions queued

3. **Scenario**: Rejudge with no affected submissions
   - **Given** a contest is active
   - **And** the authenticated user is the contest owner or a Lead
   - **And** all submissions for the problem were submitted after problemJudgingUpdatedAt
   - **When** they trigger rejudge
   - **Then** the system returns success with 0 submissions queued

4. **Scenario**: Non-owner non-Lead attempts to rejudge in contest
   - **Given** a contest is active
   - **And** the authenticated user is NOT the contest owner and NOT a Lead
   - **When** they attempt to rejudge a problem in that contest
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

5. **Scenario**: Contest not found
   - **Given** the contestId does not exist
   - **When** rejudge is attempted
   - **Then** the system rejects with 404 Not Found

6. **Scenario**: Problem not in contest
   - **Given** a contest exists
   - **And** the authenticated user is the contest owner or a Lead
   - **And** the problem is not part of that contest
   - **When** rejudge is attempted
   - **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_IN_CONTEST)

7. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a rejudge request is submitted
   - **Then** the system rejects with 401 Unauthorized

---

### User Story 2 – Contestant manually rejudges own submission (Priority: P1)

As a contestant, I want to manually rejudge my submission when I notice the problem's judging components (test cases, checker, or validator) have been updated so that I can see my verdict with the new judging criteria.

**Why this priority**: Contestants need the ability to update their submission results when judging components change, especially for practice submissions or submissions in finished contests where the standing is frozen.

**Independent Test**: This user story can be tested independently by consuming the `POST /submissions/{submissionId}/rejudge` endpoint, validating that the contestant's own submission is rejudged when judging components have been updated.

**Acceptance Scenarios**:

1. **Scenario**: Successful manual rejudge by contestant
   - **Given** a submission exists where `submittedAt < problemJudgingUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge for their submission
   - **Then** the system changes verdict to PENDING
   - **And** queues the submission for asynchronous rejudging
   - **And** returns confirmation

2. **Scenario**: Rejudge submission that doesn't need rejudging
   - **Given** a submission exists where `submittedAt >= problemJudgingUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they attempt to rejudge
   - **Then** the system rejects with 400 Bad Request (NO_REJUDGE_NEEDED)

3. **Scenario**: Rejudge submission from finished contest
   - **Given** a submission exists in a finished contest
   - **And** `submittedAt < problemJudgingUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge
   - **Then** the system rejudges the submission
   - **And** does NOT update the contest Standing (standing is frozen)

4. **Scenario**: Rejudge submission from active contest - restricted
   - **Given** a submission exists in an active contest
   - **And** `submittedAt < problemJudgingUpdatedAt`
   - **And** `submittedAt <= contest.endTime` (not postcompetition)
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge
   - **Then** the system rejects with 403 Forbidden (CANNOT_REJUDGE_IN_ACTIVE_CONTEST)
   - **And** indicates that rejudge is restricted in active contests

5. **Scenario**: Rejudge own submission in postcompetition - allowed
   - **Given** a submission exists in a finished contest
   - **And** the contest has `enablePostContest = true`
   - **And** `submittedAt > contest.endTime` (postcompetition submission)
   - **And** `submittedAt < problemJudgingUpdatedAt`
   - **And** the authenticated user is the contestant who made the submission
   - **When** they trigger rejudge
   - **Then** the system rejudges the submission
   - **And** does NOT update the contest Standing (standing is frozen)
   - **And** returns confirmation

6. **Scenario**: User attempts to rejudge another user's submission
   - **Given** a submission exists
   - **And** the authenticated user is NOT the submission owner
   - **And** the authenticated user is NOT an admin
   - **And** the authenticated user is NOT a Lead
   - **When** they attempt to rejudge
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

7. **Scenario**: Submission not found
   - **Given** the submissionId does not exist
   - **When** rejudge is attempted
   - **Then** the system rejects with 404 Not Found

---

### User Story 3 – Admin rejudges all affected submissions for a problem (Priority: P1)

As an Admin, I want to rejudge all submissions affected by judging component changes (test cases, checker, or validator) for a specific problem (across all contexts) so that I can ensure data consistency across the entire platform.

**Why this priority**: Admins need comprehensive control to fix issues when judging components are updated, affecting submissions across multiple contests and practice sessions.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/problems/{slug}/rejudge` endpoint, validating that all submissions where `submittedAt < problemJudgingUpdatedAt` are rejudged regardless of context.

**Acceptance Scenarios**:

1. **Scenario**: Admin rejudges all affected submissions
   - **Given** the authenticated user has ADMIN role
   - **And** a problem has submissions where `submittedAt < problemJudgingUpdatedAt`
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
   - **Given** all submissions for the problem have `submittedAt >= problemJudgingUpdatedAt`
   - **When** admin triggers rejudge
   - **Then** the system returns success with 0 submissions queued

---

### User Story 4 – Admin rejudges submissions for specific contest (Priority: P2)

As an Admin, I want to rejudge all affected submissions for a problem within a specific contest so that I can fix issues caused by judging component changes in a targeted way without affecting other contests.

**Why this priority**: Provides more granular control for admins when issues are isolated to a specific contest. Lower priority than global rejudge as it's a more specific use case.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/problems/{slug}/rejudge?contestId={contestId}` endpoint, validating that only submissions from the specified contest are rejudged.

**Acceptance Scenarios**:

1. **Scenario**: Admin rejudges problem in specific contest
   - **Given** the authenticated user has ADMIN role
   - **And** a problem has submissions in a specific contest
   - **And** those submissions have `submittedAt < problemJudgingUpdatedAt`
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

As an Admin, I want to rejudge a specific submission regardless of judging component updates so that I can fix individual issues or investigate problems.

**Why this priority**: Provides fine-grained control for debugging and fixing individual submission issues. Lower priority as it's an edge case for troubleshooting.

**Independent Test**: This user story can be tested independently by consuming the `POST /admin/submissions/{submissionId}/rejudge` endpoint, validating that the specific submission is rejudged regardless of problemJudgingUpdatedAt.

**Acceptance Scenarios**:

1. **Scenario**: Admin rejudges specific submission
   - **Given** the authenticated user has ADMIN role
   - **And** a submission exists
   - **When** the admin triggers rejudge for that submission
   - **Then** the system changes verdict to PENDING
   - **And** queues the submission for rejudging (even if `submittedAt >= problemJudgingUpdatedAt`)
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
- Judging components (test cases, checker, validator) updated multiple times in quick succession.
- Submission code reference is invalid or missing (should fail gracefully).
- Standing calculation fails during rejudge (should log error, not block rejudge).
- Admin rejudges while contest owner simultaneously rejudges same submissions.
- Rejudge request for a problem with status DRAFT (should be allowed - submissions exist from when problem was PUBLISHED).
- Network interruption during asynchronous rejudging.
- Only checker updated (no test case changes) - submissions should still be rejudged.
- Only validator updated (no test case changes) - submissions should still be rejudged.
- Multiple components updated simultaneously (test cases + checker) - single timestamp update.

---

## API Contract

### POST /contests/{contestId}/problems/{slug}/rejudge

Rejudge all affected submissions for a problem within a specific contest. Only the contest owner or Leads of the group can trigger this.

> **Important**: This endpoint only works during active contests. Only submissions where `submittedAt < problemJudgingUpdatedAt` are rejudged. The Standing is automatically updated after each submission is rejudged (only for submissions where submittedAt <= contest.endTime). The authenticated user must be the contest owner or a Lead of the group.

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
  "message": "Only the contest owner or Leads of the group can rejudge submissions in this contest"
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

Manually rejudge a specific submission. Contestant can rejudge their own submissions outside of active contests or in postcompetition phase.

> **Important**: 
> - Cannot rejudge own submissions in active contests (submittedAt <= contest.endTime)
> - Can rejudge own submissions in postcompetition (submittedAt > contest.endTime AND contest.enablePostContest = true)
> - Can rejudge own submissions outside of contests (practice submissions)
> - Can rejudge own submissions in finished contests (submittedAt <= contest.endTime)
> - Only the submission owner can rejudge their submission
> - Only works when `submittedAt < problemJudgingUpdatedAt`
> - Standing is NOT updated for postcompetition submissions

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
  "message": "This submission does not need rejudging. Judging components have not been updated since submission."
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
  "error": "CANNOT_REJUDGE_IN_ACTIVE_CONTEST",
  "message": "Cannot rejudge own submissions in active contests. Please contact the contest owner or Leads to rejudge all affected submissions in the contest."
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

> **Important**: Only users with ADMIN role can use this endpoint. Rejudges all submissions where `submittedAt < problemJudgingUpdatedAt`. Standing is updated only for active contests, not for finished contests.

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
- **FR-001**: The system MUST determine if a submission needs rejudging by comparing `submittedAt < problemJudgingUpdatedAt`.
- **FR-002**: The system MUST automatically update `problemJudgingUpdatedAt` timestamp whenever any judging component is uploaded via `POST /problems/{slug}/files`:
  - `fileType=testCases` (test cases ZIP)
  - `fileType=checker` (custom output checker)
  - `fileType=validator` (input validator)
- **FR-003**: The system MUST allow contest owners and Leads of the group to rejudge all affected submissions in active contests.
- **FR-004**: The system MUST allow contestants to manually rejudge their own submissions outside of active contests or in postcompetition phase.
- **FR-005**: The system MUST allow admins to rejudge submissions at global, contest, or individual submission level.

**Rejudging Process**
- **FR-006**: The system MUST change submission verdict to PENDING when rejudging is initiated.
- **FR-007**: The system MUST process all rejudging asynchronously (non-blocking).
- **FR-008**: The system MUST execute the submission code against all current test cases during rejudging.
- **FR-009**: The system MUST update the submission status based on rejudging results (ACCEPTED, WRONG_ANSWER, TIME_LIMIT_EXCEEDED, MEMORY_LIMIT_EXCEEDED, RUNTIME_EXCEPTION, COMPILATION_ERROR, PRESENTATION_ERROR).
- **FR-010**: The system MUST NOT maintain a history of previous verdicts.

**Standing Updates**
- **FR-011**: The system MUST automatically recalculate and update Standing after each submission is rejudged in an ACTIVE contest.
- **FR-011.1**: The system MUST update Standing only for submissions where `submittedAt <= contest.endTime` (not postcompetition submissions).
- **FR-012**: The system MUST NOT update Standing for rejudged submissions in FINISHED contests (standing is frozen).
- **FR-012.1**: The system MUST NOT update Standing for rejudged submissions in postcompetition phase (submittedAt > contest.endTime).
- **FR-013**: The system MUST update Standing only for contests where `currentTime` is between `startTime` and `endTime` at the moment the rejudge completes.
- **FR-014**: If a submission was made during an active contest but the contest finishes before rejudging completes, the Standing MUST NOT be updated.

**Permissions**
- **FR-015**: The system MUST allow the contest owner and Leads of the group to rejudge submissions in their contest.
- **FR-016**: The system MUST allow contestants to rejudge only their own submissions.
- **FR-017**: The system MUST prevent contestants from rejudging their own submissions in active contests (submittedAt <= contest.endTime).
- **FR-017.1**: The system MUST allow contestants to rejudge their own submissions in postcompetition (submittedAt > contest.endTime AND contest.enablePostContest = true).
- **FR-018**: The system MUST allow only users with ADMIN role to use admin rejudge endpoints.
- **FR-019**: The system MUST allow admins to rejudge any submission regardless of owner or context.

**Validation**
- **FR-020**: The system MUST validate that the problem exists and is part of the specified contest (for contest-scoped rejudge).
- **FR-021**: The system MUST validate that the contest owner or a Lead of the group is the authenticated user (for contest rejudge).
- **FR-022**: The system MUST validate that submissions belong to the authenticated user (for contestant manual rejudge).
- **FR-023**: The system MUST reject contestant manual rejudge requests for their own submissions in active contests (submittedAt <= contest.endTime).
- **FR-023.1**: The system MUST allow contestant manual rejudge requests for their own submissions in postcompetition (submittedAt > contest.endTime AND contest.enablePostContest = true).
- **FR-024**: The system MUST allow rejudging submissions of problems with status DRAFT (submissions may exist from when problem was PUBLISHED).

**General**
- **FR-025**: The system MUST NOT return internal IDs in responses (except where needed as identifiers).
- **FR-026**: The system MUST return the count of submissions queued for rejudging.
- **FR-027**: The system MUST return validation and authorization errors with consistent structure.

### Key Entities

- **Problem**: Extended with rejudging-related timestamp.  
  Key attributes:
  - `slug` (string, unique, user-provided, 3-70 chars, immutable)
  - `problemJudgingUpdatedAt` (timestamp, updated when any judging component is uploaded: test cases, checker, or validator)
  - `status` (enum: `DRAFT` | `PUBLISHED`)
  - `accessibility` (enum: `PUBLIC` | `PRIVATE`, default: `PRIVATE`)
  - Other attributes from Create Problem spec

> **Judging Components** (components that affect submission verdicts):
> - **Test Cases**: Input/output files used to evaluate submissions
> - **Checker**: Custom program that validates submission output (optional, default: exact match)
> - **Validator**: Program that validates test case inputs conform to constraints (optional)
>
> When ANY of these components is uploaded/replaced, `problemJudgingUpdatedAt` is updated.

- **Submission**: Represents a contestant's solution attempt.  
  Key attributes:
  - `id` (string, UUID)
  - `contestant_id` (string, UUID, FK to User)
  - `problem_id` (string, UUID, FK to Problem)
  - `contest_id` (string, UUID, FK to Contest, nullable - null for practice submissions)
  - `filePath` (string, storage path/key to solution file)
  - `fileHash` (string, SHA256 hash for duplicate detection)
  - `status` (enum: `PENDING` | `RUNNING` | `ACCEPTED` | `WRONG_ANSWER` | `TIME_LIMIT_EXCEEDED` | `MEMORY_LIMIT_EXCEEDED` | `RUNTIME_EXCEPTION` | `COMPILATION_ERROR` | `PRESENTATION_ERROR`)
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

### Submission States

| Status | Description |
|--------|-------------|
| PENDING | Submission is queued for judging (not yet started) |
| RUNNING | Submission is currently being judged |
| ACCEPTED | Solution passed all test cases |
| WRONG_ANSWER | Solution produced incorrect output |
| TIME_LIMIT_EXCEEDED | Solution exceeded time limit |
| MEMORY_LIMIT_EXCEEDED | Solution exceeded memory limit |
| RUNTIME_EXCEPTION | Solution crashed during execution |
| COMPILATION_ERROR | Solution failed to compile |
| PRESENTATION_ERROR | Solution output format is incorrect |
| COMPILATION_ERROR | Solution failed to compile |

### Permission Matrix

| Action | Contest Owner | Lead | Contestant (own) | Admin | Other Users |
|--------|---------------|------|------------------|-------|-------------|
| Rejudge all in active contest | ✅ (own contest) | ✅ (group contests) | ❌ | ✅ | ❌ |
| Rejudge own submission (outside active contest) | ✅ | ✅ | ✅ | ✅ | ❌ |
| Rejudge own submission (in postcompetition) | ✅ | ✅ | ✅ | ✅ | ❌ |
| Rejudge own submission (in active contest) | ❌ | ❌ | ❌ | ✅ | ❌ |
| Rejudge all for problem (global) | ❌ | ❌ | ❌ | ✅ | ❌ |
| Rejudge all in specific contest | ❌ | ❌ | ❌ | ✅ | ❌ |
| Rejudge specific submission | ✅ (own only) | ✅ (own only) | ✅ (own only) | ✅ | ❌ |

### Rejudge Decision Logic

```
Function: needsRejudge(submission, problem)
  return submission.submittedAt < problem.problemJudgingUpdatedAt

Function: shouldUpdateStanding(submission, contest)
  if contest is null:
    return false
  // Postcompetition submissions do NOT affect standing
  if submission.submittedAt > contest.endTime:
    return false
  currentTime = now()
  isActive = currentTime >= contest.startTime AND currentTime <= contest.endTime
  return isActive
```

### When problemJudgingUpdatedAt is Updated

The `problemJudgingUpdatedAt` timestamp is updated when ANY of the following files are uploaded via `POST /problems/{slug}/files`:

| File Type | Triggers Update | Reason |
|-----------|-----------------|--------|
| `testCases` | ✅ Yes | Changes input/output used for evaluation |
| `checker` | ✅ Yes | Changes how output correctness is determined |
| `validator` | ✅ Yes | Changes input validation rules (may indicate constraint changes) |
| `solution` | ❌ No | Solutions are for validation during publish, not for judging submissions |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Contest owners and Leads can rejudge all affected submissions in their active contests via `POST /contests/{contestId}/problems/{slug}/rejudge` with HTTP 200.
- **SC-002**: Contestants can manually rejudge their own submissions outside active contests or in postcompetition via `POST /submissions/{submissionId}/rejudge` with HTTP 200.
- **SC-002.1**: Contestants cannot rejudge their own submissions in active contests (submittedAt <= contest.endTime) - HTTP 403.
- **SC-002.2**: Contestants can rejudge their own submissions in postcompetition (submittedAt > contest.endTime AND contest.enablePostContest = true) - HTTP 200.
- **SC-003**: Admins can rejudge all affected submissions globally via `POST /admin/problems/{slug}/rejudge` with HTTP 200.
- **SC-004**: Admins can rejudge all affected submissions in a specific contest via `POST /admin/problems/{slug}/rejudge?contestId={contestId}` with HTTP 200.
- **SC-005**: Admins can rejudge a specific submission via `POST /admin/submissions/{submissionId}/rejudge` with HTTP 200.
- **SC-006**: Submissions are correctly identified for rejudging when `submittedAt < problemJudgingUpdatedAt`.
- **SC-007**: Submission verdict changes to PENDING when rejudging is initiated.
- **SC-008**: Standing is automatically updated after rejudging for submissions in active contests (submittedAt <= contest.endTime).
- **SC-008.1**: Standing is NOT updated for rejudged submissions in postcompetition (submittedAt > contest.endTime).
- **SC-009**: Standing is NOT updated after rejudging for submissions in finished contests.
- **SC-010**: Contestants cannot rejudge their own submissions in active contests (submittedAt <= contest.endTime) - HTTP 403.
- **SC-010.1**: Contestants can rejudge their own submissions in postcompetition - HTTP 200.
- **SC-011**: Contest owners and Leads can rejudge submissions in their contests - HTTP 200.
- **SC-012**: Only admins can use admin rejudge endpoints (HTTP 403 for non-admins).
- **SC-013**: Users cannot rejudge other users' submissions (except admins).
- **SC-014**: All rejudging is processed asynchronously.
- **SC-015**: `problemJudgingUpdatedAt` is automatically updated when test cases are uploaded.
- **SC-016**: `problemJudgingUpdatedAt` is automatically updated when checker is uploaded.
- **SC-017**: `problemJudgingUpdatedAt` is automatically updated when validator is uploaded.

---

## Optional Notes

- **Asynchronous Processing**: All rejudging should be processed through a queue system to handle large volumes without blocking API responses.
- **Idempotency**: Consider making rejudge requests idempotent - if a submission is already queued or being rejudged, subsequent requests should not create duplicate jobs.
- **Batch Operations**: For admin operations affecting many submissions, consider implementing progress tracking or webhook notifications.
- **Standing Calculation**: The detailed algorithm for Standing calculation will be defined in a separate Standing/Ranking spec.
- **File Storage**: Submission code is stored in cloud storage with path `{problemId}/{userId}/{contestId}/{submissionId}.{ext}` for contest submissions or `{problemId}/{userId}/general/{submissionId}.{ext}` for practice submissions. See Submit Solution spec for details.
- **Monitoring**: Consider adding logging and metrics for rejudge operations to track performance and identify issues.
- **Rate Limiting**: Consider rate limiting rejudge requests to prevent abuse, especially for the manual contestant rejudge endpoint.
- **Single Timestamp Design**: Using a single `problemJudgingUpdatedAt` timestamp instead of separate timestamps for each component (test cases, checker, validator) simplifies the logic and ensures all submissions are rejudged when any judging-related change occurs.
- **Validator Inclusion**: Although the validator is primarily used during problem publication to validate test inputs, changes to the validator may indicate that problem constraints have changed, which could affect how submissions should be evaluated.
- **Related specs**:
  - Create Problem: Initial problem creation
  - Update Problem: Uploading judging components (triggers problemJudgingUpdatedAt update)
  - Publish Problem: Problem validation and publication
  - Submit Solution: Creating submissions (to be defined)
  - Contest Management: Creating and managing contests (to be defined)
  - Standing/Ranking: Calculating and displaying contest rankings (to be defined)

