# Feature Specification: View Contest

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Contest Details (Priority: P1)

As a user, I want to view the details of a contest so that I can understand when it starts, how long it lasts, and what problems it contains.

**Why this priority**: Viewing contest details is essential for users to decide whether to participate and to access contest problems during the competition.

**Independent Test**: Authenticated request to the contest detail endpoint. Verify correct data is returned based on user's relationship with the group and registration status.

**Acceptance Scenarios**:

1. **Scenario**: Registered participant views ACTIVE contest

* **Given** a contest is ACTIVE
* **And** the authenticated user is registered to the contest
* **When** they request the contest details
* **Then** the system returns full contest information
* **And** includes the list of problems with titles, slugs, and limits
* **And** includes `isRegistered: true`

2. **Scenario**: Group member (not registered) views ACTIVE contest

* **Given** a contest is ACTIVE
* **And** the authenticated user is a member of the group but NOT registered
* **When** they request the contest details
* **Then** the system returns full contest information
* **And** includes the list of problems (can view but not submit)
* **And** includes `isRegistered: false`

3. **Scenario**: Non-member views ACTIVE contest in VISIBLE group

* **Given** a contest is ACTIVE in a VISIBLE group
* **And** the authenticated user is NOT a member of the group
* **When** they request the contest details
* **Then** the system returns basic contest information (name, description, times)
* **And** the problems list is EMPTY (cannot see problems)
* **And** includes `isRegistered: false`

4. **Scenario**: Anyone views SCHEDULED contest

* **Given** a contest is SCHEDULED (not yet started)
* **And** the authenticated user is a member or registered participant
* **When** they request the contest details
* **Then** the system returns contest information
* **And** the problems list is EMPTY (problems hidden until contest starts)

5. **Scenario**: Participant views FINISHED contest

* **Given** a contest is FINISHED
* **And** the authenticated user was registered to the contest
* **When** they request the contest details
* **Then** the system returns full contest information
* **And** includes the list of problems with all details

6. **Scenario**: Group member views FINISHED contest

* **Given** a contest is FINISHED
* **And** the authenticated user is a member of the group
* **When** they request the contest details
* **Then** the system returns full contest information
* **And** includes the list of problems

7. **Scenario**: Non-member views contest in NOT_VISIBLE group

* **Given** a contest exists in a NOT_VISIBLE group
* **And** the authenticated user is NOT a member of the group
* **When** they request the contest details
* **Then** the system rejects with 404 Not Found
* **And** does not reveal that the contest exists

8. **Scenario**: Admin views any contest

* **Given** any contest exists
* **And** the authenticated user is an Admin
* **When** they request the contest details
* **Then** the system returns full contest information including `locked` field
* **And** includes all problems regardless of contest state

9. **Scenario**: Lead views contest with locked field

* **Given** a contest exists in a group
* **And** the authenticated user is a Lead of that group
* **When** they request the contest details
* **Then** the system returns full contest information including `locked` field

10. **Scenario**: Non-Lead views contest without locked field

* **Given** a contest exists
* **And** the authenticated user is a Member (not Lead, not Admin)
* **When** they request the contest details
* **Then** the `locked` field is NOT included in the response

---

### User Story 2 - List Contests in Group (Priority: P1)

As a user, I want to see a list of contests in a group so that I can browse and choose which contests to participate in.

**Why this priority**: Users need to discover available contests within their groups.

**Independent Test**: Request to list contests endpoint with various filters. Verify correct pagination, filtering, and visibility rules.

**Acceptance Scenarios**:

1. **Scenario**: Group member lists all contests

* **Given** a group has multiple contests (past, active, future)
* **And** the authenticated user is a member of the group
* **When** they request the contests list without filters
* **Then** the system returns all contests sorted by startTime descending
* **And** includes pagination metadata

2. **Scenario**: Filter by status ACTIVE

* **Given** a group has contests with various statuses
* **When** the user requests contests filtered by `status=ACTIVE`
* **Then** only ACTIVE contests are returned

