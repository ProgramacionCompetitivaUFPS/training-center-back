# Feature Specification: View Contest Standings

**Created**: 2026-01-24
**Synced with the implementation**: 2026-11-18

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Current Standings (Priority: P1)

As a contest participant, I want to view the current standings during a contest so that I can see my rank and compare my progress with other participants.

**Why this priority**: Live standings are essential for competitive programming contests, providing motivation and strategy insights.

**Independent Test**: Authenticated request to the standings endpoint during an ACTIVE contest. Verify correct ranking calculation and ICPC-style display.

**Acceptance Scenarios**:

1. **Scenario**: Participant views standings during ACTIVE contest (no freeze)

* **Given** a contest is ACTIVE with `freezeMinutes = 0` (no freeze)
* **And** the authenticated user is registered to the contest
* **When** they request the standings
* **Then** the system returns the full standings computed from every judged submission up to `endTime`
* **And** includes an ICPC-style problem matrix for each participant

2. **Scenario**: Participant views standings during freeze period

* **Given** a contest is ACTIVE with `freezeMinutes = 60`
* **And** current time is within the last 60 minutes before `endTime`
* **And** the authenticated user is a participant (not Lead/Admin)
* **When** they request the standings
* **Then** the system returns standings as they were at `endTime - freezeMinutes`
* **And** submissions made after the freeze time are hidden: they do not count as attempts, accepted submissions do not count as solved, and ranking and penalty ignore them

3. **Scenario**: Lead views real standings during freeze period

* **Given** a contest is ACTIVE with freeze enabled and the freeze period has started
* **And** the authenticated user is a Lead of the group
* **When** they request the standings with `?realtime=true`
* **Then** the system returns the actual standings (ignoring freeze)

4. **Scenario**: Admin views real standings during freeze period

* **Given** a contest is ACTIVE with freeze enabled and the freeze period has started
* **And** the authenticated user is an Admin
* **When** they request the standings with `?realtime=true`
* **Then** the system returns the actual standings (ignoring freeze)

5. **Scenario**: Participant requests real-time standings during freeze

* **Given** a contest is in its freeze period
* **And** the authenticated user is neither Lead nor Admin
* **When** they request the standings with `?realtime=true`
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

6. **Scenario**: `realtime` outside a freeze period

* **Given** the contest has no freeze or the freeze has not started
* **When** any authorized user requests the standings with `?realtime=true`
* **Then** the parameter has no effect and the request succeeds

7. **Scenario**: Group member (not registered) views standings

* **Given** a contest is ACTIVE
* **And** the authenticated user is a member of the group but NOT registered
* **When** they request the standings
* **Then** the system returns the standings (can view but not participate)

8. **Scenario**: Non-member views standings in VISIBLE group

* **Given** a contest is ACTIVE in a VISIBLE group
* **And** the authenticated user is NOT a member of the group
* **When** they request the standings
* **Then** the system returns the standings (read-only access)

9. **Scenario**: Non-member attempts to view standings in NOT_VISIBLE group

* **Given** a contest exists in a NOT_VISIBLE group
* **And** the authenticated user is NOT a member (and not Admin)
* **When** they request the standings
* **Then** the system rejects with 404 Not Found

---

### User Story 2 - View Final Standings (Priority: P1)

As a user, I want to view the final standings of a finished contest so that I can see the official results.

**Why this priority**: Final standings must remain accessible for historical records.

**Independent Test**: Request standings for a FINISHED contest. Verify the freeze is lifted and later submissions are ignored.

**Acceptance Scenarios**:

1. **Scenario**: View final standings of FINISHED contest

* **Given** a contest is FINISHED
* **And** the authenticated user has view access (member, participant, or non-member of VISIBLE group)
* **When** they request the standings
* **Then** the system returns the full standings (the freeze no longer applies)
* **And** the result is stable across requests unless a submission with `submittedAt <= endTime` is judged after `endTime`; its verdict then changes the standings the next time the projection is rebuilt. The problem list of a finished contest cannot be changed

2. **Scenario**: View standings during postcompetition

* **Given** a contest is FINISHED with `enablePostContest = true`
* **And** users are submitting in postcompetition
* **When** they request the standings
* **Then** the system returns the same standings as before (postcompetition submissions do NOT affect them)

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

