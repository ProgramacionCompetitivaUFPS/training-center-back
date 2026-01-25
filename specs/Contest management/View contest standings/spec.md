# Feature Specification: View Contest Standings

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Current Standings (Priority: P1)

As a contest participant, I want to view the current standings during a contest so that I can see my rank and compare my progress with other participants.

**Why this priority**: Real-time standings are essential for competitive programming contests, providing motivation and strategy insights.

**Independent Test**: Authenticated request to the standings endpoint during an ACTIVE contest. Verify correct ranking calculation and ICPC-style display.

**Acceptance Scenarios**:

1. **Scenario**: Participant views standings during ACTIVE contest (no freeze)

* **Given** a contest is ACTIVE with `freezeMinutes = null`
* **And** the authenticated user is registered to the contest
* **When** they request the standings
* **Then** the system returns the full standings with real-time data
* **And** includes ICPC-style problem matrix for each participant

2. **Scenario**: Participant views standings during freeze period

* **Given** a contest is ACTIVE with `freezeMinutes = 60`
* **And** current time is within the last 60 minutes before `endTime`
* **And** the authenticated user is a participant (not Lead/Admin)
* **When** they request the standings
* **Then** the system returns standings frozen at `endTime - freezeMinutes`
* **And** submissions after freeze time show as "pending" (?)

3. **Scenario**: Lead views real standings during freeze period

* **Given** a contest is ACTIVE with freeze enabled
* **And** the authenticated user is a Lead of the group
* **When** they request the standings with `?realtime=true`
* **Then** the system returns the actual real-time standings (ignoring freeze)

4. **Scenario**: Admin views real standings during freeze period

* **Given** a contest is ACTIVE with freeze enabled
* **And** the authenticated user is an Admin
* **When** they request the standings with `?realtime=true`
* **Then** the system returns the actual real-time standings (ignoring freeze)

5. **Scenario**: Group member (not registered) views standings

* **Given** a contest is ACTIVE
* **And** the authenticated user is a member of the group but NOT registered
* **When** they request the standings
* **Then** the system returns the standings (can view but not participate)

6. **Scenario**: Non-member views standings in VISIBLE group

* **Given** a contest is ACTIVE in a VISIBLE group
* **And** the authenticated user is NOT a member of the group
* **When** they request the standings
* **Then** the system returns the standings (read-only access)

7. **Scenario**: Non-member attempts to view standings in NOT_VISIBLE group

* **Given** a contest exists in a NOT_VISIBLE group
* **And** the authenticated user is NOT a member
* **When** they request the standings
* **Then** the system rejects with 404 Not Found

---

### User Story 2 - View Final Standings (Priority: P1)

As a user, I want to view the final standings of a finished contest so that I can see the official results.

**Why this priority**: Final standings must be preserved and accessible for historical records.

**Independent Test**: Request standings for a FINISHED contest. Verify frozen data is returned from snapshot collection.

**Acceptance Scenarios**:

1. **Scenario**: View final standings of FINISHED contest

* **Given** a contest is FINISHED
* **And** the authenticated user has view access (member, participant, or non-member of VISIBLE group)
* **When** they request the standings
* **Then** the system returns the frozen final standings from `contest_{contestId}_standings_final`
* **And** the standings are immutable (same on every request)

2. **Scenario**: View final standings during postcompetition

* **Given** a contest is FINISHED with `enablePostContest = true`
* **And** users are submitting in postcompetition
* **When** they request the standings
* **Then** the system returns the frozen final standings (postcompetition does NOT affect)

---

### User Story 3 - Filter Standings (Priority: P2)

As a user, I want to filter standings by country, city, or institution so that I can compare with specific groups.

**Why this priority**: Filtering enables regional comparisons and institutional rankings.

**Acceptance Scenarios**:

1. **Scenario**: Filter by country

* **Given** a contest has participants from multiple countries
* **When** the user requests standings with `?country=Colombia`
* **Then** only participants from Colombia are returned
* **And** rankings are recalculated within the filtered set

