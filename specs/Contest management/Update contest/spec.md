# Feature Specification: Update Contest

**Created**: 2026-01-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Lead updates contest in their group (Priority: P1)

As a Lead of a group, I want to update contest information (name, description, times, penalty, problems) so that I can adjust contest details before or during the competition.

**Why this priority**: Contest updates are essential for correcting errors, adjusting schedules, and managing problems during active contests. Leads need flexibility to adapt contests to changing circumstances.

**Independent Test**: This user story can be tested independently by consuming the `PUT /api/groups/{groupId}/contests/{contestId}` endpoint with valid Lead authentication, validating that the contest is updated and changes are reflected correctly.

**Acceptance Scenarios**:

1. **Scenario**: Successful contest update - modify name and description
   * **Given** a contest exists in a group
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request with new name and description
   * **Then** the system updates the contest information
   * **And** returns the updated contest data with `updatedAt` timestamp

2. **Scenario**: Successful contest update - extend endTime during active contest
   * **Given** a contest exists and is currently ACTIVE
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request extending endTime by 30 minutes
   * **Then** the system updates the endTime
   * **And** the contest remains ACTIVE (if still within new time range)
   * **And** returns the updated contest data

3. **Scenario**: Successful contest update - reduce endTime during active contest
   * **Given** a contest exists and is currently ACTIVE
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request reducing endTime (must still be in the future)
   * **Then** the system updates the endTime
   * **And** if current time exceeds new endTime, contest status becomes FINISHED
   * **And** returns the updated contest data

4. **Scenario**: Successful contest update - change startTime after contest started
   * **Given** a contest exists and has already started (current time > startTime)
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request changing startTime to a future time
   * **Then** the system updates the startTime
   * **And** contest status becomes SCHEDULED (if current time < new startTime)
   * **And** returns the updated contest data

5. **Scenario**: Successful contest update - add problems to contest
   * **Given** a contest exists
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **And** problems P1 (PUBLIC) and P2 (PRIVATE, user is modifier) exist and are PUBLISHED
   * **When** they submit an update request adding problem slugs [P1, P2] with order positions
   * **Then** the system adds the problems to the contest
   * **And** assigns the specified order positions
   * **And** returns the updated contest data with problem list
   * **Note**: Standing is NOT recalculated when adding problems (no submissions exist yet)

6. **Scenario**: Successful contest update - remove problem from contest
   * **Given** a contest exists with problems P1, P2, P3
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **And** problem P2 has submissions in this contest
   * **When** they submit an update request removing problem slug P2
   * **Then** the system removes P2 from the contest
   * **And** sets `contest_id` to `null` for all submissions to P2 in this contest
   * **And** recalculates Standing if contest is ACTIVE (removing P2 from solved problems count)
   * **And** returns the updated contest data

7. **Scenario**: Successful contest update - reorder problems
   * **Given** a contest exists with problems [P1 (order:1), P2 (order:2), P3 (order:3)]
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request reordering problems [P3, P1, P2]
   * **Then** the system updates problem order to [P3 (order:1), P1 (order:2), P2 (order:3)]
   * **And** returns the updated contest data with reordered problems

8. **Scenario**: Successful contest update - change penalty during active contest
   * **Given** a contest exists and is currently ACTIVE
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request changing penalty from 20 to 30 minutes
   * **Then** the system updates the penalty
   * **And** recalculates Standing with new penalty values
   * **And** returns the updated contest data

9. **Scenario**: Successful contest update - lock contest (owner or admin)
   * **Given** a contest exists (can be in any status)
   * **And** the authenticated user is the contest owner or has Admin role
   * **And** the contest is not locked
   * **When** they submit an update request setting `locked` to true
   * **Then** the system locks the contest
   * **And** prevents all future modifications (except lock/unlock by owner or admin)
   * **And** returns the updated contest data with locked status

10. **Scenario**: Successful contest update - unlock contest (owner or admin)
   * **Given** a contest exists and is locked
   * **And** the authenticated user is the contest owner OR has Admin role
   * **When** they submit an update request setting `locked` to false
   * **Then** the system unlocks the contest
   * **And** allows modifications again
   * **And** returns the updated contest data

