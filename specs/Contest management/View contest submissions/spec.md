# Feature Specification: View Contest Submissions

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View contest submissions during competition (Priority: P1)

As a contest participant, I want to view all submissions in the contest so that I can see the progress and verdicts of other participants.

**Why this priority**: Essential for participants to understand the competition dynamics and see which problems others are solving. This is a standard feature in competitive programming platforms.

**Independent Test**: This user story can be tested independently by consuming the `GET /api/groups/{groupId}/contests/{contestId}/submissions` endpoint with valid participant authentication, validating that submissions are returned with verdict information only (no execution details or source code during competition).

**Acceptance Scenarios**:

1. **Scenario**: Participant views submissions during ACTIVE contest
   * **Given** a contest is ACTIVE (startTime <= currentTime <= endTime)
   * **And** the authenticated user is registered to the contest
   * **When** they request the contest submissions
   * **Then** the system returns all submissions in the contest
   * **And** each submission shows: problem, submitter, language, submittedAt, status (verdict)
   * **And** does NOT show: executionTime, memoryUsed, sourceCode
   * **And** submissions are ordered by submittedAt DESC (most recent first)

2. **Scenario**: Participant views submissions during freeze period
   * **Given** a contest is ACTIVE and in freeze period (currentTime > endTime - freezeMinutes)
   * **And** the authenticated user is registered to the contest
   * **When** they request the contest submissions
   * **Then** submissions after freeze time show status as "?" (pending)
   * **And** submissions before freeze time show actual verdict

3. **Scenario**: Lead views submissions during freeze period with realtime flag
   * **Given** a contest is ACTIVE and in freeze period
   * **And** the authenticated user is a Lead of the group
   * **When** they request submissions with `?realtime=true`
   * **Then** all submissions show actual verdicts (no "?" for frozen submissions)

4. **Scenario**: Non-participant attempts to view contest submissions
   * **Given** a contest exists
   * **And** the authenticated user is NOT registered to the contest
   * **And** the user is NOT a Lead or Admin
   * **When** they attempt to view contest submissions
   * **Then** the system rejects with 403 Forbidden (NOT_REGISTERED)

5. **Scenario**: Filter submissions by problem
   * **Given** a contest has multiple problems
   * **And** the authenticated user is registered
   * **When** they request submissions with `?problemSlug=sum-of-two-numbers`
   * **Then** only submissions for that problem are returned

6. **Scenario**: Filter submissions by user (individual)
   * **Given** a contest has multiple participants
   * **And** user john_doe is registered individually
   * **And** the authenticated user is registered
   * **When** they request submissions with `?nickname=john_doe`
   * **Then** only submissions by john_doe are returned

7. **Scenario**: Filter submissions by user (team member)
   * **Given** a contest has team registrations
   * **And** user alice is part of Team Alpha
   * **And** the authenticated user is registered
   * **When** they request submissions with `?nickname=alice`
   * **Then** all submissions by Team Alpha are returned (regardless of which member submitted)

8. **Scenario**: Filter submissions by phase (competition only)
   * **Given** a contest has submissions during and after endTime
   * **And** the authenticated user is registered
   * **When** they request submissions with `?phase=competition`
   * **Then** only submissions where submittedAt <= endTime are returned

9. **Scenario**: View team submissions
   * **Given** a contest has team registrations
   * **And** Team Alpha (members: alice, bob, carol) is registered
   * **And** alice submits a solution
   * **And** the authenticated user is registered
   * **When** they request the contest submissions
   * **Then** the submission shows Team Alpha as submitter (not alice individually)
   * **And** if showTeamMembers = true, member names are included

10. **Scenario**: Contest deleted - submissions not accessible
    * **Given** a contest existed and had submissions
    * **And** the contest was deleted (submissions preserved with contest_id = NULL)
    * **When** a user attempts to access submissions via the contest endpoint
    * **Then** the system rejects with 404 Not Found (contest not found)

---

### User Story 2 – View contest submissions in postcompetition (Priority: P1)

As a contest participant, I want to view detailed submission information after the contest ends so that I can learn from other solutions and analyze performance.

**Why this priority**: Critical for learning and post-contest analysis. Participants need to see execution details to understand performance differences.

**Independent Test**: This user story can be tested independently by consuming the same endpoint after contest endTime, validating that execution details (time, memory) are now visible but source code still respects visibility settings.

**Acceptance Scenarios**:

