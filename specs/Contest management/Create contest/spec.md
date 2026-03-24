# Feature Specification: Create Contest

**Created**: 2026-01-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Lead creates contest in their group (Priority: P1)

As a Lead of a group, I want to create a contest within my group so that members can participate in programming competitions.

**Why this priority**: Contest creation is the foundation for the competitive programming functionality of the platform.

**Independent Test**: This user story can be tested independently by consuming the `POST /api/groups/{groupId}/contests` endpoint with valid Lead authentication, validating that the contest is created and associated with the group.

**Acceptance Scenarios**:

1. **Scenario**: Successful contest creation with minimal data
   * **Given** a group exists (not the global group)
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a contest creation request with name, startTime, and endTime
   * **Then** the system creates the contest with default penalty (20 minutes)
   * **And** sets the authenticated user as the contest owner
   * **And** returns the created contest data

2. **Scenario**: Successful contest creation with full data
   * **Given** a group exists (not the global group)
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a contest creation request with name, description, startTime, endTime, and penalty
   * **Then** the system creates the contest with all provided data
   * **And** returns the created contest data

3. **Scenario**: Contest creation with initial problems
   * **Given** a group exists (not the global group)
   * **And** the authenticated user is a Lead of the group
   * **And** problems P1 (PUBLIC) and P2 (PRIVATE, user is modifier) exist
   * **When** they submit a contest creation request including problem slugs [P1, P2]
   * **Then** the system creates the contest with both problems attached
   * **And** returns the created contest data with problem list

4. **Scenario**: Contest creation fails - PRIVATE problem without access
   * **Given** a group exists (not the global group)
   * **And** the authenticated user is a Lead of the group
   * **And** problem P3 exists with accessibility PRIVATE
   * **And** the user is NOT a modifier of P3
   * **When** they submit a contest creation request including problem slug P3
   * **Then** the system rejects with 403 Forbidden (PROBLEM_ACCESS_DENIED)
   * **And** indicates which problem(s) the user cannot add

5. **Scenario**: Member (non-Lead) attempts to create contest
   * **Given** a group exists (not the global group)
   * **And** the authenticated user is a Member (not Lead) of the group
   * **When** they attempt to create a contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

6. **Scenario**: Non-member attempts to create contest in group
   * **Given** a group exists
   * **And** the authenticated user is not a member of the group
   * **When** they attempt to create a contest in that group
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

7. **Scenario**: Invalid time range - endTime before startTime
   * **Given** a group exists
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a contest with endTime earlier than startTime
   * **Then** the system rejects with 400 Bad Request (INVALID_TIME_RANGE)

8. **Scenario**: Invalid time range - startTime in the past
   * **Given** a group exists
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a contest with startTime in the past
   * **Then** the system rejects with 400 Bad Request (START_TIME_IN_PAST)

9. **Scenario**: Problem not found
   * **Given** a group exists
   * **And** the authenticated user is a Lead of the group
   * **When** they submit a contest including a non-existent problem slug
   * **Then** the system rejects with 404 Not Found (PROBLEM_NOT_FOUND)

10. **Scenario**: Problem not published (DRAFT status)
    * **Given** a group exists
    * **And** the authenticated user is a Lead of the group
    * **And** problem P4 exists with status DRAFT
    * **When** they submit a contest including problem slug P4
    * **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_PUBLISHED)

---

### User Story 2 – Lead creates contest in global group (Priority: P1)

As a Lead of the global group (Admin or assigned Coach), I want to create a contest in the global group so that all platform users can participate in public competitions.

**Why this priority**: Global contests are essential for platform-wide events and open competitions.

**Independent Test**: This user story can be tested independently by consuming the `POST /api/groups/{globalGroupId}/contests` endpoint with Lead authentication (Admin or Coach who is lead of global group), validating that the contest is created in the global group.

**Acceptance Scenarios**:

1. **Scenario**: Lead (Coach) creates contest in global group
   * **Given** the global group exists (`is_default = true`)
   * **And** the authenticated user has Coach role and is a Lead of the global group
   * **When** they submit a valid contest creation request for the global group
   * **Then** the system creates the contest in the global group
   * **And** sets the Coach as the contest owner
   * **And** returns the created contest data