11. **Scenario**: Update fails - contest is locked
   * **Given** a contest exists and is locked
   * **And** the authenticated user is a Lead of the group (not owner, not admin)
   * **When** they attempt to update any contest field (except lock/unlock)
   * **Then** the system rejects with 403 Forbidden (CONTEST_LOCKED)
   * **And** indicates that only the owner or admin can unlock the contest

12. **Scenario**: Update fails - non-Lead attempts update
    * **Given** a contest exists in a group
    * **And** the authenticated user is a Member (not Lead) of the group
    * **When** they attempt to update the contest
    * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

13. **Scenario**: Update fails - non-member attempts update
    * **Given** a contest exists in a group
    * **And** the authenticated user is not a member of the group
    * **And** the authenticated user is not an Admin
    * **When** they attempt to update the contest
    * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

14. **Scenario**: Update fails - unlock attempt by non-owner non-admin
   * **Given** a contest exists and is locked
   * **And** the authenticated user is a Lead but NOT the contest owner and NOT an Admin
   * **When** they attempt to unlock the contest
   * **Then** the system rejects with 403 Forbidden (ONLY_OWNER_OR_ADMIN_CAN_UNLOCK)

15. **Scenario**: Update fails - invalid time range
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **When** they submit an update with endTime before startTime
    * **Then** the system rejects with 400 Bad Request (INVALID_TIME_RANGE)

16. **Scenario**: Update fails - startTime in the past
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **When** they submit an update with startTime in the past
    * **Then** the system rejects with 400 Bad Request (START_TIME_IN_PAST)

17. **Scenario**: Update fails - endTime in the past
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **When** they submit an update with endTime in the past
    * **Then** the system rejects with 400 Bad Request (END_TIME_IN_PAST)

18. **Scenario**: Update fails - PRIVATE problem without access
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **And** problem P3 exists with accessibility PRIVATE
    * **And** the user is NOT a modifier of P3
    * **When** they submit an update request adding problem slug P3
    * **Then** the system rejects with 403 Forbidden (PROBLEM_ACCESS_DENIED)
    * **And** indicates which problem(s) the user cannot add

19. **Scenario**: Update fails - problem not published
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **And** problem P4 exists with status DRAFT
    * **When** they submit an update request adding problem slug P4
    * **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_PUBLISHED)

20. **Scenario**: Update fails - problem not found
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **When** they submit an update request adding a non-existent problem slug
    * **Then** the system rejects with 404 Not Found (PROBLEM_NOT_FOUND)

21. **Scenario**: Update fails - empty update payload
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **When** they submit an update request with no fields
    * **Then** the system rejects with 400 Bad Request (NO_FIELDS_TO_UPDATE)

22. **Scenario**: Update fails - attempt to change group_id
    * **Given** a contest exists in group G1
    * **And** the authenticated user is a Lead of group G1
    * **And** the contest is not locked
    * **When** they submit an update request attempting to change group_id
    * **Then** the system ignores the group_id field
    * **And** updates other fields if provided
    * **Note**: group_id is immutable and cannot be changed

23. **Scenario**: Update fails - attempt to change owner_id
    * **Given** a contest exists
    * **And** the authenticated user is a Lead of the group
    * **And** the contest is not locked
    * **When** they submit an update request attempting to change owner_id
    * **Then** the system ignores the owner_id field
    * **And** updates other fields if provided
    * **Note**: owner_id is immutable and cannot be changed

24. **Scenario**: Partial update - only update name
   * **Given** a contest exists
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request with only the name field
   * **Then** the system updates only the name
   * **And** leaves all other fields unchanged
   * **And** returns the updated contest data

25. **Scenario**: Successful contest update - enable postcompetition before contest ends
   * **Given** a contest exists with status SCHEDULED or ACTIVE
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request setting `enablePostContest` to true
   * **Then** the system updates `enablePostContest` to true
   * **And** postcompetition will start automatically after endTime
   * **And** returns the updated contest data