1. **Scenario**: Participant views submissions after contest ends
   * **Given** a contest is FINISHED (currentTime > endTime)
   * **And** the authenticated user was registered to the contest
   * **When** they request the contest submissions
   * **Then** the system returns all submissions with full details
   * **And** each submission shows: problem, submitter, language, submittedAt, status, executionTime, memoryUsed
   * **And** does NOT include sourceCode (must use individual submission endpoint to view code)

2. **Scenario**: Participant views submissions during postcompetition phase
   * **Given** a contest is FINISHED and enablePostContest = true
   * **And** the authenticated user was registered
   * **When** they request submissions with `?phase=postcompetition`
   * **Then** only submissions where submittedAt > endTime are returned
   * **And** full details (verdict, time, memory) are visible

3. **Scenario**: Filter submissions by phase (all)
   * **Given** a contest is FINISHED
   * **And** the authenticated user was registered
   * **When** they request submissions with `?phase=all`
   * **Then** all submissions (competition + postcompetition) are returned

4. **Scenario**: View source code of PUBLIC submission
   * **Given** a contest is FINISHED
   * **And** a submission has visibility = PUBLIC
   * **When** any registered participant requests that submission via `GET /api/submissions/{id}`
   * **Then** the source code is included in the individual submission response

5. **Scenario**: Cannot view source code of PRIVATE submission
   * **Given** a contest is FINISHED
   * **And** a submission has visibility = PRIVATE
   * **And** the requesting user is NOT the author, admin, or lead
   * **When** they request that submission via `GET /api/submissions/{id}`
   * **Then** the system rejects with 403 Forbidden

---

### User Story 3 – Lead/Admin views all contest submissions (Priority: P1)

As a Lead or Admin, I want to view all contest submissions with full details at any time so that I can monitor the contest and assist participants.

**Why this priority**: Essential for contest management, debugging, and providing support to participants.

**Independent Test**: This user story can be tested independently by consuming the endpoint with Lead/Admin authentication, validating that all details are visible regardless of contest phase.

**Acceptance Scenarios**:

1. **Scenario**: Lead views submissions during ACTIVE contest
   * **Given** a contest is ACTIVE
   * **And** the authenticated user is a Lead of the group
   * **When** they request the contest submissions
   * **Then** all submissions are returned with full details (verdict, time, memory)
   * **And** source code is NOT included (must use individual submission endpoint)

2. **Scenario**: Admin views submissions in any contest
   * **Given** a contest exists in any group
   * **And** the authenticated user is Admin
   * **When** they request the contest submissions
   * **Then** all submissions are returned with full details

3. **Scenario**: Lead views submissions with realtime during freeze
   * **Given** a contest is in freeze period
   * **And** the authenticated user is a Lead
   * **When** they request submissions with `?realtime=true`
   * **Then** actual verdicts are shown (no "?" for frozen submissions)

---

### Edge Cases

* Contest with no submissions - return empty array with 200 OK
* Contest with thousands of submissions - pagination handles large datasets
* Submissions during exact freeze boundary (submittedAt == endTime - freezeMinutes) - treated as before freeze
* User registered as individual AND as team member - should not happen (validation prevents this)
* Team submission attribution - always shows team info, not individual submitter
* Team member names visibility - controlled by contest.showTeamMembers setting
* Concurrent submissions at same timestamp - stable sort order by submission ID
* Filter by non-existent problem slug - return empty array (not error)
* Filter by non-existent nickname - return empty array (not error)
* Multiple filters combined (problem + user + phase) - AND logic applied
* Contest deleted - endpoint returns 404 Not Found (submissions orphaned with contest_id = NULL)
* Orphaned submissions - not accessible via this endpoint, only via user's submission history

---

## API Contract

### GET /api/groups/{groupId}/contests/{contestId}/submissions

Retrieve the list of submissions for a contest.

> **Important**: 
> - **Source code is NEVER included in this endpoint** - not during competition, not after
> - To view source code, use the individual submission endpoint: `GET /api/submissions/{id}`
> - During competition (ACTIVE): Only verdict is visible to participants
> - During freeze: Submissions after freeze show as "?" unless `realtime=true` for Leads/Admin
> - After competition (FINISHED): Full details (verdict, time, memory) visible

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Query Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| phase | string | No | Filter by phase: `competition` (submittedAt <= endTime), `postcompetition` (submittedAt > endTime), `all` (default: all) |
| problemSlug | string | No | Filter by problem slug |
| nickname | string | No | Filter by participant nickname. For individuals: shows their submissions. For team members: shows all submissions by their team |
| page | integer | No | Page number for pagination (default: 1) |
| limit | integer | No | Items per page (default: 50, max: 100) |
| realtime | boolean | No | Show actual verdicts during freeze (only for Leads/Admin, default: false) |