2. **Scenario**: Filter by institution

* **Given** a contest has participants from multiple institutions
* **When** the user requests standings with `?institution=MIT`
* **Then** only participants from MIT are returned
* **And** rankings show their position within the filtered group

3. **Scenario**: Filter by city

* **Given** a contest has participants from multiple cities
* **When** the user requests standings with `?city=Bogotá`
* **Then** only participants from Bogotá are returned

4. **Scenario**: Combine filters

* **Given** a contest has participants with various attributes
* **When** the user filters by `?country=Colombia&city=Medellín`
* **Then** only participants matching ALL filters are returned

---

### User Story 4 - Real-Time Standings via SSE (Priority: P1)

As a contest participant, I want to receive real-time standing updates so that I can see ranking changes immediately without refreshing.

**Why this priority**: Real-time updates enhance the competitive experience significantly.

**Acceptance Scenarios**:

1. **Scenario**: Subscribe to standing updates during ACTIVE contest

* **Given** a contest is ACTIVE
* **And** the user has view access to the standings
* **When** they connect to the SSE endpoint
* **Then** they receive the initial standings snapshot
* **And** receive incremental updates when any participant's standing changes

2. **Scenario**: Receive update when submission is judged

* **Given** a user is connected to SSE
* **And** another participant submits and gets ACCEPTED
* **When** the standing is updated
* **Then** all connected clients receive the update within 2 seconds

3. **Scenario**: SSE respects freeze rules

* **Given** freeze is active and user is not Lead/Admin
* **When** a standing update occurs after freeze time
* **Then** the update is NOT sent (or sent as "pending")

4. **Scenario**: SSE disconnection and reconnection

* **Given** a user's SSE connection is lost
* **When** they reconnect
* **Then** they receive the full current standings snapshot
* **And** resume receiving incremental updates

5. **Scenario**: Contest ends while connected

* **Given** a user is connected to SSE during ACTIVE contest
* **When** the contest ends (currentTime > endTime)
* **Then** they receive a final "contest_ended" event
* **And** the connection remains open for final standings

---

### Edge Cases

- Contest with no participants (empty standings).
- Contest with 1000+ participants (pagination stress test).
- Participant with same score and penalty (tie-breaking by earliest accepted time).
- Filter that returns no results.
- SSE connection during contest status transition.
- Freeze time equals contest duration (entire contest frozen - edge case).
- Multiple submissions judged simultaneously (atomic standing updates).

---

## API Contract

### GET /contests/{contestId}/standings

Retrieve standings for a contest.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| contestId | UUID | Yes | The contest ID |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| page | integer | No | 1 | Page number (1-indexed) |
| limit | integer | No | 50 | Items per page (max 100) |
| country | string | No | - | Filter by country |
| city | string | No | - | Filter by city |
| institution | string | No | - | Filter by institution |
| realtime | boolean | No | false | Bypass freeze (Leads/Admin only) |

**Success Response (200 OK)**:

