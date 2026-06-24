# Feature Specification: View Submission List

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View my submissions (Priority: P1)

As a user, I want to see a list of all my submissions so that I can track my progress and review my attempts.

**Why this priority**: Users need to see their submission history to track progress across problems.

**Independent Test**: Authenticated user GET `/api/users/me/submissions`. Verify paginated list of user's submissions.

**Acceptance Scenarios**:

1. **Scenario**: User views all their submissions
   * **Given** user has multiple submissions across problems
   * **When** they request their submissions list
   * **Then** system returns paginated list of their submissions
   * **And** sorted by submittedAt descending (newest first)

2. **Scenario**: User filters by verdict
   * **Given** user has submissions with different verdicts
   * **When** they request with `?verdict=ACCEPTED`
   * **Then** system returns only ACCEPTED submissions

3. **Scenario**: User filters by problem
   * **Given** user has submissions to multiple problems
   * **When** they request with `?problemSlug=sum-of-two-numbers`
   * **Then** system returns only submissions to that problem

4. **Scenario**: User filters by language
   * **Given** user has submissions in multiple languages
   * **When** they request with `?language=cpp20`
   * **Then** system returns only C++ submissions

5. **Scenario**: User filters by date range
   * **Given** user has submissions across different dates
   * **When** they request with `?from=2026-01-01&to=2026-01-31`
   * **Then** system returns only submissions in that range

---

### User Story 2 - View public submissions for a problem (Priority: P1)

As a user, I want to see public submissions for a problem so that I can learn from other solutions.

**Why this priority**: Learning from others' solutions is a key feature for improvement.

**Acceptance Scenarios**:

1. **Scenario**: User views public submissions for a problem (outside contest)
   * **Given** a problem has PUBLIC submissions from various users
   * **When** user requests public submissions for that problem
   * **Then** system returns only PUBLIC submissions
   * **And** excludes contest submissions

2. **Scenario**: Admin views all submissions for a problem
   * **Given** a problem has PUBLIC and PRIVATE submissions
   * **And** requesting user is Admin
   * **When** they request submissions for that problem
   * **Then** system returns ALL submissions (treats all as public)

---

### User Story 3 - View submissions in a contest (Priority: P1)

> ⚠️ **SUPERSEDED (2026-06-21)** — El modelo de parámetro `scope` (`mine`/`team`/`public`/`all`) y el error `SCOPE_NOT_ALLOWED` descritos en esta historia y en su contrato (`GET /api/groups/{groupId}/contests/{contestId}/submissions`, ver más abajo) **NO** se implementaron. El endpoint sigue la spec **`Contest management/View contest submissions`**, que usa lógica de *freeze* + filtros `problemSlug`/`nickname`/`phase` en lugar de `scope`. Esta sección queda reemplazada por esa spec. Ver `.agent/audit/01-submission-management.md` §3.

As a contest participant, I want to see submissions in a contest so that I can review my and my team's progress.

**Why this priority**: Contest submission visibility is essential for team coordination and post-contest analysis.

**Acceptance Scenarios**:

1. **Scenario**: User views their own submissions in a contest
   * **Given** user is registered to a contest
   * **And** has submitted solutions
   * **When** they request their contest submissions
   * **Then** system returns all their submissions in that contest

2. **Scenario**: User views their team's submissions in a contest
   * **Given** user is in a team registered to a contest
   * **And** teammates have submitted solutions
   * **When** they request team submissions
   * **Then** system returns all submissions from team members

3. **Scenario**: User views public submissions in finished contest
   * **Given** a contest is FINISHED
   * **And** has PUBLIC submissions from participants
   * **When** user requests public submissions
   * **Then** system returns PUBLIC submissions from all participants

4. **Scenario**: Lead views all submissions in their group's contest
   * **Given** user is Lead of a group
   * **And** a contest exists in that group
   * **When** Lead requests contest submissions
   * **Then** system returns ALL submissions (Lead sees all as public)

5. **Scenario**: Non-participant cannot view submissions in active contest
   * **Given** a contest is ACTIVE
   * **And** user is NOT registered
   * **When** they request contest submissions
   * **Then** system rejects with 403 Forbidden

---

## Requirements *(mandatory)*

### Functional Requirements

**General Submission List (My Submissions)**

* **FR-VSL-001**: System MUST provide endpoint for users to list their own submissions.
* **FR-VSL-002**: System MUST paginate results (default 20, max 100 per page).
* **FR-VSL-003**: System MUST sort by `submittedAt` descending by default.
* **FR-VSL-004**: System MUST support filtering by: verdict, problemSlug, language, dateFrom, dateTo.
* **FR-VSL-005**: System MUST return summary info (no source code in list view).

**Problem Submissions (Outside Contest)**

* **FR-VSL-006**: System MUST provide endpoint to list submissions for a specific problem.
* **FR-VSL-007**: System MUST only return PUBLIC submissions for non-admin users.
* **FR-VSL-008**: System MUST exclude contest submissions from this view.
* **FR-VSL-009**: Admin users MUST see all submissions regardless of visibility.