5. **Scenario**: Filter with team participants

* **Given** a contest has team participants
* **When** the user filters by `?institution=MIT`
* **Then** a team is returned if at least one of its members matches the combined filter

---

### User Story 4 - List Filter Options (Priority: P3)

As a user, I want to know which country, city and institution values exist among a contest's participants so that I can pick a filter value instead of typing it.

**Acceptance Scenarios**:

1. **Scenario**: Request the available filter values

* **Given** the user has view access to the contest standings
* **When** they request `GET /groups/{groupId}/contests/{contestId}/standings/filters`
* **Then** the system returns the distinct, non-empty, alphabetically sorted countries, cities and institutions of the contest's individual participants and team members

---

### Edge Cases

- Contest with no participants (empty standings).
- Contest with 1000+ participants (pagination).
- Participants with the same score, penalty and last accepted time share the same rank.
- Filter that returns no results.
- A problem removed from the contest stops counting; it counts again if put back.
- A user registered individually who is also a selected team member (MIXED contests): their submissions count for the team only.
- A participant whose profile cannot be resolved is returned with an empty `displayName`.
- Submissions still `PENDING` or `RUNNING` in the judge are not part of the standings until judged.
- `page` or `limit` that are not positive integers fall back to their defaults.

---

## API Contract

### GET /groups/{groupId}/contests/{contestId}/standings

