# Feature Specification: User Activity Dashboard

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View personal activity dashboard (Priority: P2)

As an authenticated user, I want to view my personal activity dashboard so that I can see my recent submissions, upcoming contests, problem-solving statistics, recent materials, streak information, global ranking, and recent contest results in one centralized view.

**Why this priority**: This feature significantly improves user experience by centralizing relevant information and facilitating personal progress tracking. It increases platform engagement by providing users with a comprehensive overview of their activity. However, it's not critical for basic system functionality as individual features (submissions, contests, etc.) are already accessible separately.

**Independent Test**: This user story can be tested independently by consuming the `GET /users/me/dashboard` endpoint with valid authentication, validating that the authenticated user's complete dashboard information is returned with all sections properly populated.

**Acceptance Scenarios**:

1. **Scenario**: Successful retrieval of complete dashboard
   - **Given** a user exists in the system with ACTIVE status and is authenticated
   - **And** the user has submissions, contest registrations, and group memberships
   - **When** they request their dashboard
   - **Then** the system returns complete dashboard information including:
     - Last 10 submissions with problem, verdict, language, date, execution time, and memory
     - Up to 3 upcoming contests (starting after current time) with name, start date, and duration
     - Up to 3 active contests (currently running) with name, start date, and duration
     - Total count of unique problems solved (ACCEPTED verdict only)
     - Last 10 materials from all user's groups published in last 30 days with title, group, publication date, and author
     - Current streak and maximum streak (consecutive days with at least one submission)
     - Global ranking position based on unique problems solved
     - Last 10 finished contests where user participated with position, problems solved, and penalty

2. **Scenario**: Dashboard for user with no activity
   - **Given** a newly created user with ACTIVE status and is authenticated
   - **And** the user has no submissions, no contest registrations, and no group memberships
   - **When** they request their dashboard
   - **Then** the system returns dashboard with empty/zero values:
     - Empty submissions array
     - Empty upcoming contests array
     - Empty active contests array
     - Zero problems solved
     - Empty materials array
     - Zero current streak and zero maximum streak
     - Ranking position (could be last or null if no problems solved)
     - Empty contest results array

3. **Scenario**: Dashboard with partial data
   - **Given** an authenticated user with some activity
   - **And** the user has 5 submissions but no contest registrations
   - **When** they request their dashboard
   - **Then** the system returns dashboard with:
     - 5 submissions in the array
     - Empty upcoming and active contests arrays
     - Problems solved count based on ACCEPTED submissions
     - Other sections populated based on available data

4. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a dashboard request is submitted
   - **Then** the system rejects the operation with 401 Unauthorized

5. **Scenario**: User not found
   - **Given** the token is valid but does not resolve an existing user
   - **When** a dashboard request is submitted
   - **Then** the system rejects with 404 Not Found

6. **Scenario**: Deactivated user attempts to view dashboard
   - **Given** a user with DEACTIVATED status attempts to authenticate
   - **When** they request their dashboard
   - **Then** the system rejects with 401 Unauthorized (authentication fails for deactivated users)

---

### Edge Cases

- User with more than 10 submissions (only last 10 should be returned).
- User registered to more than 3 upcoming contests (only 3 should be returned).
- User with more than 3 active contests (only 3 should be returned).
- User with materials from multiple groups published in last 30 days (max 10 most recent).
- User with more than 10 finished contest participations (only last 10 should be returned).
- User with broken streak (days without submissions).
- User with same ranking position as other users (tie handling).
- Contest that just started (transitions from upcoming to active).
- Contest that just ended (should appear in contest results).
- Material published exactly 30 days ago (boundary condition).
- User solves same problem multiple times (should count as 1 unique problem).
- User with submissions but no ACCEPTED verdicts (zero problems solved).
- Concurrent requests to dashboard endpoint.
- Very long problem names, contest names, or material titles in responses.
- Unicode characters in names and titles.

## API Contract

### GET /users/me/dashboard

Retrieve the authenticated user's personal activity dashboard with comprehensive statistics and recent activity.

> **Important**: User identity is resolved exclusively from the authentication token. This endpoint aggregates data from multiple sources (submissions, contests, groups, materials) to provide a centralized view of user activity and statistics.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Responses**:

#### 200 OK
Dashboard retrieved successfully.