**Responses**:

#### 200 OK - During ACTIVE contest (before freeze)
Participant view - only verdict visible.

```json
{
  "contest": {
    "id": "contest-uuid",
    "name": "Weekly Contest #15",
    "status": "ACTIVE",
    "startTime": "2026-02-19T14:00:00Z",
    "endTime": "2026-02-19T19:00:00Z",
    "freezeMinutes": 60,
    "freezeTime": "2026-02-19T18:00:00Z"
  },
  "submissions": [
    {
      "id": "submission-uuid-1",
      "problem": {
        "slug": "sum-of-two-numbers",
        "title": "Sum of Two Numbers",
        "order": 1
      },
      "submittedBy": {
        "type": "INDIVIDUAL",
        "nickname": "john_doe",
        "name": "John Doe"
      },
      "language": "cpp20",
      "submittedAt": "2026-02-19T15:30:00Z",
      "status": "ACCEPTED"
    },
    {
      "id": "submission-uuid-2",
      "problem": {
        "slug": "longest-path",
        "title": "Longest Path in DAG",
        "order": 2
      },
      "submittedBy": {
        "type": "TEAM",
        "teamId": "team-uuid",
        "teamName": "Team Alpha",
        "members": ["alice", "bob", "carol"]
      },
      "language": "python310",
      "submittedAt": "2026-02-19T16:15:00Z",
      "status": "WRONG_ANSWER"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 127,
    "totalPages": 3,
    "hasNextPage": true,
    "hasPrevPage": false
  }
}
```

#### 200 OK - During freeze period
Submissions after freeze time show status as "?".

```json
{
  "contest": {
    "id": "contest-uuid",
    "name": "Weekly Contest #15",
    "status": "ACTIVE",
    "inFreeze": true,
    "freezeTime": "2026-02-19T18:00:00Z"
  },
  "submissions": [
    {
      "id": "submission-uuid-3",
      "problem": {
        "slug": "sum-of-two-numbers",
        "title": "Sum of Two Numbers",
        "order": 1
      },
      "submittedBy": {
        "type": "INDIVIDUAL",
        "nickname": "jane_smith",
        "name": "Jane Smith"
      },
      "language": "java17",
      "submittedAt": "2026-02-19T18:30:00Z",
      "status": "?"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 15,
    "totalPages": 1,
    "hasNextPage": false,
    "hasPrevPage": false
  }
}
```

#### 200 OK - After contest ends (FINISHED)
Full details visible (verdict, time, memory).