**Contest Submissions**

* **FR-VSL-010**: System MUST provide endpoint to list submissions in a contest.
* **FR-VSL-011**: Participants MUST see their own submissions.
* **FR-VSL-012**: Team members MUST see all submissions from their team.
* **FR-VSL-013**: In FINISHED contests, participants MUST see PUBLIC submissions from all participants.
* **FR-VSL-014**: Lead MUST see all submissions in contests within their group.
* **FR-VSL-015**: Admin MUST see all submissions.
* **FR-VSL-016**: Non-registered users MUST NOT see submissions in ACTIVE contests.

**Filters**

* **FR-VSL-017**: System MUST support filtering by verdict (ACCEPTED, WRONG_ANSWER, etc.).
* **FR-VSL-018**: System MUST support filtering by language (cpp20, java17, python310).
* **FR-VSL-019**: System MUST support filtering by problem (for contest submissions).
* **FR-VSL-020**: System MUST support filtering by date range.

### Submission Summary Entity

Each submission in list view contains:
* `id` (UUID)
* `status` (enum)
* `visibility` (enum)
* `submittedAt` (timestamp)
* `problem` ({ slug, title })
* `contest` ({ id, name } or null)
* `submittedBy` ({ id, nickname })
* `language` (string)
* `executionTime` (integer, nullable)
* `memoryKb` (integer, nullable)

**Note**: Source code is NOT included in list view for performance.

---

## API Contract

### GET /api/users/me/submissions

List all submissions from the authenticated user.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| page | integer | No | 1 | Page number (1-indexed) |
| limit | integer | No | 20 | Items per page (max 100) |
| verdict | string | No | - | Filter by verdict |
| problemSlug | string | No | - | Filter by problem |
| language | string | No | - | Filter by language |
| from | date | No | - | Filter from date (inclusive) |
| to | date | No | - | Filter to date (inclusive) |
| sort | string | No | submittedAt:desc | Sort order |

**Success Response (200 OK)**:

```json
{
  "submissions": [
    {
      "id": "submission-uuid",
      "status": "ACCEPTED",
      "visibility": "PRIVATE",
      "submittedAt": "2026-02-07T14:30:00Z",
      "problem": {
        "slug": "sum-of-two-numbers",
        "title": "Sum of Two Numbers"
      },
      "contest": null,
      "submittedBy": {
        "id": "user-uuid",
        "nickname": "john_doe"
      },
      "language": "cpp20",
      "executionTime": 45,
      "memoryKb": 12288
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 150,
    "totalPages": 8,
    "hasNextPage": true,
    "hasPrevPage": false
  },
  "filters": {
    "verdict": null,
    "problemSlug": null,
    "language": null,
    "from": null,
    "to": null
  }
}
```

---

### GET /api/problems/{problemSlug}/submissions

List public submissions for a problem (outside contests).

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| problemSlug | string | Yes | The problem slug |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| page | integer | No | 1 | Page number |
| limit | integer | No | 20 | Items per page |
| verdict | string | No | - | Filter by verdict |
| language | string | No | - | Filter by language |
| mine | boolean | No | false | Include only my submissions |

**Notes**:
* Returns only PUBLIC submissions by default
* Admin sees ALL submissions
* Use `?mine=true` to see only your submissions (regardless of visibility)

---

### GET /api/groups/{groupId}/contests/{contestId}/submissions

> ⚠️ **SUPERSEDED** — El parámetro `scope` y el error `SCOPE_NOT_ALLOWED` de abajo no se implementaron. Ver la spec vigente `Contest management/View contest submissions` (freeze + `problemSlug`/`nickname`/`phase`).

List submissions in a contest.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group ID |
| contestId | UUID | Yes | The contest ID |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| page | integer | No | 1 | Page number |
| limit | integer | No | 20 | Items per page |
| scope | enum | No | mine | `mine`, `team`, `public`, `all` |
| problemSlug | string | No | - | Filter by problem |
| verdict | string | No | - | Filter by verdict |
| language | string | No | - | Filter by language |

**Scope Values**:

| Scope | Who Can Use | Description |
|-------|-------------|-------------|
| `mine` | Registered user | Only my submissions |
| `team` | Team member | All team submissions |
| `public` | Any registered user (FINISHED) | PUBLIC submissions in finished contest |
| `all` | Lead, Admin | All submissions |

**Error Responses**:

#### 403 Forbidden

```json
{
  "error": "ACCESS_DENIED",
  "message": "You must be registered to the contest to view submissions"
}
```

```json
{
  "error": "SCOPE_NOT_ALLOWED",
  "message": "The 'all' scope is only available for Leads and Admins"
}
```

---

## Notes / Implementation hints

* Submissions in list view should NOT include source code for performance
* For contest submissions, determine user's role (participant, team member, Lead, Admin) first
* Team membership is determined by checking `selectedMembers` in `ContestTeamParticipant`
* Consider caching "my submissions" lists as they are frequently accessed
* Index on `submittedBy` and `contestId` for efficient queries