26. **Scenario**: Successful contest update - enable postcompetition after contest ended
   * **Given** a contest exists with status FINISHED
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request setting `enablePostContest` to true
   * **Then** the system updates `enablePostContest` to true
   * **And** postcompetition starts immediately
   * **And** registered users can now submit (submissions won't affect standings)
   * **And** returns the updated contest data

27. **Scenario**: Successful contest update - disable postcompetition (no postcompetition submissions)
   * **Given** a contest exists with `enablePostContest = true`
   * **And** the contest has ended (status FINISHED)
   * **And** there are NO submissions after endTime
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request setting `enablePostContest` to false
   * **Then** the system updates `enablePostContest` to false
   * **And** postcompetition is disabled
   * **And** returns the updated contest data

28. **Scenario**: Update fails - disable postcompetition with existing postcompetition submissions
   * **Given** a contest exists with `enablePostContest = true`
   * **And** the contest has ended (status FINISHED)
   * **And** there are submissions after endTime
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they attempt to set `enablePostContest` to false
   * **Then** the system rejects with 400 Bad Request (POSTCOMPETITION_SUBMISSIONS_EXIST)
   * **And** indicates that postcompetition cannot be disabled because submissions exist after endTime

29. **Scenario**: Update fails - extend endTime when postcompetition is active
   * **Given** a contest exists with `enablePostContest = true`
   * **And** the contest has ended (status FINISHED)
   * **And** postcompetition is active (there may be submissions after endTime)
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they attempt to extend endTime
   * **Then** the system rejects with 400 Bad Request (CANNOT_EXTEND_WITH_POSTCOMPETITION)
   * **And** indicates that endTime cannot be extended when postcompetition is active

30. **Scenario**: Successful contest update - extend endTime when postcompetition is not active
   * **Given** a contest exists with `enablePostContest = false`
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they submit an update request extending endTime
   * **Then** the system updates the endTime
   * **And** if `enablePostContest = true`, postcompetition will start after new endTime
   * **And** returns the updated contest data

31. **Scenario**: Successful contest update - reduce endTime to last submission before original endTime
   * **Given** a contest exists
   * **And** the contest has submissions before endTime
   * **And** the last submission before endTime was at time T
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **And** postcompetition is not active (`enablePostContest = false` OR no submissions after endTime)
   * **When** they submit an update request reducing endTime to T or later
   * **Then** the system updates the endTime
   * **And** returns the updated contest data

32. **Scenario**: Update fails - reduce endTime beyond last submission before original endTime
   * **Given** a contest exists
   * **And** the contest has submissions before endTime
   * **And** the last submission before endTime was at time T
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they attempt to reduce endTime to before time T
   * **Then** the system rejects with 400 Bad Request (ENDTIME_BEFORE_LAST_SUBMISSION)
   * **And** indicates that endTime cannot be reduced beyond the last submission before original endTime

33. **Scenario**: Update fails - modify endTime when postcompetition is active
   * **Given** a contest exists with `enablePostContest = true`
   * **And** the contest has ended (status FINISHED)
   * **And** there are submissions after endTime (postcompetition is active)
   * **And** the authenticated user is a Lead of the group
   * **And** the contest is not locked
   * **When** they attempt to modify endTime (extend or reduce)
   * **Then** the system rejects with 400 Bad Request (CANNOT_MODIFY_ENDTIME_WITH_POSTCOMPETITION)
   * **And** indicates that endTime cannot be modified when postcompetition is active

---

### User Story 2 – Lead updates contest in global group (Priority: P1)

As a Lead of the global group (Admin or assigned Coach), I want to update contests in the global group so that I can manage public competitions effectively.

**Why this priority**: Global contests need the same update capabilities as group contests. Leads managing public competitions require flexibility to adjust contest details.

**Independent Test**: This user story can be tested independently by consuming the `PUT /api/groups/{globalGroupId}/contests/{contestId}` endpoint with Lead authentication (Admin or Coach who is Lead of global group), validating that the contest is updated correctly.

**Acceptance Scenarios**:

1. **Scenario**: Lead (Coach) updates contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Coach role and is a Lead of the global group
   * **And** the contest is not locked
   * **When** they submit a valid update request
   * **Then** the system updates the contest
   * **And** returns the updated contest data

2. **Scenario**: Admin (Lead) updates contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Admin role (automatically Lead of global group)
   * **And** the contest is not locked
   * **When** they submit a valid update request
   * **Then** the system updates the contest
   * **And** returns the updated contest data

3. **Scenario**: Coach (non-Lead) attempts to update contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Coach role but is NOT a Lead of the global group
   * **When** they attempt to update the contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

4. **Scenario**: Contestant attempts to update contest in global group
   * **Given** a contest exists in the global group (`is_default = true`)
   * **And** the authenticated user has Contestant role
   * **When** they attempt to update the contest
   * **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

---

### User Story 3 – Admin updates contest in any group (Priority: P2)

As an Admin, I want to update contests in any group so that I can assist with administrative tasks across the platform.

**Why this priority**: Admin override capability is important for platform management and support, but secondary to normal usage flows.

**Independent Test**: This user story can be tested independently by consuming the `PUT /api/groups/{groupId}/contests/{contestId}` endpoint with Admin authentication on a group where they are not a member.

**Acceptance Scenarios**:

1. **Scenario**: Admin updates contest in group they don't belong to
   * **Given** a contest exists in a group
   * **And** the authenticated user has Admin role (not a member of the group)
   * **And** the contest is not locked
   * **When** they submit a valid update request
   * **Then** the system updates the contest
   * **And** returns the updated contest data

2. **Scenario**: Admin unlocks contest
   * **Given** a contest exists and is locked
   * **And** the authenticated user has Admin role
   * **When** they attempt to unlock the contest
   * **Then** the system unlocks the contest
   * **And** allows modifications again
   * **And** returns the updated contest data

---

### Edge Cases

* Contest update with all fields provided (partial updates should be supported).
* Contest update during high-traffic period (concurrent updates).
* Adding duplicate problem slugs in the same update request (should deduplicate silently).
* Reordering problems with invalid order positions (should validate and reject).
* Adding problem at order position beyond current problem count (should append to end).
* Changing penalty to 0 (should be allowed).
* Changing penalty to maximum value (1440 minutes / 24 hours).
* Locking contest while update is in progress (should prevent the update).
* Unlocking contest and immediately updating (should work).
* Updating contest that has no problems (should allow adding problems).
* Removing all problems from contest (should be allowed).
* Updating contest with very long name or description (should respect max length).
* Concurrent lock/unlock requests (should handle race conditions).
* Updating contest when group is being deleted simultaneously.
* Problem's accessibility changed between validation and update (race condition).
* Test cases updated during contest update (rejudge should use latest test cases).

---

## API Contract

### PUT /api/groups/{groupId}/contests/{contestId}

Update an existing contest within a group.

> **Important**: 
> - For regular groups: only Leads can update contests
> - For global group: only Leads of the global group can update contests (Admin and assigned Coaches)
> - Admin can update contests in any group
> - Locked contests cannot be modified (except lock/unlock by owner or admin)
> - Only contest owner or Admin can lock/unlock contests
> - Removing problems or changing penalty during ACTIVE contests triggers Standing recalculation
> - Adding problems does NOT trigger Standing recalculation (no submissions exist yet)
> - When a problem is removed, submissions to that problem have `contest_id` set to `null`
> - endTime must always be in the future (cannot be set to past)
> - endTime cannot be modified when postcompetition is active (submissions exist after endTime)
> - endTime can only be reduced to time >= last submission before original endTime
> - Postcompetition cannot be disabled if submissions exist after endTime
> - group_id and owner_id are immutable and cannot be changed

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Request Body** (at least one field required):

```json
{
  "name": "Updated Contest Name",
  "description": "Updated contest description",
  "startTime": "2026-01-15T14:00:00Z",
  "endTime": "2026-01-15T19:00:00Z",
  "penalty": 25,
  "enablePostContest": true,
  "problems": [
    {
      "slug": "sum-of-two-numbers",
      "order": 1
    },
    {
      "slug": "new-problem",
      "order": 2
    }
  ],
  "locked": false
}
```

**Partial update example** (only update name):

```json
{
  "name": "New Contest Name"
}
```

**Add problems example** (add to existing problems):

```json
{
  "problems": [
    {
      "slug": "new-problem-1",
      "order": 3
    },
    {
      "slug": "new-problem-2",
      "order": 4
    }
  ]
}
```

**Remove problems example** (remove specific problems):

```json
{
  "problems": [
    {
      "slug": "problem-to-keep-1",
      "order": 1
    },
    {
      "slug": "problem-to-keep-2",
      "order": 2
    }
  ]
}
```

**Reorder problems example**:

```json
{
  "problems": [
    {
      "slug": "problem-3",
      "order": 1
    },
    {
      "slug": "problem-1",
      "order": 2
    },
    {
      "slug": "problem-2",
      "order": 3
    }
  ]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | No* | Contest name (max 200 characters) |
| description | string | No* | Contest description (max 5000 characters) |
| startTime | string (ISO 8601) | No* | Contest start time (must be in the future) |
| endTime | string (ISO 8601) | No* | Contest end time (must be in the future and after startTime). Cannot be modified if postcompetition is active. |
| penalty | integer | No* | Penalty in minutes for wrong submission (min: 0, max: 1440) |
| enablePostContest | boolean | No* | Enable post-competition phase. Cannot be changed from true to false if submissions exist after endTime. |
| problems | array | No* | Array of problem objects with slug and order. If provided, replaces entire problem list. To add/remove, include all problems you want to keep plus new ones. |
| locked | boolean | No* | Lock status. Only owner or admin can change this. When locked, contest cannot be modified (except lock/unlock by owner or admin). |

> *At least one field must be provided in the request.

**Problems array structure**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| slug | string | Yes | Problem slug |
| order | integer | Yes | Order position (1-based, sequential) |

**Responses**:

#### 200 OK
Contest updated successfully.

```json
{
  "id": "c1d2e3f4-g5h6-7890-1234-567890123456",
  "name": "Updated Contest Name",
  "description": "Updated contest description",
  "startTime": "2026-01-15T14:00:00Z",
  "endTime": "2026-01-15T19:00:00Z",
  "duration": 300,
  "penalty": 25,
  "enablePostContest": true,
  "locked": false,
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
      "slug": "new-problem",
      "title": "New Problem",
      "order": 2
    }
  ],
  "problemCount": 2,
  "status": "SCHEDULED",
  "createdAt": "2026-01-03T10:00:00Z",
  "updatedAt": "2026-01-10T15:30:00Z"
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
      "message": "Name exceeds maximum length of 200 characters"
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
  "error": "END_TIME_IN_PAST",
  "message": "End time must be in the future"
}
```

```json
{
  "error": "NO_FIELDS_TO_UPDATE",
  "message": "At least one field must be provided for update"
}
```

```json
{
  "error": "PROBLEM_NOT_PUBLISHED",
  "message": "Cannot add problem 'draft-problem' - problem is not published",
  "problemSlug": "draft-problem"
}
```

```json
{
  "error": "INVALID_PROBLEM_ORDER",
  "message": "Problem order must be sequential starting from 1",
  "details": [
    {
      "slug": "problem-1",
      "order": 1
    },
    {
      "slug": "problem-2",
      "order": 3
    }
  ]
}
```

```json
{
  "error": "POSTCOMPETITION_SUBMISSIONS_EXIST",
  "message": "Cannot disable postcompetition - submissions exist after contest endTime"
}
```

```json
{
  "error": "CANNOT_EXTEND_WITH_POSTCOMPETITION",
  "message": "Cannot extend endTime when postcompetition is active"
}
```

```json
{
  "error": "CANNOT_MODIFY_ENDTIME_WITH_POSTCOMPETITION",
  "message": "Cannot modify endTime when postcompetition is active"
}
```

```json
{
  "error": "ENDTIME_BEFORE_LAST_SUBMISSION",
  "message": "Cannot reduce endTime beyond the last submission before original endTime",
  "lastSubmissionTime": "2026-01-15T18:45:00Z"
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
User doesn't have permission or contest is locked.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only group Leads can update contests in this group"
}
```

```json
{
  "error": "CONTEST_LOCKED",
  "message": "Contest is locked and cannot be modified. Only the contest owner or Admin can lock/unlock it."
}
```

```json
{
  "error": "ONLY_OWNER_OR_ADMIN_CAN_UNLOCK",
  "message": "Only the contest owner or Admin can lock or unlock the contest"
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
Group or contest not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Group not found"
}
```

```json
{
  "error": "NOT_FOUND",
  "message": "Contest not found"
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

**Contest Updates**
* **FR-UC-001**: The system MUST allow Leads to update contests in their groups.
* **FR-UC-002**: The system MUST allow only Leads of the global group to update contests in the global group (Admin is automatically Lead, and Admin can assign other Coaches as Leads).
* **FR-UC-003**: The system MUST allow Admin to update contests in any group.
* **FR-UC-004**: The system MUST support partial updates (only provided fields are updated).
* **FR-UC-005**: The system MUST require at least one field in the update request.
* **FR-UC-006**: The system MUST update the `updatedAt` timestamp on every successful modification.

**Locking Mechanism**
* **FR-UC-007**: The system MUST allow contest owner or Admin to lock the contest (set `locked = true`).
* **FR-UC-008**: The system MUST allow contest owner or Admin to unlock the contest (set `locked = false`).
* **FR-UC-009**: The system MUST prevent all modifications to locked contests (except lock/unlock by owner or admin).
* **FR-UC-010**: The system MUST allow only the contest owner or Admin to change the `locked` field.
* **FR-UC-011**: The system MUST allow locked contests to be viewed (read-only access).

**Time Validation**
* **FR-UC-012**: The system MUST reject updates where startTime is in the past.
* **FR-UC-013**: The system MUST reject updates where endTime is in the past (endTime must always be in the future).
* **FR-UC-014**: The system MUST reject updates where endTime is before or equal to startTime.
* **FR-UC-015**: The system MUST allow changing startTime even after contest has started (but must still be in the future).
* **FR-UC-016**: The system MUST allow extending or reducing endTime during active contests (but endTime must still be in the future).
* **FR-UC-017**: The system MUST store times in UTC format.
* **FR-UC-017.1**: The system MUST reject extending endTime when postcompetition is active (`enablePostContest = true` AND there are submissions after endTime).
* **FR-UC-017.2**: The system MUST reject modifying endTime (extend or reduce) when postcompetition is active (`enablePostContest = true` AND there are submissions after endTime).
* **FR-UC-017.3**: The system MUST allow reducing endTime only to a time equal to or after the last submission before the original endTime.
* **FR-UC-017.4**: The system MUST reject reducing endTime to before the last submission before the original endTime.

**Postcompetition Management**
* **FR-UC-017.5**: The system MUST allow updating `enablePostContest` field.
* **FR-UC-017.6**: The system MUST allow enabling postcompetition before contest ends (will start automatically after endTime).
* **FR-UC-017.7**: The system MUST allow enabling postcompetition after contest ends (starts immediately).
* **FR-UC-017.8**: The system MUST reject disabling postcompetition (`enablePostContest` from true to false) if there are submissions after endTime.
* **FR-UC-017.9**: The system MUST allow disabling postcompetition if there are NO submissions after endTime.
* **FR-UC-017.10**: When `enablePostContest = true` and contest ends, postcompetition phase starts automatically.
* **FR-UC-017.11**: During postcompetition, registered users can submit but submissions do NOT affect standings.
* **FR-UC-017.12**: Submissions during postcompetition are identified by having `contest_id` and `submittedAt > endTime`.

**Problem Management**
* **FR-UC-018**: The system MUST allow adding problems to contests.
* **FR-UC-019**: The system MUST allow removing problems from contests.
* **FR-UC-020**: The system MUST allow reordering problems in contests.
* **FR-UC-021**: The system MUST validate that all problems exist.
* **FR-UC-022**: The system MUST validate that all problems have status PUBLISHED.
* **FR-UC-023**: For PUBLIC problems, any authorized user can add them to a contest.
* **FR-UC-024**: For PRIVATE problems, only problem modifiers (author or assigned modifiers) can add them to a contest.
* **FR-UC-025**: The system MUST validate that problem order is sequential starting from 1.
* **FR-UC-026**: The system MUST silently deduplicate if the same problem slug appears multiple times in the problems array.
* **FR-UC-027**: When problems array is provided, it MUST replace the entire problem list (not merge).
* **FR-UC-028**: The system MUST allow removing problems that have submissions.
* **FR-UC-028.1**: When a problem is removed from a contest, the system MUST set `contest_id` to `null` for all submissions to that problem in the contest.

**Standing Recalculation**
* **FR-UC-029**: When a problem is added to a contest, the system MUST NOT recalculate Standing (no submissions exist yet).
* **FR-UC-030**: When a problem is removed from an ACTIVE contest, the system MUST set `contest_id` to `null` for all submissions to that problem in the contest.
* **FR-UC-031**: When a problem is removed from an ACTIVE contest, the system MUST recalculate Standing (removing the problem from solved counts).
* **FR-UC-032**: When penalty is changed during an ACTIVE contest, the system MUST recalculate Standing with new penalty values.
* **FR-UC-033**: Standing recalculation MUST only occur for ACTIVE contests (not for SCHEDULED or FINISHED contests).

**Penalty Updates**
* **FR-UC-034**: The system MUST allow updating penalty value.
* **FR-UC-035**: The system MUST validate penalty is between 0 and 1440 minutes (24 hours).
* **FR-UC-036**: The system MUST allow changing penalty during active contests.

**Immutable Fields**
* **FR-UC-037**: The system MUST NOT allow changing `group_id` (field is ignored if provided).
* **FR-UC-038**: The system MUST NOT allow changing `owner_id` (field is ignored if provided).
* **FR-UC-039**: The system MUST NOT allow changing `createdAt` (field is ignored if provided).

**Permissions**
* **FR-UC-040**: For regular groups, only Leads can update contests.
* **FR-UC-041**: For the global group, only Leads of the global group can update contests.
* **FR-UC-042**: Admin has implicit permission to update contests in any group.
* **FR-UC-043**: Members who are not Leads MUST NOT be able to update contests in regular groups.
* **FR-UC-044**: Contestants MUST NOT be able to update contests in the global group.

**Response**
* **FR-UC-045**: The system MUST return the updated contest with computed status.
* **FR-UC-046**: The system MUST return the computed duration in minutes.
* **FR-UC-047**: The system MUST NOT return internal IDs except for contest ID.

### Key Entities

* **Contest**: Represents a programming competition.
  * `id` (string, UUID, PK)
  * `name` (string, required, max 200 chars, **mutable**)
  * `description` (string, nullable, max 5000 chars, **mutable**)
  * `startTime` (timestamp, required, **mutable**, must be in future)
  * `endTime` (timestamp, required, **mutable**, must be in future and after startTime, cannot be modified if postcompetition is active)
  * `penalty` (integer, default: 20, range: 0-1440 minutes, **mutable**)
  * `enablePostContest` (boolean, default: false, **mutable**, cannot change from true to false if submissions exist after endTime)
  * `locked` (boolean, default: false, **mutable by owner or admin only**)
  * `group_id` (string, UUID, FK to Group, **immutable**)
  * `owner_id` (string, UUID, FK to User, **immutable**)
  * `createdAt` (timestamp, **immutable**)
  * `updatedAt` (timestamp, nullable, updated on modification)

* **Contest_Problem**: Links problems to contests with ordering.
  * `id` (string, UUID, PK)
  * `contest_id` (string, UUID, FK to Contest)
  * `problem_id` (string, UUID, FK to Problem)
  * `order` (integer, sequential starting from 1, **mutable**)

> **Contest Status** (computed, not stored):
> * `SCHEDULED`: current time < startTime
> * `ACTIVE`: startTime <= current time <= endTime
> * `FINISHED`: current time > endTime

> **Locking Behavior**:
> * When `locked = true`: Contest cannot be modified (except lock/unlock by owner or admin)
> * When `locked = false`: Contest can be modified by authorized users
> * Only contest owner or Admin can change `locked` field (lock or unlock)
> * Locked contests can still be viewed (read-only)

> **Postcompetition Behavior**:
> * When `enablePostContest = true` and contest ends: Postcompetition phase starts automatically
> * When `enablePostContest = true` and contest already ended: Postcompetition starts immediately when enabled
> * During postcompetition: Registered users can submit, but submissions do NOT affect standings
> * Submissions during postcompetition: Identified by `submittedAt > endTime` and have `contest_id`
> * Standing: Frozen at endTime, remains visible but immutable during postcompetition
> * Cannot disable postcompetition: If submissions exist after endTime, `enablePostContest` cannot be changed from true to false
> * Cannot modify endTime: When postcompetition is active (submissions exist after endTime), endTime cannot be extended or reduced
> * Can reduce endTime: Only if no postcompetition submissions exist, and only to time >= last submission before original endTime

### Permission Matrix

| Role | Regular Group (as Lead) | Regular Group (as Member) | Global Group (as Lead) | Global Group (as Member) | Lock/Unlock |
|------|------------------------|--------------------------|----------------------|------------------------|-------------|
| Admin | ✅ | ✅ (implicit Lead) | ✅ (auto Lead) | N/A | ✅ (can unlock any) |
| Coach | ✅ | ❌ | ✅ | ❌ | ✅ (only own contests) |
| Contestant | ❌ | ❌ | ❌ | ❌ | ❌ |

### Problem Accessibility Rules

| Problem Accessibility | Who Can Add to Contest |
|----------------------|----------------------|
| PUBLIC | Any authorized contest updater |
| PRIVATE | Only problem modifiers (author + assigned modifiers) |

### Standing Recalculation

When problems are added to contests:
1. Standing is NOT recalculated (no submissions exist yet for the new problem)
2. New problems will appear in standings once participants start submitting

When problems are removed from ACTIVE contests:
1. System sets `contest_id` to `null` for all submissions to the removed problem in the contest
2. System recalculates Standing removing the problem from solved counts
3. Submissions remain in the system but are no longer associated with the contest

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-UC-001**: Leads can update contests in their groups via `PUT /api/groups/{groupId}/contests/{contestId}` with HTTP 200.
* **SC-UC-002**: Leads of the global group (Admin and assigned Coaches) can update contests in global group with HTTP 200.
* **SC-UC-003**: Admin can update contests in any group with HTTP 200.
* **SC-UC-004**: Contest owner or Admin can lock/unlock contests with HTTP 200.
* **SC-UC-005**: Locked contests cannot be modified (except lock/unlock by owner or admin) - HTTP 403.
* **SC-UC-006**: Non-owner non-admin cannot unlock contests - HTTP 403.
* **SC-UC-007**: Problems can be added/removed/reordered if accessibility rules are satisfied.
* **SC-UC-008**: PUBLIC problems can be added by any authorized user.
* **SC-UC-009**: PRIVATE problems can only be added by their modifiers.
* **SC-UC-010**: Non-PUBLISHED problems are rejected with HTTP 400.
* **SC-UC-011**: Invalid time ranges are rejected with HTTP 400.
* **SC-UC-012**: Times in the past are rejected with HTTP 400.
* **SC-UC-013**: Unauthorized users receive HTTP 403.
* **SC-UC-014**: Non-existent groups or contests receive HTTP 404.
* **SC-UC-015**: Response includes computed status based on current time.
* **SC-UC-016**: Partial updates work correctly (only provided fields are updated).
* **SC-UC-017**: Immutable fields (group_id, owner_id) are ignored if provided.
* **SC-UC-018**: Adding problems to contests does NOT recalculate Standing (no submissions exist yet).
* **SC-UC-019**: Removing problems from ACTIVE contests sets `contest_id` to `null` for affected submissions.
* **SC-UC-020**: Removing problems from ACTIVE contests recalculates Standing.
* **SC-UC-021**: Changing penalty during ACTIVE contests recalculates Standing.
* **SC-UC-022**: Standing recalculation only occurs for ACTIVE contests.
* **SC-UC-023**: Postcompetition can be enabled before or after contest ends.
* **SC-UC-024**: Postcompetition cannot be disabled if submissions exist after endTime.
* **SC-UC-025**: endTime cannot be extended when postcompetition is active.
* **SC-UC-026**: endTime cannot be modified when postcompetition is active.
* **SC-UC-027**: endTime can only be reduced to time >= last submission before original endTime.
* **SC-UC-028**: Submissions during postcompetition do not affect standings.

---

## Optional Notes

* **Idempotency**: Consider making update requests idempotent - submitting the same update twice should have the same effect.
* **Concurrent Updates**: Consider implementing optimistic locking to handle concurrent update requests.
* **Problem List Management**: The `problems` array replaces the entire list. To add problems, include all existing problems plus new ones. To remove, include only problems you want to keep.
* **Future enhancements**:
  * Bulk problem operations (add/remove multiple problems in separate endpoints)
  * Problem visibility control (hide/show problems during contest)
  * Contest cloning/duplication
  * Update history/audit log
* **Related specs**:
  * Create Contest: Initial contest creation
  * Rejudge Submissions: Automatic rejudge mechanism
  * Contest Standings: Standing calculation details
  * Delete Contest: Remove contest
  * View Contest: Contest details and problem list