```json
{
  "recentSubmissions": [
    {
      "id": 12345,
      "problemSlug": "two-sum",
      "problemTitle": "Two Sum",
      "verdict": "ACCEPTED",
      "language": "CPP",
      "submittedAt": "2026-02-19T10:30:00Z",
      "executionTime": 45,
      "memoryUsed": 2048
    },
    {
      "id": 12344,
      "problemSlug": "reverse-string",
      "problemTitle": "Reverse String",
      "verdict": "WRONG_ANSWER",
      "language": "PYTHON",
      "submittedAt": "2026-02-19T09:15:00Z",
      "executionTime": 32,
      "memoryUsed": 1024
    }
  ],
  "upcomingContests": [
    {
      "id": 101,
      "name": "Weekly Contest #45",
      "startDate": "2026-02-20T14:00:00Z",
      "durationMinutes": 180
    },
    {
      "id": 102,
      "name": "Monthly Challenge February",
      "startDate": "2026-02-25T10:00:00Z",
      "durationMinutes": 300
    }
  ],
  "activeContests": [
    {
      "id": 100,
      "name": "Daily Practice #12",
      "startDate": "2026-02-19T08:00:00Z",
      "durationMinutes": 240
    }
  ],
  "problemsSolved": 47,
  "recentMaterials": [
    {
      "id": 501,
      "title": "Dynamic Programming Basics",
      "groupId": 10,
      "groupName": "Advanced Algorithms",
      "publishedAt": "2026-02-15T12:00:00Z",
      "authorNickname": "coach_john"
    },
    {
      "id": 502,
      "title": "Graph Theory Introduction",
      "groupId": 11,
      "groupName": "Data Structures",
      "publishedAt": "2026-02-10T09:00:00Z",
      "authorNickname": "prof_smith"
    }
  ],
  "streak": {
    "current": 7,
    "maximum": 15
  },
  "ranking": {
    "position": 142,
    "totalUsers": 1523
  },
  "recentContestResults": [
    {
      "contestId": 99,
      "contestName": "Weekly Contest #44",
      "position": 23,
      "problemsSolved": 3,
      "penalty": 145
    },
    {
      "contestId": 98,
      "contestName": "Daily Practice #11",
      "position": 15,
      "problemsSolved": 5,
      "penalty": 89
    }
  ]
}
```

