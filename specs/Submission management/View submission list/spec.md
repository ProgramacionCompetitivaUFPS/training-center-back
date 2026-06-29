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

## Notes / Implementation hints

* Submissions in list view should NOT include source code for performance
* For contest submissions, determine user's role (participant, team member, Lead, Admin) first
* Team membership is determined by checking `selectedMembers` in `ContestTeamParticipant`
* Consider caching "my submissions" lists as they are frequently accessed
* Index on `submittedBy` and `contestId` for efficient queries