3. **Scenario**: Filter by status SCHEDULED

* **Given** a group has contests with various statuses
* **When** the user requests contests filtered by `status=SCHEDULED`
* **Then** only SCHEDULED (future) contests are returned

4. **Scenario**: Filter by status FINISHED

* **Given** a group has contests with various statuses
* **When** the user requests contests filtered by `status=FINISHED`
* **Then** only FINISHED (past) contests are returned

5. **Scenario**: Pagination with custom page size

* **Given** a group has 50 contests
* **When** the user requests with `page=2&limit=10`
* **Then** contests 11-20 are returned
* **And** pagination metadata shows total count and pages

6. **Scenario**: Non-member lists contests in VISIBLE group

* **Given** a VISIBLE group has contests
* **And** the authenticated user is NOT a member
* **When** they request the contests list
* **Then** the system returns the list of contests (basic info only)
* **And** each contest shows basic info but no problems

7. **Scenario**: Non-member attempts to list contests in NOT_VISIBLE group

* **Given** a NOT_VISIBLE group exists
* **And** the authenticated user is NOT a member
* **When** they request the contests list
* **Then** the system rejects with 404 Not Found

8. **Scenario**: Empty result when no contests match filter

* **Given** a group has no ACTIVE contests
* **When** the user filters by `status=ACTIVE`
* **Then** an empty array is returned with total=0

---

### Edge Cases

- Contest that just started (boundary: currentTime == startTime).
- Contest that just ended (boundary: currentTime == endTime).
- User who unregistered from contest after it started.
- Contest with no problems (edge case, should be valid).
- Very large number of contests in a group (pagination stress test).
- Concurrent request while contest status is changing.

---

## API Contract

### GET /contests/{contestId}

View details of a specific contest.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| contestId | UUID | Yes | The contest ID |

**Success Response (200 OK)**:

```json
{
  "id": "contest-123",
  "name": "Weekly Contest #42",
  "description": "Practice contest for algorithms",
  "startTime": "2026-01-25T14:00:00Z",
  "endTime": "2026-01-25T17:00:00Z",
  "duration": 10800,
  "status": "SCHEDULED",
  "penalty": 20,
  "freezeMinutes": 60,
  "enablePostContest": true,
  "locked": false,
  "participantCount": 25,
  "isRegistered": true,
  "group": {
    "id": "group-456",
    "name": "Competitive Programming Club"
  },
  "owner": {
    "id": "user-789",
    "nickname": "coach_john"
  },
  "problems": [
    {
      "position": 1,
      "slug": "sum-two-numbers",
      "title": "Sum of Two Numbers",
      "timeLimit": 2000,
      "memoryLimit": 256
    },
    {
      "position": 2,
      "slug": "binary-search",
      "title": "Binary Search Implementation",
      "timeLimit": 1000,
      "memoryLimit": 128
    }
  ],
  "createdAt": "2026-01-20T10:00:00Z",
  "updatedAt": "2026-01-22T15:30:00Z"
}
```

**Response Fields**:

| Field | Type | Description | Visibility |
|-------|------|-------------|------------|
| id | UUID | Contest identifier | All |
| name | string | Contest name | All |
| description | string | Contest description | All |
| startTime | timestamp | When contest starts (UTC) | All |
| endTime | timestamp | When contest ends (UTC) | All |
| duration | integer | Duration in seconds (calculated) | All |
| status | enum | SCHEDULED, ACTIVE, or FINISHED | All |
| penalty | integer | Penalty minutes per wrong submission | All |
| freezeMinutes | integer | Minutes before end to freeze standings (null = no freeze) | All |
| enablePostContest | boolean | Whether postcompetition is enabled | All |
| locked | boolean | Whether contest is locked | Leads/Admin only |
| participantCount | integer | Number of registered participants | All |
| isRegistered | boolean | Whether current user is registered | All |
| group | object | Group info (id, name) | All |
| owner | object | Owner info (id, nickname) | All |
| problems | array | List of problems | See visibility rules |
| createdAt | timestamp | Creation timestamp | All |
| updatedAt | timestamp | Last update timestamp | All |