**Response when user has no activity**:
```json
{
  "recentSubmissions": [],
  "upcomingContests": [],
  "activeContests": [],
  "problemsSolved": 0,
  "recentMaterials": [],
  "streak": {
    "current": 0,
    "maximum": 0
  },
  "ranking": {
    "position": null,
    "totalUsers": 1523
  },
  "recentContestResults": []
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

**Dashboard Access**
- **FR-001**: The system MUST allow authenticated users with ACTIVE status to retrieve their personal dashboard via `GET /users/me/dashboard`.
- **FR-002**: User identity for `GET /users/me/dashboard` MUST be resolved exclusively from the authentication token.
- **FR-003**: The system MUST reject requests from unauthenticated users with 401 Unauthorized.
- **FR-004**: The system MUST reject requests from deactivated users with 401 Unauthorized.

**Recent Submissions**
- **FR-005**: The system MUST return the last 10 submissions ordered by submission date (most recent first).
- **FR-006**: Each submission MUST include: id, problemSlug, problemTitle, verdict, language, submittedAt, executionTime, memoryUsed.
- **FR-007**: If the user has fewer than 10 submissions, the system MUST return all available submissions.
- **FR-008**: If the user has no submissions, the system MUST return an empty array.

**Upcoming Contests**
- **FR-009**: The system MUST return up to 3 upcoming contests where the user is registered.
- **FR-010**: Upcoming contests are defined as contests with startDate after the current time.
- **FR-011**: Upcoming contests MUST be ordered by startDate (earliest first).
- **FR-012**: Each upcoming contest MUST include: id, name, startDate, durationMinutes.

**Active Contests**
- **FR-013**: The system MUST return up to 3 active contests where the user is registered.
- **FR-014**: Active contests are defined as contests currently running (current time between startDate and endDate).
- **FR-015**: Active contests MUST be ordered by startDate (earliest first).
- **FR-016**: Each active contest MUST include: id, name, startDate, durationMinutes.

**Problems Solved**
- **FR-017**: The system MUST return the total count of unique problems solved by the user.
- **FR-018**: A problem is considered "solved" only if the user has at least one submission with ACCEPTED verdict.
- **FR-019**: If the same problem is solved multiple times, it MUST count as 1 unique problem.
- **FR-020**: If the user has no ACCEPTED submissions, problemsSolved MUST be 0.

**Recent Materials**
- **FR-021**: The system MUST return up to 10 most recent materials from all groups the user belongs to.
- **FR-022**: Only materials published in the last 30 days MUST be included.
- **FR-023**: Only materials with PUBLISHED visibility MUST be included.
- **FR-024**: Materials MUST be ordered by publishedAt (most recent first).
- **FR-025**: Each material MUST include: id, title, groupId, groupName, publishedAt, authorNickname.
- **FR-026**: If no materials match the criteria, the system MUST return an empty array.

**Streak Information**
- **FR-027**: The system MUST calculate and return the user's current streak (consecutive days with at least one submission).
- **FR-028**: The system MUST calculate and return the user's maximum streak (longest consecutive days with at least one submission in history).
- **FR-029**: A day is considered "active" if the user has at least one submission on that day (any verdict).
- **FR-030**: If the user has no submissions, both current and maximum streak MUST be 0.
- **FR-031**: If the user has submissions but no consecutive days, current streak MUST be 1 (or 0 if last submission was not today/yesterday).

**Global Ranking**
- **FR-032**: The system MUST return the user's global ranking position based on unique problems solved.
- **FR-033**: The system MUST return the total number of users for context.
- **FR-034**: Users with the same number of problems solved MUST have the same ranking position.
- **FR-035**: If the user has solved 0 problems, position MAY be null or the last position.

**Recent Contest Results**
- **FR-036**: The system MUST return the last 10 finished contests where the user participated.
- **FR-037**: Only contests with FINISHED status MUST be included.
- **FR-038**: Contests MUST be ordered by end date (most recent first).
- **FR-039**: Each contest result MUST include: contestId, contestName, position (final ranking), problemsSolved, penalty.
- **FR-040**: If the user has not participated in any finished contests, the system MUST return an empty array.

**Response Structure**
- **FR-041**: The system MUST return all dashboard sections in a single response.
- **FR-042**: The system MUST return HTTP 200 with complete dashboard data for successful requests.
- **FR-043**: The system MUST return appropriate HTTP status codes (401, 404) with clear error messages for failures.

### Key Entities

- **User**: Registered person in the system.  
  Relevant attributes for this feature:
  - `id` (integer, primary key)
  - `nickname` (string, UNIQUE, lowercase)
  - `status` (enum: ACTIVE | DEACTIVATED)

- **Submission**: Code submission for a problem.  
  Relevant attributes:
  - `id` (integer, primary key)
  - `userId` (integer, foreign key)
  - `problemId` (integer, foreign key)
  - `verdict` (enum: ACCEPTED, WRONG_ANSWER, TIME_LIMIT_EXCEEDED, etc.)
  - `language` (enum: CPP, JAVA, PYTHON, etc.)
  - `submittedAt` (timestamp)
  - `executionTime` (integer, milliseconds)
  - `memoryUsed` (integer, KB)

- **Problem**: Programming problem.  
  Relevant attributes:
  - `id` (integer, primary key)
  - `slug` (string, UNIQUE)
  - `title` (string)

- **Contest**: Programming contest.  
  Relevant attributes:
  - `id` (integer, primary key)
  - `name` (string)
  - `startDate` (timestamp)
  - `durationMinutes` (integer)
  - `status` (enum: PENDING, ACTIVE, FINISHED, CANCELLED)

- **Contest_Participant**: User registration to contest.  
  Relevant attributes:
  - `userId` (integer, foreign key)
  - `contestId` (integer, foreign key)
  - `finalPosition` (integer, nullable)
  - `problemsSolved` (integer)
  - `penalty` (integer)

- **Group**: Study or practice group.  
  Relevant attributes:
  - `id` (integer, primary key)
  - `name` (string)

- **Group_Member**: User membership in group.  
  Relevant attributes:
  - `userId` (integer, foreign key)
  - `groupId` (integer, foreign key)

- **Material**: Educational material in a group.  
  Relevant attributes:
  - `id` (integer, primary key)
  - `title` (string)
  - `groupId` (integer, foreign key)
  - `authorId` (integer, foreign key)
  - `publishedAt` (timestamp, nullable)
  - `visibility` (enum: DRAFT, PUBLISHED)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Authenticated users with ACTIVE status can successfully retrieve their dashboard via `GET /users/me/dashboard` with HTTP 200.
- **SC-002**: Dashboard returns last 10 submissions ordered by most recent first.
- **SC-003**: Dashboard returns up to 3 upcoming contests (startDate > current time) ordered by earliest first.
- **SC-004**: Dashboard returns up to 3 active contests (currently running) ordered by earliest first.
- **SC-005**: Dashboard returns accurate count of unique problems solved (ACCEPTED verdict only).
- **SC-006**: Dashboard returns up to 10 most recent materials from user's groups published in last 30 days.
- **SC-007**: Dashboard returns current streak and maximum streak based on consecutive days with submissions.
- **SC-008**: Dashboard returns user's global ranking position based on unique problems solved.
- **SC-009**: Dashboard returns last 10 finished contests where user participated with position, problems solved, and penalty.
- **SC-010**: Dashboard returns empty arrays/zero values for users with no activity.
- **SC-011**: Unauthenticated requests return HTTP 401.
- **SC-012**: Requests for non-existent users return HTTP 404.
- **SC-013**: All dashboard sections are returned in a single response.
- **SC-014**: Response includes totalUsers count for ranking context.