2. **Scenario**: Admin (Lead) creates contest in global group
   * **Given** the global group exists (`is_default = true`)
   * **And** the authenticated user has Admin role (automatically Lead of global group)
   * **When** they submit a valid contest creation request for the global group
   * **Then** the system creates the contest in the global group
   * **And** sets the Admin as the contest owner
   * **And** returns the created contest data

3. **Scenario**: Coach (non-Lead) attempts to create contest in global group
   * **Given** the global group exists (`is_default = true`)
   * **And** the authenticated user has Coach role but is NOT a Lead of the global group
   * **When** they attempt to create a contest in the global group
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

4. **Scenario**: Contestant attempts to create contest in global group
   * **Given** the global group exists (`is_default = true`)
   * **And** the authenticated user has Contestant role
   * **When** they attempt to create a contest in the global group
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

---

### User Story 3 – Admin creates contest in any group (Priority: P2)

As an Admin, I want to create a contest in any group so that I can assist with administrative tasks across the platform.

**Why this priority**: Admin override capability is important but secondary to normal usage flows.

**Independent Test**: This user story can be tested independently by consuming the `POST /api/groups/{groupId}/contests` endpoint with Admin authentication on a group where they are not a lead.

**Acceptance Scenarios**:

1. **Scenario**: Admin creates contest in group they don't lead
   * **Given** a group exists
   * **And** the authenticated user has Admin role (has implicit permissions on all groups)
   * **When** they submit a valid contest creation request
   * **Then** the system creates the contest
   * **And** sets the Admin as the contest owner
   * **And** returns the created contest data

---

### Edge Cases

* Contest with duration of exactly 0 minutes (startTime == endTime) - should be rejected
* Contest with very long duration (e.g., 30 days) - should be allowed
* Duplicate problem slugs in the initial problem list - should deduplicate silently
* Very large number of initial problems (e.g., 100+) - should be allowed
* Contest name with special characters and Unicode
* Concurrent contest creation requests in the same group
* Problem's accessibility changed between validation and contest creation (race condition)
* Creating contest when group is being deleted simultaneously

---

## API Contract

### POST /api/groups/{groupId}/contests

Create a new contest within a group.

> **Important**: 
> - For regular groups: only Leads can create contests
> - For global group: only Leads of the global group can create contests (Admin and assigned Coaches)
> - Admin can create contests in any group (has implicit Lead permissions)
> - Problems can be added at creation time if they meet accessibility requirements
> - Problems must be in PUBLISHED status

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |
| Idempotency-Key | string | No | Optional UUID for idempotent retries |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |

**Request Body**:

```json
{
  "name": "Weekly Contest #15",
  "description": "Practice contest focusing on dynamic programming",
  "startTime": "2026-01-10T14:00:00Z",
  "endTime": "2026-01-10T19:00:00Z",
  "penalty": 20,
  "freezeMinutes": 60,
  "enablePostContest": false,
  "problems": ["sum-of-two-numbers", "longest-path", "matrix-multiplication"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Contest name (max 200 characters) |
| description | string | No | Contest description (max 5000 characters) |
| startTime | string (ISO 8601) | Yes | Contest start time (must be in the future) |
| endTime | string (ISO 8601) | Yes | Contest end time (must be after startTime) |
| penalty | integer | No | Penalty in minutes for wrong submission (default: 20, min: 0, max: 1440) |
| freezeMinutes | integer | No | Minutes before endTime to freeze standings (default: 60, null = no freeze) |
| enablePostContest | boolean | No | Enable post-competition phase (default: false). If enabled, registered users can submit after endTime but submissions won't affect standings |
| problems | string[] | No | Array of problem slugs to add to the contest |

**Responses**:

#### 201 Created
Contest created successfully.

```json
{
  "id": "c1d2e3f4-g5h6-7890-1234-567890123456",
  "name": "Weekly Contest #15",
  "description": "Practice contest focusing on dynamic programming",
  "startTime": "2026-01-10T14:00:00Z",
  "endTime": "2026-01-10T19:00:00Z",
  "duration": 300,
  "penalty": 20,
  "freezeMinutes": 60,
  "enablePostContest": false,
  "group": {
    "id": "a1b2c3d4-e5f6-7890-1234-567890123456",
    "name": "Training Camp 2026"
  },
  "owner": {
    "nickname": "coach_john",
    "name": "John Smith"
  },
  "problems": [
    {
      "slug": "sum-of-two-numbers",
      "title": "Sum of Two Numbers",
      "order": 1
    },
    {
      "slug": "longest-path",
      "title": "Longest Path in DAG",
      "order": 2
    },
    {
      "slug": "matrix-multiplication",
      "title": "Matrix Chain Multiplication",
      "order": 3
    }
  ],
  "problemCount": 3,
  "status": "SCHEDULED",
  "createdAt": "2026-01-03T10:00:00Z"
}
```

> **Note**: The `status` field is computed based on current time:
> - `SCHEDULED`: current time < startTime
> - `ACTIVE`: startTime <= current time <= endTime
> - `FINISHED`: current time > endTime

#### 400 Bad Request
Validation errors.

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "name",
      "message": "Name is required"
    }
  ]
}
```