**Problem Visibility Rules**:

| Contest Status | Registered | Member (not registered) | Non-member (VISIBLE group) |
|----------------|------------|-------------------------|----------------------------|
| SCHEDULED | ❌ Empty | ❌ Empty | ❌ Empty |
| ACTIVE | ✅ Full list | ✅ Full list | ❌ Empty |
| FINISHED | ✅ Full list | ✅ Full list | ❌ Empty |

**Error Responses**:

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 404 Not Found

```json
{
  "error": "CONTEST_NOT_FOUND",
  "message": "Contest not found",
  "contestId": "nonexistent-id"
}
```

> **Note**: Returns 404 for contests in NOT_VISIBLE groups when user is not a member (does not reveal existence).

---

### GET /groups/{groupId}/contests

List contests in a group with optional filters and pagination.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group ID |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| status | enum | No | (all) | Filter by status: SCHEDULED, ACTIVE, FINISHED |
| page | integer | No | 1 | Page number (1-indexed) |
| limit | integer | No | 20 | Items per page (max 100) |
| sortBy | string | No | startTime | Sort field: startTime, createdAt, name |
| sortOrder | string | No | desc | Sort order: asc, desc |

**Success Response (200 OK)**:

```json
{
  "data": [
    {
      "id": "contest-123",
      "name": "Weekly Contest #42",
      "description": "Practice contest for algorithms",
      "startTime": "2026-01-25T14:00:00Z",
      "endTime": "2026-01-25T17:00:00Z",
      "duration": 10800,
      "status": "SCHEDULED",
      "penalty": 20,
      "enablePostContest": true,
      "participantCount": 25,
      "isRegistered": false,
      "problemCount": 5
    },
    {
      "id": "contest-456",
      "name": "Algorithm Challenge",
      "description": "Test your skills",
      "startTime": "2026-01-20T10:00:00Z",
      "endTime": "2026-01-20T13:00:00Z",
      "duration": 10800,
      "status": "FINISHED",
      "penalty": 20,
      "enablePostContest": false,
      "participantCount": 42,
      "isRegistered": true,
      "problemCount": 4
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 42,
    "totalPages": 3,
    "hasNextPage": true,
    "hasPrevPage": false
  }
}
```

**List Item Fields** (summary, not full detail):

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Contest identifier |
| name | string | Contest name |
| description | string | Contest description (truncated to 200 chars) |
| startTime | timestamp | When contest starts |
| endTime | timestamp | When contest ends |
| duration | integer | Duration in seconds |
| status | enum | Computed status |
| penalty | integer | Penalty minutes |
| freezeMinutes | integer | Minutes before end to freeze standings |
| enablePostContest | boolean | Postcompetition enabled |
| participantCount | integer | Number of participants |
| isRegistered | boolean | Whether current user is registered |
| problemCount | integer | Number of problems in contest |

> **Note**: `locked` and `problems` array are NOT included in list view. Use GET /contests/{id} for full details.

**Error Responses**:

#### 400 Bad Request - Invalid Parameters

```json
{
  "error": "INVALID_PARAMETER",
  "message": "Invalid status filter",
  "validValues": ["SCHEDULED", "ACTIVE", "FINISHED"]
}
```

#### 400 Bad Request - Limit Too High

```json
{
  "error": "LIMIT_TOO_HIGH",
  "message": "Maximum limit is 100",
  "maxLimit": 100,
  "providedLimit": 500
}
```

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 404 Not Found - Group Not Found

```json
{
  "error": "GROUP_NOT_FOUND",
  "message": "Group not found",
  "groupId": "nonexistent-id"
}
```

> **Note**: Returns 404 for NOT_VISIBLE groups when user is not a member.

---

## Functional Requirements

### Authentication & Authorization