```json
{
  "contest": {
    "id": "contest-123",
    "name": "Weekly Contest #42",
    "status": "ACTIVE",
    "startTime": "2026-01-25T14:00:00Z",
    "endTime": "2026-01-25T17:00:00Z",
    "penalty": 20,
    "freezeMinutes": 60,
    "isFrozen": true,
    "frozenAt": "2026-01-25T16:00:00Z"
  },
  "problems": [
    { "position": 1, "slug": "sum-two-numbers", "title": "A" },
    { "position": 2, "slug": "binary-search", "title": "B" },
    { "position": 3, "slug": "dijkstra", "title": "C" }
  ],
  "standings": [
    {
      "rank": 1,
      "participant": {
        "id": "user-456",
        "nickname": "speed_coder",
        "country": "Colombia",
        "city": "Medellín",
        "institution": "Universidad EAFIT"
      },
      "problemsSolved": 3,
      "totalPenalty": 125,
      "problems": [
        {
          "position": 1,
          "status": "ACCEPTED",
          "attempts": 2,
          "time": 45,
          "penalty": 20
        },
        {
          "position": 2,
          "status": "ACCEPTED",
          "attempts": 1,
          "time": 60,
          "penalty": 0
        },
        {
          "position": 3,
          "status": "ACCEPTED",
          "attempts": 1,
          "time": 80,
          "penalty": 0
        }
      ]
    },
    {
      "rank": 2,
      "participant": {
        "id": "user-789",
        "nickname": "algo_master",
        "country": "México",
        "city": "Ciudad de México",
        "institution": "UNAM"
      },
      "problemsSolved": 2,
      "totalPenalty": 90,
      "problems": [
        {
          "position": 1,
          "status": "ACCEPTED",
          "attempts": 1,
          "time": 30,
          "penalty": 0
        },
        {
          "position": 2,
          "status": "PENDING",
          "attempts": 3,
          "time": null,
          "penalty": 0
        },
        {
          "position": 3,
          "status": "ACCEPTED",
          "attempts": 1,
          "time": 60,
          "penalty": 0
        }
      ]
    },
    {
      "rank": 3,
      "participant": {
        "id": "user-101",
        "nickname": "newbie_123",
        "country": "Colombia",
        "city": "Bogotá",
        "institution": "Universidad Nacional"
      },
      "problemsSolved": 2,
      "totalPenalty": 150,
      "problems": [
        {
          "position": 1,
          "status": "WRONG_ANSWER",
          "attempts": 5,
          "time": null,
          "penalty": 0
        },
        {
          "position": 2,
          "status": "ACCEPTED",
          "attempts": 2,
          "time": 90,
          "penalty": 20
        },
        {
          "position": 3,
          "status": "ACCEPTED",
          "attempts": 1,
          "time": 40,
          "penalty": 0
        }
      ]
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 125,
    "totalPages": 3,
    "hasNextPage": true,
    "hasPrevPage": false
  },
  "filters": {
    "country": null,
    "city": null,
    "institution": null,
    "filteredTotal": 125
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| contest | object | Contest metadata with freeze info |
| contest.isFrozen | boolean | Whether standings are currently frozen |
| contest.frozenAt | timestamp | When freeze started (null if not frozen) |
| problems | array | Problem positions and display titles (A, B, C...) |
| standings | array | Ranked list of participants |
| standings[].rank | integer | Current rank (1-indexed) |
| standings[].participant | object | User info (id, nickname, country, city, institution) |
| standings[].problemsSolved | integer | Number of problems solved |
| standings[].totalPenalty | integer | Total time + penalties in minutes |
| standings[].problems | array | Per-problem results |
| standings[].problems[].status | enum | ACCEPTED, WRONG_ANSWER, PENDING, NOT_ATTEMPTED |
| standings[].problems[].attempts | integer | Number of submissions |
| standings[].problems[].time | integer | Minutes from start to AC (null if not solved) |
| standings[].problems[].penalty | integer | Penalty for this problem |
| pagination | object | Pagination metadata |
| filters | object | Applied filters and filtered count |

**Problem Status Values**:

| Status | Description | Display |
|--------|-------------|---------|
| ACCEPTED | Problem solved | +{attempts} / {time} |
| WRONG_ANSWER | Attempted but not solved | -{attempts} |
| PENDING | Submission(s) during freeze, unknown result | ? or ?{attempts} |
| NOT_ATTEMPTED | No submissions | - |

**Error Responses**:

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 403 Forbidden - Realtime Not Allowed

```json
{
  "error": "REALTIME_NOT_ALLOWED",
  "message": "Only Leads and Admin can view real-time standings during freeze"
}
```

#### 404 Not Found

```json
{
  "error": "CONTEST_NOT_FOUND",
  "message": "Contest not found"
}
```

---

### GET /contests/{contestId}/standings/stream

Subscribe to real-time standing updates via Server-Sent Events (SSE).

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Accept | string | Yes | text/event-stream |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| contestId | UUID | Yes | The contest ID |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| realtime | boolean | No | false | Bypass freeze (Leads/Admin only) |

**SSE Event Types**:

#### Event: `snapshot`

Sent immediately upon connection with full current standings.

```
event: snapshot
data: {"standings": [...], "contest": {...}, "problems": [...]}
```

#### Event: `update`

Sent when a participant's standing changes.

```
event: update
data: {
  "participantId": "user-456",
  "rank": 2,
  "previousRank": 3,
  "problemsSolved": 3,
  "totalPenalty": 125,
  "changedProblem": {
    "position": 2,
    "status": "ACCEPTED",
    "attempts": 2,
    "time": 75,
    "penalty": 20
  }
}
```

#### Event: `freeze`

Sent when freeze period begins.

```
event: freeze
data: {"frozenAt": "2026-01-25T16:00:00Z", "message": "Standings are now frozen"}
```

#### Event: `contest_ended`

Sent when contest ends.

```
event: contest_ended
data: {"endedAt": "2026-01-25T17:00:00Z", "finalStandingsAvailable": true}
```

#### Event: `ping`

Heartbeat every 30 seconds to keep connection alive.

```
event: ping
data: {"timestamp": "2026-01-25T15:30:00Z"}
```

**Connection Behavior**:

- Connection remains open for the duration of the contest
- Server sends `ping` every 30 seconds
- Client should handle reconnection automatically (SSE built-in)
- On reconnect, client receives fresh `snapshot`

**Error Responses**:

#### 401 Unauthorized (before connection)

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
  "message": "Contest not found"
}
```