```json
{
  "error": "INVALID_TIME_RANGE",
  "message": "End time must be after start time"
}
```

```json
{
  "error": "START_TIME_IN_PAST",
  "message": "Start time must be in the future"
}
```

```json
{
  "error": "PROBLEM_NOT_PUBLISHED",
  "message": "Cannot add problem 'draft-problem' - problem is not published",
  "problemSlug": "draft-problem"
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
User doesn't have permission.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only group Leads can create contests in this group"
}
```

```json
{
  "error": "PROBLEM_ACCESS_DENIED",
  "message": "Cannot add PRIVATE problem(s) - you are not a modifier",
  "deniedProblems": ["private-problem-1", "private-problem-2"]
}
```

#### 404 Not Found
Group or problem not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Group not found"
}
```

```json
{
  "error": "PROBLEM_NOT_FOUND",
  "message": "Problem 'non-existent-problem' not found",
  "problemSlug": "non-existent-problem"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Contest Creation**
* **FR-CC-001**: The system MUST allow Leads to create contests in their groups.
* **FR-CC-002**: The system MUST allow only Leads of the global group to create contests in the global group (Admin is automatically Lead, and Admin can assign other Coaches as Leads).
* **FR-CC-003**: The system MUST allow Admin to create contests in any group (has implicit Lead permissions).
* **FR-CC-004**: The system MUST set the authenticated user as the contest owner.
* **FR-CC-005**: The system MUST require name, startTime, and endTime fields.
* **FR-CC-006**: The system MUST set default penalty to 20 minutes if not provided.
* **FR-CC-007**: The system MUST validate penalty is between 0 and 1440 minutes.
* **FR-CC-007.1**: The system MUST set default `enablePostContest` to false if not provided.
* **FR-CC-007.2**: The system MUST allow setting `enablePostContest` to true at contest creation.

**Time Validation**
* **FR-CC-008**: The system MUST reject contests where startTime is in the past.
* **FR-CC-009**: The system MUST reject contests where endTime is before or equal to startTime.
* **FR-CC-010**: The system MUST store times in UTC format.

**Problem Attachment**
* **FR-CC-011**: The system MUST allow adding problems at contest creation time.
* **FR-CC-012**: The system MUST validate that all problems exist.
* **FR-CC-013**: The system MUST validate that all problems have status PUBLISHED.
* **FR-CC-014**: For PUBLIC problems, any authorized user can add them to a contest.
* **FR-CC-015**: For PRIVATE problems, only problem modifiers (author or assigned modifiers) can add them to a contest.
* **FR-CC-016**: The system MUST assign sequential order to problems (1, 2, 3, ...).
* **FR-CC-017**: The system MUST silently deduplicate if the same problem slug appears multiple times.

**Permissions**
* **FR-CC-018**: For regular groups, only Leads can create contests.
* **FR-CC-019**: For the global group, only Leads of the global group can create contests.
* **FR-CC-020**: Admin has implicit Lead permission to create contests in any group.
* **FR-CC-021**: Members who are not Leads MUST NOT be able to create contests in any group.
* **FR-CC-022**: Coaches who are not Leads of the global group MUST NOT be able to create contests in the global group.
* **FR-CC-023**: Contestants MUST NOT be able to create contests in any group.

**Response**
* **FR-CC-023**: The system MUST return the created contest with computed status.
* **FR-CC-024**: The system MUST return the computed duration in minutes.
* **FR-CC-025**: The system MUST NOT return internal IDs except for contest ID.

### Key Entities

* **Contest**: Represents a programming competition.
  * `id` (string, UUID, PK)
  * `name` (string, required, max 200 chars)
  * `description` (string, nullable, max 5000 chars)
  * `startTime` (timestamp, required, must be in future at creation)
  * `endTime` (timestamp, required, must be after startTime)
  * `penalty` (integer, default: 20, range: 0-1440 minutes)
  * `enablePostContest` (boolean, default: false)
  * `group_id` (string, UUID, FK to Group)
  * `owner_id` (string, UUID, FK to User)
  * `createdAt` (timestamp)

* **Contest_Problem**: Links problems to contests with ordering.
  * `id` (string, UUID, PK)
  * `contest_id` (string, UUID, FK to Contest)
  * `problem_id` (string, UUID, FK to Problem)
  * `order` (integer, sequential starting from 1)

> **Contest Status** (computed, not stored):
> * `SCHEDULED`: current time < startTime
> * `ACTIVE`: startTime <= current time <= endTime
> * `FINISHED`: current time > endTime

### Permission Matrix

| Role | Regular Group (as Lead) | Regular Group (as Member) | Global Group (as Lead) | Global Group (as Member) |
|------|------------------------|--------------------------|----------------------|------------------------|
| Admin | ✅ | ✅ (implicit Lead) | ✅ (auto Lead) | N/A |
| Coach | ✅ | ❌ | ✅ | ❌ |
| Contestant | ❌ | ❌ | ❌ | ❌ |

### Problem Accessibility Rules

| Problem Accessibility | Who Can Add to Contest |
|----------------------|----------------------|
| PUBLIC | Any authorized contest creator |
| PRIVATE | Only problem modifiers (author + assigned modifiers) |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-CC-001**: Leads can create contests in their groups via `POST /api/groups/{groupId}/contests` with HTTP 201.
* **SC-CC-002**: Leads of the global group (Admin and assigned Coaches) can create contests in global group with HTTP 201.
* **SC-CC-003**: Admin can create contests in any group with HTTP 201.
* **SC-CC-004**: Contest is created with provided data and default penalty if not specified.
* **SC-CC-005**: Problems can be added at creation time if accessibility rules are satisfied.
* **SC-CC-006**: PUBLIC problems can be added by any authorized user.
* **SC-CC-007**: PRIVATE problems can only be added by their modifiers.
* **SC-CC-008**: Non-PUBLISHED problems are rejected with HTTP 400.
* **SC-CC-009**: Invalid time ranges are rejected with HTTP 400.
* **SC-CC-010**: Unauthorized users receive HTTP 403.
* **SC-CC-011**: Non-existent groups or problems receive HTTP 404.
* **SC-CC-012**: Response includes computed status based on current time.
* **SC-CC-013**: Contest owner is set to the authenticated user.

---

## Optional Notes

* **Idempotency**: If `Idempotency-Key` header is provided and a request with the same key was already processed, return the same response without creating a duplicate.
* **Duration limits**: Consider enforcing minimum (e.g., 5 minutes) and maximum (e.g., 30 days) contest duration.
* **Problem order**: Problems are ordered by the sequence provided in the request array.
* **Future enhancements**:
  * Contest visibility (public/private within group)
  * Different scoring modes (ICPC, IOI, custom)
  * Registration requirements
  * Team contests
* **Related specs**:
  * Update Contest: Modify contest details
  * Manage Contest Problems: Add/remove problems after creation
  * Register to Contest: Participant registration
  * Delete Contest: Remove contest