- **FR-001**: The system MUST require authentication for all endpoints.
- **FR-002**: Admin MUST have access to all contests regardless of group membership.
- **FR-003**: Group members MUST have access to all contests in their groups.
- **FR-004**: Non-members of VISIBLE groups MUST have read-only access to contest list and basic info.
- **FR-005**: Non-members of NOT_VISIBLE groups MUST receive 404 (contest not revealed).
- **FR-006**: The `locked` field MUST only be visible to Leads and Admin.

### Contest Status Computation

- **FR-007**: Status MUST be computed as SCHEDULED when currentTime < startTime.
- **FR-008**: Status MUST be computed as ACTIVE when startTime ≤ currentTime ≤ endTime.
- **FR-009**: Status MUST be computed as FINISHED when currentTime > endTime.
- **FR-010**: Duration MUST be calculated as (endTime - startTime) in seconds.

### Problem Visibility

- **FR-011**: Problems MUST be hidden (empty array) for SCHEDULED contests.
- **FR-012**: Problems MUST be visible to registered participants during ACTIVE contests.
- **FR-013**: Problems MUST be visible to group members during ACTIVE contests.
- **FR-014**: Problems MUST be hidden from non-members during ACTIVE contests.
- **FR-015**: Problems MUST be visible to registered participants after contest FINISHED.
- **FR-016**: Problems MUST be visible to group members after contest FINISHED.
- **FR-017**: Problems MUST be hidden from non-members after contest FINISHED.
- **FR-018**: Admin MUST see problems regardless of contest status.

### Registration Status

- **FR-019**: The system MUST indicate whether the current user is registered (`isRegistered`).
- **FR-020**: The system MUST show the total participant count (`participantCount`).

### Listing & Pagination

- **FR-021**: Default page size MUST be 20 items.
- **FR-022**: Maximum page size MUST be 100 items.
- **FR-023**: Default sort MUST be by startTime descending.
- **FR-024**: The system MUST support filtering by status.
- **FR-025**: The system MUST return accurate pagination metadata.
- **FR-026**: Page numbers MUST be 1-indexed.

### Response Data

- **FR-027**: Contest detail MUST include group info (id, name).
- **FR-028**: Contest detail MUST include owner info (id, nickname).
- **FR-029**: Contest list MUST include problemCount (not full problems array).
- **FR-030**: Description in list view MUST be truncated to 200 characters.

---

## Non-Functional Requirements

- **NFR-001**: Contest detail retrieval MUST complete within 200ms.
- **NFR-002**: Contest list retrieval MUST complete within 500ms for up to 100 items.
- **NFR-003**: The system MUST handle concurrent status transitions gracefully.
- **NFR-004**: The system MUST cache participant counts for performance.

---

## Data Model

### Key Entities

- **Contest**: The competition being viewed.
  Key attributes:
  - `id`, `name`, `description`
  - `startTime`, `endTime`
  - `penalty`, `enablePostContest`, `locked`
  - `group_id`, `ownerId`
  - `createdAt`, `updatedAt`

- **Contest_Problem**: Problems in the contest.
  Key attributes:
  - `contest_id`, `problem_id`, `position`

- **Problem**: Problem details.
  Key attributes:
  - `slug`, `title`, `timeLimit`, `memoryLimit`

- **ContestParticipant** (NoSQL): Registration data.
  Key attributes:
  - `contestantId`, `registeredAt`

- **Group**: Group info.
  Key attributes:
  - `id`, `name`, `visibility`

- **GroupMember**: Membership check.
  Key attributes:
  - `user_id`, `group_id`, `memberRole`

---

## Security Considerations

- **SEC-001**: NOT_VISIBLE group contests MUST NOT be discoverable by non-members.
- **SEC-002**: Problem details MUST NOT leak before contest starts.
- **SEC-003**: The `locked` field MUST NOT be exposed to non-privileged users.
- **SEC-004**: User's registration status MUST only reflect their own status.

---

## Optional Notes

- **Caching**: Consider caching participant counts and computed status.
- **Real-time Updates**: Consider WebSocket for live status updates during contests.
- **Timezone**: All times are UTC; client is responsible for conversion.
- **Standing Link**: Response could include link to standings endpoint (future).