Retrieve the standings of a contest.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group that owns the contest |
| contestId | UUID | Yes | The contest ID |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| page | integer | No | 1 | Page number (1-indexed) |
| limit | integer | No | 50 | Items per page (no upper bound is enforced) |
| country | string | No | - | Filter by country (case-insensitive exact match) |
| city | string | No | - | Filter by city (case-insensitive exact match) |
| institution | string | No | - | Filter by institution (case-insensitive exact match) |
| realtime | boolean | No | false | Bypass freeze (Leads/Admin only; only the value `true` enables it) |

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
    { "position": 1, "slug": "sum-two-numbers", "title": "Sum Two Numbers", "color": "#e53935", "accepted": 2, "tried": 2 },
    { "position": 2, "slug": "binary-search", "title": "Binary Search", "color": null, "accepted": 1, "tried": 2 },
    { "position": 3, "slug": "dijkstra", "title": "Dijkstra", "color": "#1e88e5", "accepted": 2, "tried": 2 }
  ],
  "standings": [
    {
      "rank": 1,
      "participant": {
        "id": "team-123",
        "type": "TEAM",
        "displayName": "Team Alpha",
        "members": ["alice", "bob", "carol"]
      },
      "problemsSolved": 3,
      "totalPenalty": 205,
      "problems": [
        { "position": 1, "status": "ACCEPTED", "attempts": 1, "time": 45, "penalty": 65, "isFirstSolve": false },
        { "position": 2, "status": "ACCEPTED", "attempts": 0, "time": 60, "penalty": 60, "isFirstSolve": true },
        { "position": 3, "status": "ACCEPTED", "attempts": 0, "time": 80, "penalty": 80, "isFirstSolve": false }
      ]
    },
    {
      "rank": 2,
      "participant": {
        "id": "user-789",
        "type": "INDIVIDUAL",
        "displayName": "algo_master",
        "nickname": "algo_master",
        "name": "Ana Ruiz",
        "country": "México",
        "city": "Ciudad de México",
        "institution": "UNAM"
      },
      "problemsSolved": 2,
      "totalPenalty": 90,
      "problems": [
        { "position": 1, "status": "ACCEPTED", "attempts": 0, "time": 30, "penalty": 30, "isFirstSolve": true },
        { "position": 2, "status": "WRONG_ANSWER", "attempts": 3, "time": null, "penalty": 0, "isFirstSolve": false },
        { "position": 3, "status": "ACCEPTED", "attempts": 0, "time": 60, "penalty": 60, "isFirstSolve": true }
      ]
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 2,
    "totalPages": 1,
    "hasNextPage": false,
    "hasPrevPage": false
  },
  "filters": {
    "country": null,
    "city": null,
    "institution": null,
    "filteredTotal": 2
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| contest | object | Contest metadata with freeze info |
| contest.freezeMinutes | integer | Freeze length in minutes. A contest without freeze stores `0`, and this response returns `null` in that case |
| contest.isFrozen | boolean | Whether standings are currently frozen for this response |
| contest.frozenAt | timestamp | When the freeze started (`null` if not frozen) |
| problems | array | The contest's current problems in position order |
| problems[].title | string | The problem's title |
| problems[].color | string | The balloon color of the problem in the contest as `#rrggbb`, or `null` when none was set |
| problems[].accepted | integer | Participants who solved the problem, over every participant (not only the filtered ones), as visible at the freeze |
| problems[].tried | integer | Participants with at least one visible counted attempt or solve on the problem, over every participant (solvers included) |
| standings | array | Ranked list of participants (current page) |
| standings[].rank | integer | Rank within the filtered set (1-indexed; ties share a rank) |
| standings[].participant | object | Participant info (team or individual) |
| standings[].participant.id | string | User ID or Team ID |
| standings[].participant.type | enum | `TEAM` or `INDIVIDUAL` |
| standings[].participant.displayName | string | Nickname for individuals, team name for teams |
| standings[].participant.nickname | string | Individuals only (omitted for teams) |
| standings[].participant.name | string | Individuals only (omitted for teams) |
| standings[].participant.members | array | Teams only, and only when the contest has `showTeamMembers = true`: member nicknames, sorted |
| standings[].participant.country / city / institution | string | Individuals only (omitted when empty or for teams) |
| standings[].problemsSolved | integer | Number of problems solved |
| standings[].totalPenalty | integer | Sum of the penalty of each solved problem, in minutes |
| standings[].problems | array | One element per contest problem, in position order |
| standings[].problems[].status | enum | `ACCEPTED`, `WRONG_ANSWER`, `NOT_ATTEMPTED` |
| standings[].problems[].attempts | integer | Wrong attempts before the first accepted submission (the accepted submission itself is not counted); all wrong attempts if unsolved |
| standings[].problems[].time | integer | Minutes from contest start to the accepted submission (`null` if not solved) |
| standings[].problems[].penalty | integer | `time + attempts × contest penalty` for solved problems, `0` otherwise |
| standings[].problems[].isFirstSolve | boolean | `true` on the cell of the participant who solved the problem first among every participant (not only the filtered ones); `false` on every other cell. Always present |
| pagination | object | Pagination metadata |
| filters | object | Applied filters; `filteredTotal` is the number of participants after filtering |

**Problem Status Values**:

| Status | Description | Display |
|--------|-------------|---------|
| ACCEPTED | Problem solved | +{attempts} / {time} |
| WRONG_ANSWER | Attempted (with at least one counted wrong attempt) but not solved | -{attempts} |
| NOT_ATTEMPTED | No visible attempts | - |

**Display Name**:

| Type | Format | Example |
|------|--------|---------|
| INDIVIDUAL | `{nickname}` | `algo_master` |
| TEAM | `{teamName}`; member nicknames are in `members` when `showTeamMembers` is true | `Team Alpha` |

> **Note**: Teams have no country, city or institution of their own. When filtering, a team is kept if at least one of its members matches.

**Error Responses**:

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden - Real-time Not Allowed

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "only leads and admins can view real-time standings during freeze"
}
```

#### 404 Not Found

Returned when the contest does not exist, does not belong to the group, or the group is NOT_VISIBLE and the user is not a member or Admin.

```json
{
  "error": "CONTEST_NOT_FOUND",
  "message": "contest not found"
}
```

---

### GET /groups/{groupId}/contests/{contestId}/standings/filters

List the distinct values available for the standings filters. Same authentication and visibility rules as the standings endpoint.

**Success Response (200 OK)**:

```json
{
  "countries": ["Colombia", "México"],
  "cities": ["Bogotá", "Ciudad de México"],
  "institutions": ["UNAM", "Universidad Nacional"]
}
```

**Error Responses**: `401 UNAUTHORIZED`, `404 CONTEST_NOT_FOUND`.

---

## Functional Requirements

### Ranking Calculation (ICPC-style)

- **FR-001**: Rankings MUST be sorted by:
  1. Primary: Number of problems solved (descending)
  2. Secondary: Total penalty time (ascending)
  3. Tertiary: Earliest last accepted submission time (ascending) for tie-breaking

  Participants equal in all three share the same rank.

- **FR-002**: The penalty of a solved problem MUST be calculated as:
  - Whole minutes from contest start to the accepted submission
  - Plus: `wrongAttempts × contestPenalty`, where `wrongAttempts` counts the wrong submissions made before the first accepted one

- **FR-003**: Only the first `ACCEPTED` submission of a problem, made at or before `endTime`, counts as a solve.

- **FR-004**: Postcompetition submissions (`submittedAt > endTime`) MUST NOT affect standings.

- **FR-004.1**: A submission counts as a wrong attempt only if its verdict is `WRONG_ANSWER`, `TIME_LIMIT_EXCEEDED`, `MEMORY_LIMIT_EXCEEDED`, `OUTPUT_LIMIT_EXCEEDED` or `RUNTIME_ERROR`. `COMPILATION_ERROR` and `SYSTEM_ERROR` never count. Submissions not yet judged (`PENDING`, `RUNNING`) are not considered.

- **FR-004.2**: Only the problems the contest has now count. A problem removed from the contest stops counting (its column disappears) and counts again if it is put back, because its submissions keep their `contest_id`.

- **FR-004.3**: Submissions are attributed to the participant's standing: an individual registrant to themself, a selected team member to the team. When a user is both registered individually and a selected team member, the team takes precedence.

### Freeze Behavior

- **FR-005**: If `freezeMinutes > 0`, standings MUST freeze at `endTime - freezeMinutes` while the contest is ACTIVE.

- **FR-006**: While frozen, submissions made after the freeze time MUST be hidden from non-privileged users: they count neither as attempts nor as solves, and do not influence rank or penalty.

- **FR-007**: Leads and Admin MUST be able to view real-time standings during freeze with `?realtime=true`; any other user requesting it during freeze MUST receive 403 `INSUFFICIENT_PERMISSIONS`.

- **FR-008**: If `freezeMinutes = 0`, no freeze is applied.

- **FR-009**: Default `freezeMinutes` for new contests MUST be 60.

- **FR-009.1**: Once the contest is no longer ACTIVE the freeze no longer applies.

### Data Sources and Caching

- **FR-010**: Standings are computed from a per-contest projection stored in Redis under the key `standings:{contestId}`. The projection holds, per participant, the raw timestamps of wrong attempts and of the first accepted submission per problem, plus the team members, team names and participant profiles. Ranking, freeze, penalty and filters are applied in memory on each request, so changing `penalty` or the freeze needs no cache invalidation.

- **FR-011**: When there is no cached projection, the system MUST rebuild it from the registrations, selected team members and judged submissions of the contest (at most 10,000 participants).

- **FR-012**: A projection older than 5 seconds is still served, and a background refresh is triggered (guarded by the lock `lock:standings:{contestId}`). The refresh is triggered by a request and the stale data is what that request receives, so a later request is needed to see the refreshed data; changes to the contest's problems are reflected only after such a refresh, and a failed refresh delays them further. A rebuild is limited to 30 seconds and the refresh lock lasts 45 seconds, so a second refresh cannot start while one is running. When there is no projection, concurrent requests for the same contest in one instance share a single rebuild and wait for it.

- **FR-013**: The projection is removed when the contest is deleted or when its group is deleted. There is no final snapshot of the standings.

### Board Fields

- **FR-028**: Each problem header MUST carry the problem's balloon color in the contest (`color`, `null` when none) and the totals `accepted` and `tried`.

- **FR-029**: `accepted` and `tried` MUST be computed over every participant of the contest, whatever the `country`, `city` and `institution` filters, and MUST count only what is visible: while the standings are frozen, solves and attempts after the freeze count in neither. A participant counts in `tried` if they have at least one counted wrong attempt (FR-004.1) or a solve on the problem. With `?realtime=true` during the freeze nothing is hidden.

- **FR-030**: `isFirstSolve` MUST be `true` only on the cell of the participant with the earliest visible `ACCEPTED` time on that problem, among every participant. A tie on that time goes to the smaller participant ID. A problem nobody has solved has no first solve, and a solve hidden by the freeze cannot be one.

- **FR-031**: When the filters leave out the first solver, no cell of the response carries `isFirstSolve = true` for that problem.

### Filtering

- **FR-014**: Filters MUST support `country`, `city`, and `institution`, as case-insensitive exact matches.

- **FR-015**: Multiple filters MUST be combined with AND logic.

- **FR-016**: When filtered, rankings MUST be recalculated within the filtered set.

- **FR-017**: `filteredTotal` MUST reflect the count of participants matching the filters.

- **FR-018**: A team MUST match a filter if at least one of its members matches it.

- **FR-019**: The filter options endpoint MUST return the distinct, non-empty, alphabetically sorted values of the contest's participants.

### Pagination

- **FR-020**: Default page size MUST be 50 items.

- **FR-021**: Page numbers MUST be 1-indexed. An invalid or non-positive `page` or `limit` falls back to its default.

- **FR-022**: The system does not enforce a maximum page size.

### Visibility

- **FR-023**: Group members MUST have access to standings.

- **FR-024**: Registered participants MUST have access to standings as long as they are members of the group. Access is decided by group membership (or Lead/Admin role), not by contest registration: a registered user who has left a NOT_VISIBLE group receives 404.

- **FR-025**: Non-members of VISIBLE groups MUST have read-only access.

- **FR-026**: Non-members of NOT_VISIBLE groups MUST receive 404, except Admins, who can access every group (FR-027).

- **FR-027**: Admin MUST have access to all standings with `realtime` capability.

---

## Non-Functional Requirements

- **NFR-001**: Standings retrieval SHOULD complete within 500ms for up to 100 participants.
- **NFR-002**: Standing calculations MUST be accurate to ICPC rules.
- **NFR-003**: Standings of a FINISHED contest are stable: postcompetition submissions are ignored and the contest's problem list cannot be changed once it has finished. The only way they change after `endTime` is a submission made before `endTime` that is judged afterwards (submissions still `PENDING` or `RUNNING` are not part of the projection until judged).

---

## Data Model

### Key Entities

- **Contest**: Competition with freeze configuration.
  - `freezeMinutes` (integer, default: 60, `0` = no freeze) - Minutes before end to freeze standings; the standings response returns `null` when it is `0`
  - `penalty` (integer) - Minutes added per wrong attempt on a solved problem
  - `showTeamMembers` (boolean) - Whether team member nicknames are returned

- **CachedStandings** (Redis, key `standings:{contestId}`): Per-contest projection.
  - `Participants` - one entry per standing (individual or team) with, per problem, the timestamps of the wrong attempts before the first accepted submission and the time of the first accepted submission
  - `TeamMembers` - team ID → member user IDs
  - `Profiles` - user ID → nickname, name, country, city, institution
  - `TeamNames` - team ID → team name
  - `LastUpdated` - when the projection was built

- **User**: Participant profile.
  Attributes used by standings: `nickname`, `name`, `country`, `city`, `institution`.

---

## Security Considerations

- **SEC-001**: NOT_VISIBLE group standings MUST NOT be discoverable by non-members.
- **SEC-002**: `realtime` parameter MUST be restricted to Leads and Admin.

---

## Not Implemented

These behaviours appeared in earlier drafts of this spec and are not part of the current system:

- Real-time updates over SSE (`standings/stream`) and the events `snapshot`, `update`, `freeze`, `contest_ended`, `ping`.
- A `PENDING` problem status for submissions made during the freeze (they are currently hidden).
- A final standings snapshot (`contest_{contestId}_standings_final`) and a NoSQL collection per contest.
- Export of standings (CSV, PDF, Excel).
- A maximum page size.

---

## Optional Notes

- **Problem Display**: The standings header lists problems by `position` with their `slug` and `title`.
- **Tie-breaking**: When problemsSolved and penalty are equal, the earliest last accepted time wins; if that is also equal the rank is shared.
- **Empty Problems**: If a participant has no visible attempt on a problem, it is returned as `NOT_ATTEMPTED`.