#### 400 Bad Request - Contest Not Active

```json
{
  "error": "CONTEST_NOT_ACTIVE",
  "message": "SSE streaming is only available for ACTIVE contests"
}
```

---

## Functional Requirements

### Ranking Calculation (ICPC-style)

- **FR-001**: Rankings MUST be sorted by:
  1. Primary: Number of problems solved (descending)
  2. Secondary: Total penalty time (ascending)
  3. Tertiary: Earliest last accepted submission time (ascending) for tie-breaking

- **FR-002**: Penalty MUST be calculated as:
  - Time from contest start to accepted submission (in minutes)
  - Plus: `(wrongAttempts) * contestPenalty` for each accepted problem

- **FR-003**: Only submissions with `status = ACCEPTED` before `endTime` MUST count toward standings.

- **FR-004**: Postcompetition submissions (`submittedAt > endTime`) MUST NOT affect standings.

### Freeze Behavior

- **FR-005**: If `freezeMinutes` is set, standings MUST freeze at `endTime - freezeMinutes`.

- **FR-006**: During freeze, participant updates MUST show as `PENDING` status for non-privileged users.

- **FR-007**: Leads and Admin MUST be able to view real-time standings during freeze with `?realtime=true`.

- **FR-008**: If `freezeMinutes = null`, no freeze is applied.

- **FR-009**: Default `freezeMinutes` for new contests MUST be 60.

### Data Sources

- **FR-010**: During ACTIVE contests, standings MUST be retrieved from `contest_{contestId}_standings`.

- **FR-011**: For FINISHED contests, standings MUST be retrieved from `contest_{contestId}_standings_final`.

- **FR-012**: Final snapshot MUST be created when contest status changes to FINISHED.

### Filtering

- **FR-013**: Filters MUST support `country`, `city`, and `institution`.

- **FR-014**: Multiple filters MUST be combined with AND logic.

- **FR-015**: When filtered, rankings MUST be recalculated within the filtered set.

- **FR-016**: `filteredTotal` MUST reflect the count of participants matching filters.

### Pagination

- **FR-017**: Default page size MUST be 50 items.

- **FR-018**: Maximum page size MUST be 100 items.

- **FR-019**: Page numbers MUST be 1-indexed.

### SSE Real-Time Updates

