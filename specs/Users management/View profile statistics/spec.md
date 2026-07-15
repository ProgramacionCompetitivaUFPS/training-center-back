# Feature Specification: View Profile Statistics

**Created**: 2026-07-13

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View my own achievement statistics (Priority: P2)

As an authenticated user, I want to view my achievement statistics (problems solved, submission acceptance rate, contests participated, global ranking, and a breakdown of solved problems by topic) on my profile page so that I can track my long-term progress and see my strengths, separate from the day-to-day activity shown on my dashboard.

**Why this priority**: This centralizes reputation/achievement data that changes slowly (compared to the dashboard's recent-activity data) and is naturally viewed alongside identity information on the profile page. It's not critical for basic system functionality since the underlying data (submissions, contests) is already accessible separately.

**Independent Test**: This user story can be tested independently by consuming the `GET /users/me/stats` endpoint with valid authentication, validating that the authenticated user's statistics are returned with all sections properly populated.

**Acceptance Scenarios**:

1. **Scenario**: Successful retrieval of complete statistics
   - **Given** a user exists in the system with ACTIVE status and is authenticated
   - **And** the user has submissions and contest registrations
   - **When** they request their statistics
   - **Then** the system returns:
     - Total count of unique problems solved (ACCEPTED verdict only)
     - Total submission count (all verdicts, all time)
     - Total accepted submission count (all time; may exceed problemsSolved if a problem was solved more than once)
     - Total count of distinct contests the user registered to (individually or via team), regardless of status
     - Global ranking position and total ranked users, based on unique problems solved
     - Breakdown of unique solved problems by tag, ordered by solved count descending

2. **Scenario**: Statistics for user with no activity
   - **Given** a newly created user with ACTIVE status and is authenticated
   - **And** the user has no submissions and no contest registrations
   - **When** they request their statistics
   - **Then** the system returns zero/empty values:
     - Zero problems solved
     - Zero total submissions
     - Zero accepted submissions
     - Zero contests participated
     - Ranking position is null (or last), totalUsers reflects platform-wide active user count
     - Empty topic breakdown array

3. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a statistics request is submitted
   - **Then** the system rejects the operation with 401 Unauthorized

4. **Scenario**: User not found
   - **Given** the token is valid but does not resolve an existing user
   - **When** a statistics request is submitted
   - **Then** the system rejects with 404 Not Found

5. **Scenario**: Deactivated user attempts to view statistics
   - **Given** a user with DEACTIVATED status attempts to authenticate
   - **When** they request their statistics
   - **Then** the system rejects with 401 Unauthorized (authentication fails for deactivated users)

---

### Edge Cases

- User solves the same problem multiple times (counts once in `problemsSolved`, but every ACCEPTED submission counts in `acceptedSubmissions`).
- User submits to a problem but never gets ACCEPTED (contributes to `totalSubmissions`, not to `acceptedSubmissions` or `problemsSolved`).
- User registered to a contest that was later cancelled (still counts toward `contestsParticipated` — registration-based, not outcome-based).
- User registered to the same contest individually and later as part of a team (counts once).
- Two users tied on problems solved (same ranking position).
- Problem tagged with multiple topics (contributes to each tag's count independently).
- Problem with no tags (does not contribute to `topicStats`).
- Unicode characters in tag names.

## API Contract

### GET /users/me/stats

Retrieve the authenticated user's achievement/reputation statistics: slow-changing, identity-adjacent data shown on the profile page. Recent and upcoming activity (submissions feed, live/upcoming contests, recent materials) lives in [`GET /users/me/dashboard`](../User%20activity%20dashboard/spec.md).

> **Important**: User identity is resolved exclusively from the authentication token. This endpoint is intentionally separate from the dashboard endpoint because it aggregates comparatively expensive, slow-changing statistics (global ranking, full submission counts) that the dashboard page never displays and does not need to pay the cost of computing on every visit.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Responses**:

#### 200 OK
Statistics retrieved successfully.

```json
{
  "problemsSolved": 47,
  "totalSubmissions": 132,
  "acceptedSubmissions": 58,
  "contestsParticipated": 9,
  "ranking": {
    "position": 142,
    "totalUsers": 1523
  },
  "topicStats": [
    { "tag": "graphs", "solved": 15 },
    { "tag": "dp", "solved": 12 },
    { "tag": "arrays", "solved": 10 }
  ]
}
```

**Response when user has no activity**:
```json
{
  "problemsSolved": 0,
  "totalSubmissions": 0,
  "acceptedSubmissions": 0,
  "contestsParticipated": 0,
  "ranking": {
    "position": null,
    "totalUsers": 1523
  },
  "topicStats": []
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

#### 404 Not Found
User not found for the authenticated token.

```json
{
  "error": "NOT_FOUND",
  "message": "User not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Statistics Access**
- **FR-001**: The system MUST allow authenticated users with ACTIVE status to retrieve their own statistics via `GET /users/me/stats`.
- **FR-002**: User identity for `GET /users/me/stats` MUST be resolved exclusively from the authentication token.
- **FR-003**: The system MUST reject requests from unauthenticated users with 401 Unauthorized.
- **FR-004**: The system MUST reject requests from deactivated users with 401 Unauthorized.

**Problems Solved**
- **FR-005**: The system MUST return the total count of unique problems solved by the user (at least one ACCEPTED submission).
- **FR-006**: If the same problem is solved multiple times, it MUST count as 1 unique problem.

**Submission Counts**
- **FR-007**: The system MUST return the total number of submissions the user has ever made, regardless of verdict.
- **FR-008**: The system MUST return the total number of submissions with ACCEPTED verdict. This MAY exceed `problemsSolved` when the user solved the same problem more than once.
- **FR-009**: If the user has no submissions, both counts MUST be 0.

**Contests Participated**
- **FR-010**: The system MUST return the count of distinct contests the user is or was registered to, individually or as part of a team, regardless of the contest's current status (upcoming, active, finished, or cancelled).
- **FR-011**: A contest MUST be counted once even if the user is registered both individually and via a team.
- **FR-012**: If the user has no contest registrations, `contestsParticipated` MUST be 0.

**Global Ranking**
- **FR-013**: The system MUST return the user's global ranking position based on unique problems solved.
- **FR-014**: The system MUST return the total number of ranked (active) users for context.
- **FR-015**: Users with the same number of problems solved MUST have the same ranking position.
- **FR-016**: If the user has solved 0 problems, position MUST be null.

**Topic Breakdown**
- **FR-017**: The system MUST return, for each tag present on at least one problem the user solved, the count of unique problems solved carrying that tag.
- **FR-018**: A problem carrying multiple tags MUST contribute to the count of each of its tags.
- **FR-019**: Topic breakdown entries MUST be ordered by solved count descending.
- **FR-020**: If the user has solved no problems, `topicStats` MUST be an empty array.

**Response Structure**
- **FR-021**: The system MUST return all statistics sections in a single response.
- **FR-022**: The system MUST return HTTP 200 with complete statistics for successful requests.
- **FR-023**: The system MUST return appropriate HTTP status codes (401, 404) with clear error messages for failures.

### Key Entities

- **User**: Registered person in the system.
  Relevant attributes for this feature:
  - `id` (integer, primary key)
  - `status` (enum: ACTIVE | DEACTIVATED)

- **Submission**: Code submission for a problem.
  Relevant attributes:
  - `userId` (integer, foreign key)
  - `problemId` (integer, foreign key)
  - `verdict` (enum: ACCEPTED, WRONG_ANSWER, TIME_LIMIT_EXCEEDED, etc.)

- **Problem**: Programming problem.
  Relevant attributes:
  - `id` (integer, primary key)
  - `tags` (array of string) — used to compute `topicStats`

- **Contest_Participant**: User registration to contest, individually.
  Relevant attributes:
  - `userId` (integer, foreign key)
  - `contestId` (integer, foreign key)

- **Contest_Team_Participant**: User registration to contest via a team.
  Relevant attributes:
  - `selectedMembers` (array of userId, foreign key)
  - `contestId` (integer, foreign key)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Authenticated users with ACTIVE status can successfully retrieve their statistics via `GET /users/me/stats` with HTTP 200.
- **SC-002**: Statistics return accurate count of unique problems solved (ACCEPTED verdict only).
- **SC-003**: Statistics return accurate total and accepted submission counts.
- **SC-004**: Statistics return accurate count of distinct contests participated (individual + team registrations deduplicated).
- **SC-005**: Statistics return the user's global ranking position and total ranked users.
- **SC-006**: Statistics return a topic breakdown ordered by solved count descending.
- **SC-007**: Statistics return zero/empty values for users with no activity.
- **SC-008**: Unauthenticated requests return HTTP 401.
- **SC-009**: Requests for non-existent users return HTTP 404.
- **SC-010**: All statistics sections are returned in a single response.