```json
{
  "contest": {
    "id": "contest-uuid",
    "name": "Weekly Contest #15",
    "status": "FINISHED",
    "startTime": "2026-02-19T14:00:00Z",
    "endTime": "2026-02-19T19:00:00Z"
  },
  "submissions": [
    {
      "id": "submission-uuid-1",
      "problem": {
        "slug": "sum-of-two-numbers",
        "title": "Sum of Two Numbers",
        "order": 1
      },
      "submittedBy": {
        "type": "INDIVIDUAL",
        "nickname": "john_doe",
        "name": "John Doe"
      },
      "language": "cpp20",
      "submittedAt": "2026-02-19T15:30:00Z",
      "judgedAt": "2026-02-19T15:30:05Z",
      "status": "ACCEPTED",
      "executionTime": 45,
      "memoryUsed": 12,
      "phase": "competition"
    },
    {
      "id": "submission-uuid-4",
      "problem": {
        "slug": "matrix-multiplication",
        "title": "Matrix Chain Multiplication",
        "order": 3
      },
      "submittedBy": {
        "type": "INDIVIDUAL",
        "nickname": "john_doe",
        "name": "John Doe"
      },
      "language": "python310",
      "submittedAt": "2026-02-19T19:30:00Z",
      "judgedAt": "2026-02-19T19:30:08Z",
      "status": "TIME_LIMIT_EXCEEDED",
      "executionTime": 4500,
      "memoryUsed": 256,
      "phase": "postcompetition"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 127,
    "totalPages": 3,
    "hasNextPage": true,
    "hasPrevPage": false
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| contest | object | Contest information |
| contest.id | UUID | Contest identifier |
| contest.name | string | Contest name |
| contest.status | enum | SCHEDULED, ACTIVE, or FINISHED |
| contest.inFreeze | boolean | Whether contest is in freeze period (only during ACTIVE) |
| contest.freezeTime | timestamp | When freeze started (only if inFreeze = true) |
| submissions | array | List of submissions |
| submissions[].id | UUID | Submission identifier |
| submissions[].problem | object | Problem information |
| submissions[].submittedBy | object | Submitter information (individual or team) |
| submissions[].language | string | Language identifier |
| submissions[].submittedAt | timestamp | When submission was created |
| submissions[].judgedAt | timestamp | When judging completed (only after contest ends) |
| submissions[].status | enum | Verdict (or "?" during freeze) |
| submissions[].executionTime | integer | Execution time in ms (only after contest ends) |
| submissions[].memoryUsed | integer | Memory used in MiB (only after contest ends) |
| submissions[].phase | enum | `competition` or `postcompetition` (only after contest ends) |
| pagination | object | Pagination information |
| pagination.page | integer | Current page number (1-indexed) |
| pagination.limit | integer | Items per page |
| pagination.total | integer | Total number of submissions matching filters |
| pagination.totalPages | integer | Total number of pages |
| pagination.hasNextPage | boolean | Whether there is a next page |
| pagination.hasPrevPage | boolean | Whether there is a previous page |

**Submission Status Values**:

During competition:
- `PENDING`, `RUNNING`, `ACCEPTED`, `WRONG_ANSWER`, `TIME_LIMIT_EXCEEDED`, `MEMORY_LIMIT_EXCEEDED`, `RUNTIME_EXCEPTION`, `COMPILATION_ERROR`, `PRESENTATION_ERROR`, `SYSTEM_ERROR`
- `?` (during freeze period for submissions after freeze time)

#### 400 Bad Request
Invalid query parameters.

```json
{
  "error": "INVALID_PARAMETER",
  "message": "Invalid phase value. Must be: competition, postcompetition, or all"
}
```

```json
{
  "error": "INVALID_PARAMETER",
  "message": "Limit must be between 1 and 100"
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
User not registered to contest.

```json
{
  "error": "NOT_REGISTERED",
  "message": "You must be registered to this contest to view submissions"
}
```

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only Leads and Admins can use realtime=true parameter"
}
```

#### 404 Not Found
Contest not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Contest not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Visibility During Competition (ACTIVE)**
* **FR-VCS-001**: During ACTIVE contests, registered participants MUST be able to view all submissions in the contest.
* **FR-VCS-002**: During ACTIVE contests, participants MUST only see: problem, submitter, language, submittedAt, and status (verdict).
* **FR-VCS-003**: During ACTIVE contests, participants MUST NOT see: executionTime, memoryUsed, or sourceCode.
* **FR-VCS-004**: During freeze period, submissions after freeze time MUST show status as "?" for regular participants.
* **FR-VCS-005**: Leads and Admin with `realtime=true` MUST see actual verdicts during freeze period.

**Visibility After Competition (FINISHED)**
* **FR-VCS-006**: After contest ends, registered participants MUST see full details: verdict, executionTime, memoryUsed.
* **FR-VCS-007**: Source code MUST NEVER be included in the list response (not during competition, not after).
* **FR-VCS-008**: To view source code, users MUST use the individual submission endpoint (`GET /api/submissions/{id}`).
* **FR-VCS-008.1**: Source code access via individual endpoint MUST respect visibility settings (PUBLIC/PRIVATE).

**Permissions**
* **FR-VCS-009**: Only registered participants, Leads, and Admin can view contest submissions.
* **FR-VCS-010**: Non-registered users MUST receive 403 Forbidden.
* **FR-VCS-011**: Leads and Admin MUST see full details at any time (during and after contest).
* **FR-VCS-012**: Only Leads and Admin can use `realtime=true` parameter.

**Filtering**
* **FR-VCS-013**: System MUST support filtering by phase: `competition`, `postcompetition`, `all`.
* **FR-VCS-014**: System MUST support filtering by problem slug.
* **FR-VCS-015**: System MUST support filtering by participant nickname.
* **FR-VCS-015.1**: When filtering by nickname for an individual participant, system MUST return only their submissions.
* **FR-VCS-015.2**: When filtering by nickname for a team member, system MUST return all submissions by their team (regardless of which member submitted).
* **FR-VCS-016**: Multiple filters MUST be combinable (AND logic).
* **FR-VCS-017**: Invalid filter values MUST return empty array (not error).

**Pagination**
* **FR-VCS-018**: System MUST support pagination with page and limit parameters.
* **FR-VCS-019**: Default limit MUST be 50, maximum limit MUST be 100.
* **FR-VCS-020**: System MUST return total count and total pages in response.

**Sorting**
* **FR-VCS-021**: Submissions MUST be ordered by submittedAt DESC (most recent first).
* **FR-VCS-022**: Submissions at same timestamp MUST have stable sort order.

**Team Submissions**
* **FR-VCS-023**: Team submissions MUST show team information instead of individual submitter.
* **FR-VCS-024**: Team submissions MUST include team name and team ID.
* **FR-VCS-024.1**: If contest.showTeamMembers = true, team submissions MUST include member nicknames.
* **FR-VCS-024.2**: If contest.showTeamMembers = false, team submissions MUST NOT include member nicknames.

**Deleted Contests**
* **FR-VCS-025**: If a contest is deleted, the endpoint MUST return 404 Not Found.
* **FR-VCS-026**: Orphaned submissions (contest_id = NULL) MUST NOT be accessible via this endpoint.
* **FR-VCS-027**: Orphaned submissions remain accessible via user's submission history endpoints.

**Phase Identification**
* **FR-VCS-028**: After contest ends, each submission MUST include phase field: `competition` or `postcompetition`.
* **FR-VCS-029**: Phase is determined by: submittedAt <= endTime (competition) or submittedAt > endTime (postcompetition).

### Key Entities

* **Contest**: Existing entity with freeze configuration
  * `freezeMinutes` (integer, nullable) - Minutes before endTime to freeze standings/submissions

* **Submission**: Existing entity with contest association
  * `contestId` (UUID, nullable) - Contest the submission belongs to
  * `submittedAt` (timestamp) - When submission was created
  * `judgedAt` (timestamp, nullable) - When judging completed
  * `submittedBy` (UUID, FK to User) - Who physically submitted
  * `standingId` (UUID, nullable) - Who gets credit (userId or teamId)
  * `visibility` (enum: PUBLIC | PRIVATE) - Who can view source code (not used in list view)

> **Important**: For team submissions, `submittedBy` is always the user who pressed submit, but `standingId` is the teamId. The API response shows team information (not the individual submitter) to maintain team attribution in the contest context.

### Permission Matrix

| Viewer | During ACTIVE | During Freeze | After FINISHED | Source Code |
|--------|---------------|---------------|----------------|-------------|
| Participant | Verdict only | "?" after freeze | Full details | Only PUBLIC or own |
| Lead | Full details | Full details (realtime) | Full details | Only PUBLIC or own |
| Admin | Full details | Full details (realtime) | Full details | Only PUBLIC or own |
| Non-registered | ❌ 403 | ❌ 403 | ❌ 403 | ❌ 403 |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-VCS-001**: Registered participants can view contest submissions via `GET /api/groups/{groupId}/contests/{contestId}/submissions` with HTTP 200.
* **SC-VCS-002**: During ACTIVE contests, only verdict is visible to participants (no execution details).
* **SC-VCS-003**: During freeze period, submissions after freeze show status "?" for participants.
* **SC-VCS-004**: Leads/Admin with `realtime=true` see actual verdicts during freeze.
* **SC-VCS-005**: After contest ends, full details (verdict, time, memory) are visible.
* **SC-VCS-006**: Source code is NOT included in list view at any time.
* **SC-VCS-007**: Filtering by phase, problem, and user works correctly.
* **SC-VCS-008**: Pagination works with configurable page and limit.
* **SC-VCS-009**: Non-registered users receive HTTP 403.
* **SC-VCS-010**: Team submissions show team information correctly.
* **SC-VCS-011**: Phase field is included after contest ends.

---

## Optional Notes

* **Performance**: Consider caching submission lists for large contests
* **Real-time updates**: Consider WebSocket/SSE for live submission feed during contests
* **Export**: Future enhancement to export submissions as CSV/JSON
* **Statistics**: Future enhancement to show per-problem statistics (acceptance rate, average time)
* **Memory units**: The API returns memory in MiB for consistency with competitive programming standards. The database stores in KB, so conversion is needed (KB / 1024 = MiB).
* **Related specs**:
  * View submission: For viewing individual submission details with source code
  * View contest standings: For viewing contest rankings
  * Submit solution: For creating submissions
  * View contest: For contest details and problem list