- **FR-020**: SSE endpoint MUST send `snapshot` event immediately upon connection.

- **FR-021**: SSE endpoint MUST send `update` events within 2 seconds of standing changes.

- **FR-022**: SSE endpoint MUST send `ping` events every 30 seconds.

- **FR-023**: SSE endpoint MUST send `freeze` event when freeze period begins.

- **FR-024**: SSE endpoint MUST send `contest_ended` event when contest finishes.

- **FR-025**: SSE MUST only be available for ACTIVE contests.

- **FR-026**: SSE updates MUST respect freeze rules (unless `?realtime=true` for Leads/Admin).

### Visibility

- **FR-027**: Group members MUST have access to standings.

- **FR-028**: Registered participants MUST have access to standings.

- **FR-029**: Non-members of VISIBLE groups MUST have read-only access.

- **FR-030**: Non-members of NOT_VISIBLE groups MUST receive 404.

- **FR-031**: Admin MUST have access to all standings with `realtime` capability.

---

## Non-Functional Requirements

- **NFR-001**: Standings retrieval MUST complete within 500ms for up to 100 participants.
- **NFR-002**: SSE updates MUST be delivered within 2 seconds of the triggering event.
- **NFR-003**: SSE connections MUST support at least 1000 concurrent clients per contest.
- **NFR-004**: Standing calculations MUST be accurate to ICPC rules.
- **NFR-005**: Final standings snapshot MUST be immutable once created.

---

## Data Model

### Key Entities

- **Contest**: Competition with freeze configuration.
  Updated attributes:
  - `freezeMinutes` (integer, nullable, default: 60) - Minutes before end to freeze standings

- **ContestParticipant** (NoSQL): Participant standing data.
  Key attributes:
  - `contestantId` (string, PK)
  - `registeredAt` (timestamp)
  - `problemsSolved` (integer)
  - `penalty` (integer, total minutes)
  - `problems` (json map: slug → { attempts, acceptedAt, penalty })
  - `lastUpdated` (timestamp)

- **User**: Participant profile.
  Required attributes for standings:
  - `nickname` (string)
  - `country` (string) ← NEW
  - `city` (string) ← NEW
  - `institution` (string)

### NoSQL Collections

- `contest_{contestId}_standings`: Active standings (updated in real-time)
- `contest_{contestId}_standings_final`: Frozen snapshot (created when contest ends)

### Example Document Structure

```json
{
  "contestantId": "user-456",
  "registeredAt": "2026-01-20T10:00:00Z",
  "problemsSolved": 3,
  "penalty": 125,
  "problems": {
    "sum-two-numbers": {
      "attempts": 2,
      "acceptedAt": "2026-01-25T14:45:00Z",
      "penalty": 20
    },
    "binary-search": {
      "attempts": 1,
      "acceptedAt": "2026-01-25T15:00:00Z",
      "penalty": 0
    },
    "dijkstra": {
      "attempts": 1,
      "acceptedAt": "2026-01-25T15:20:00Z",
      "penalty": 0
    }
  },
  "lastUpdated": "2026-01-25T15:20:00Z"
}
```

---

## Security Considerations

- **SEC-001**: NOT_VISIBLE group standings MUST NOT be discoverable by non-members.
- **SEC-002**: `realtime` parameter MUST be restricted to Leads and Admin.
- **SEC-003**: SSE connections MUST be authenticated.
- **SEC-004**: SSE MUST NOT leak standings data to unauthorized users.

---

## Optional Notes

- **Caching**: Consider caching filtered standings for common filter combinations.
- **Problem Display**: Problems are labeled A, B, C... based on position (1, 2, 3...).
- **Tie-breaking**: When problemsSolved and penalty are equal, earliest lastAcceptedTime wins.
- **Deactivated Users**: Show anonymized nickname (e.g., "deleted_user_abc123") in standings.
- **Empty Problems**: If a participant hasn't attempted a problem, show `NOT_ATTEMPTED`.

