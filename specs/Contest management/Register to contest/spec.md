# Feature Specification: Register to Contest

**Created**: 2026-01-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Member registers to contest in their group (Priority: P1)

As a Member of a group, I want to register to a contest in my group so that I can participate in the programming competition.

**Why this priority**: Registration is essential for participants to join contests. Members need to register before contests start to be able to submit solutions and appear in standings.

**Independent Test**: This user story can be tested independently by consuming the `POST /api/groups/{groupId}/contests/{contestId}/register` endpoint with valid Member authentication, validating that the registration is created and the user can participate.

**Acceptance Scenarios**:

1. **Scenario**: Successful registration to SCHEDULED contest
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is a Member (not Lead) of the group
   * **And** the user is not already registered
   * **When** they submit a registration request
   * **Then** the system creates the registration
   * **And** returns 204 No Content
   * **And** the user can now participate in the contest when it starts

2. **Scenario**: Idempotent registration - already registered
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is a Member of the group
   * **And** the user is already registered to the contest
   * **When** they submit a registration request again
   * **Then** the system returns 204 No Content (idempotent)
   * **And** no duplicate registration is created

3. **Scenario**: Registration fails - contest already started
   * **Given** a contest exists with status ACTIVE or FINISHED
   * **And** the authenticated user is a Member of the group
   * **When** they attempt to register
   * **Then** the system rejects with 400 Bad Request (CONTEST_ALREADY_STARTED)
   * **And** indicates that registration is only allowed before contest starts

4. **Scenario**: Registration fails - Lead attempts registration
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is a Lead of the group
   * **When** they attempt to register
   * **Then** the system rejects with 403 Forbidden (LEADS_CANNOT_REGISTER)
   * **And** indicates that Leads cannot register to contests

5. **Scenario**: Registration fails - Admin attempts registration
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user has Admin role
   * **When** they attempt to register
   * **Then** the system rejects with 403 Forbidden (ADMINS_CANNOT_REGISTER)
   * **And** indicates that Admins cannot register to contests

6. **Scenario**: Registration fails - non-member attempts registration
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is not a member of the group
   * **When** they attempt to register
   * **Then** the system rejects with 403 Forbidden (NOT_GROUP_MEMBER)
   * **And** indicates that only group members can register

7. **Scenario**: Registration fails - contest not found
   * **Given** no contest exists with the provided contestId
   * **And** the authenticated user is a Member of the group
   * **When** they attempt to register
   * **Then** the system rejects with 404 Not Found
   * **And** indicates that the contest does not exist

8. **Scenario**: Registration fails - group not found
   * **Given** no group exists with the provided groupId
   * **And** the authenticated user is authenticated
   * **When** they attempt to register
   * **Then** the system rejects with 404 Not Found
   * **And** indicates that the group does not exist

---

### User Story 2 – Member unregisters from contest (Priority: P1)

As a Member registered to a contest, I want to unregister before the contest starts so that I can withdraw my participation if needed.

**Why this priority**: Members may need to withdraw from contests due to scheduling conflicts or other reasons. Unregistration should be allowed before contests start.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /api/groups/{groupId}/contests/{contestId}/register` endpoint with valid Member authentication, validating that the registration is removed.

**Acceptance Scenarios**:

1. **Scenario**: Successful unregistration from SCHEDULED contest
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is a Member of the group
   * **And** the user is registered to the contest
   * **When** they submit an unregistration request
   * **Then** the system removes the registration
   * **And** returns 204 No Content
   * **And** the user is no longer registered

2. **Scenario**: Unregistration fails - contest already started
   * **Given** a contest exists with status ACTIVE or FINISHED
   * **And** the authenticated user is a Member of the group
   * **And** the user is registered to the contest
   * **When** they attempt to unregister
   * **Then** the system rejects with 400 Bad Request (CANNOT_UNREGISTER_AFTER_START)
   * **And** indicates that unregistration is only allowed before contest starts

3. **Scenario**: Unregistration fails - not registered
   * **Given** a contest exists in a group with status SCHEDULED
   * **And** the authenticated user is a Member of the group
   * **And** the user is NOT registered to the contest
   * **When** they attempt to unregister
   * **Then** the system rejects with 404 Not Found (NOT_REGISTERED)
   * **And** indicates that the user is not registered

4. **Scenario**: Unregistration fails - non-member attempts unregistration
   * **Given** a contest exists in a group
   * **And** the authenticated user is not a member of the group
   * **When** they attempt to unregister
   * **Then** the system rejects with 403 Forbidden (NOT_GROUP_MEMBER)

---

### User Story 3 – Member checks registration status (Priority: P2)

As a Member, I want to check if I am registered to a contest so that I can verify my registration status.

**Why this priority**: Provides a way for users to verify their registration status. Lower priority as it's a convenience feature.

**Independent Test**: This user story can be tested independently by consuming the `GET /api/groups/{groupId}/contests/{contestId}/register/status` endpoint, validating that the registration status is returned correctly.

**Acceptance Scenarios**:

1. **Scenario**: Check registration status - registered
   * **Given** a contest exists in a group
   * **And** the authenticated user is a Member of the group
   * **And** the user is registered to the contest
   * **When** they check registration status
   * **Then** the system returns 200 OK with `registered: true`
   * **And** includes registration timestamp

2. **Scenario**: Check registration status - not registered
   * **Given** a contest exists in a group
   * **And** the authenticated user is a Member of the group
   * **And** the user is NOT registered to the contest
   * **When** they check registration status
   * **Then** the system returns 200 OK with `registered: false`

3. **Scenario**: Check registration status - non-member
   * **Given** a contest exists in a group
   * **And** the authenticated user is not a member of the group
   * **When** they check registration status
   * **Then** the system returns 200 OK with `registered: false`
   * **Note**: Non-members cannot register, so status is always false

---

### User Story 4 – View registered participants list (Priority: P2)

As a Member or Lead of a group, I want to view the list of registered participants for a contest so that I can see who will be participating.

**Why this priority**: Provides visibility into contest participation. Lower priority as it's informational rather than functional.

**Independent Test**: This user story can be tested independently by consuming the `GET /api/groups/{groupId}/contests/{contestId}/registrations` endpoint, validating that the list of registered users is returned with pagination.

**Acceptance Scenarios**:

1. **Scenario**: View registrations list - with pagination
   * **Given** a contest exists in a group
   * **And** the contest has many registered participants (more than page size)
   * **And** the authenticated user is a Member or Lead of the group
   * **When** they request the registrations list
   * **Then** the system returns the first page of registered users
   * **And** includes pagination metadata (total count, hasMore, nextPage)
   * **And** returns only nickname for each registered user
   * **And** orders by registration date (oldest first)

2. **Scenario**: View registrations list - empty contest
   * **Given** a contest exists in a group
   * **And** the contest has no registered participants
   * **And** the authenticated user is a Member or Lead of the group
   * **When** they request the registrations list
   * **Then** the system returns an empty array
   * **And** includes pagination metadata indicating no results

3. **Scenario**: View registrations list - non-member attempts access
   * **Given** a contest exists in a group
   * **And** the authenticated user is not a member of the group
   * **When** they attempt to view the registrations list
   * **Then** the system rejects with 403 Forbidden (NOT_GROUP_MEMBER)

---

### Edge Cases

* User registers, then is removed from group before contest starts (automatic unregistration).
* User registers, then is removed from group after contest starts (remains registered but cannot access).
* Contest modification (date change) - registrations are preserved.
* Contest deletion - all registrations are deleted (cascade).
* Concurrent registration requests (should be idempotent).
* User registers, contest is modified to start earlier (registrations preserved).
* Very large number of registrations (pagination handling).
* User checks status for non-existent contest (404).
* User registers to contest, then contest is deleted (registration removed).

---

## API Contract

### POST /api/groups/{groupId}/contests/{contestId}/register

Register the authenticated user to a contest.

> **Important**: 
> - Only Members (not Leads, not Admin) can register
> - Registration is only allowed for contests with status SCHEDULED
> - Registration is idempotent (registering twice returns success)
> - If user is removed from group before contest starts, registration is automatically removed
> - If user is removed from group after contest starts, registration remains but user cannot access

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Responses**:

#### 204 No Content
Registration successful. User is now registered to the contest.

(No body)

#### 400 Bad Request
Contest has already started.

```json
{
  "error": "CONTEST_ALREADY_STARTED",
  "message": "Registration is only allowed before the contest starts"
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
User doesn't have permission to register.

```json
{
  "error": "LEADS_CANNOT_REGISTER",
  "message": "Leads cannot register to contests"
}
```

```json
{
  "error": "ADMINS_CANNOT_REGISTER",
  "message": "Admins cannot register to contests"
}
```

```json
{
  "error": "NOT_GROUP_MEMBER",
  "message": "Only group members can register to contests"
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

---

### DELETE /api/groups/{groupId}/contests/{contestId}/register

Unregister the authenticated user from a contest.

> **Important**: 
> - Only Members can unregister
> - Unregistration is only allowed before contest starts (status SCHEDULED)
> - Cannot unregister once contest has started (ACTIVE or FINISHED)

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Responses**:

#### 204 No Content
Unregistration successful. User is no longer registered to the contest.

(No body)

#### 400 Bad Request
Contest has already started.

```json
{
  "error": "CANNOT_UNREGISTER_AFTER_START",
  "message": "Unregistration is only allowed before the contest starts"
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
  "error": "NOT_GROUP_MEMBER",
  "message": "Only group members can unregister from contests"
}
```

#### 404 Not Found
Group, contest, or registration not found.

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
  "error": "NOT_REGISTERED",
  "message": "You are not registered to this contest"
}
```

---

### GET /api/groups/{groupId}/contests/{contestId}/register/status

Check if the authenticated user is registered to a contest.

> **Important**: 
> - Any authenticated user can check their registration status
> - Returns `registered: false` for non-members (they cannot register)
> - Returns registration timestamp if registered

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |

**Responses**:

#### 200 OK
Registration status retrieved successfully.

```json
{
  "registered": true,
  "registeredAt": "2026-01-05T10:30:00Z"
}
```

```json
{
  "registered": false
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

---

### GET /api/groups/{groupId}/contests/{contestId}/registrations

Get the list of registered participants for a contest.

> **Important**: 
> - Only Members and Leads of the group can view registrations
> - Returns only nickname for each registered user
> - Ordered by registration date (oldest first)
> - Paginated if there are many registrations

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
| page | integer | No | Page number (default: 1, min: 1) |
| limit | integer | No | Items per page (default: 50, min: 1, max: 100) |

**Responses**:

#### 200 OK
Registrations list retrieved successfully.

```json
{
  "registrations": [
    {
      "nickname": "user1",
      "registeredAt": "2026-01-05T10:00:00Z"
    },
    {
      "nickname": "user2",
      "registeredAt": "2026-01-05T10:15:00Z"
    },
    {
      "nickname": "user3",
      "registeredAt": "2026-01-05T10:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 3,
    "totalPages": 1,
    "hasMore": false
  }
}
```

```json
{
  "registrations": [],
  "pagination": {
    "page": 1,
    "limit": 50,
    "total": 0,
    "totalPages": 0,
    "hasMore": false
  }
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
User is not a member of the group.

```json
{
  "error": "NOT_GROUP_MEMBER",
  "message": "Only group members can view contest registrations"
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

---

## Requirements *(mandatory)*

### Functional Requirements

**Registration**
* **FR-RC-001**: The system MUST allow Members to register to contests in their groups.
* **FR-RC-002**: The system MUST prevent Leads from registering to contests.
* **FR-RC-003**: The system MUST prevent Admins from registering to contests.
* **FR-RC-004**: The system MUST allow registration only for contests with status SCHEDULED.
* **FR-RC-005**: The system MUST reject registration attempts for contests with status ACTIVE or FINISHED.
* **FR-RC-006**: The system MUST make registration idempotent (registering twice returns success without error).
* **FR-RC-007**: The system MUST validate that the user is a member of the group before allowing registration.
* **FR-RC-008**: The system MUST return 204 No Content on successful registration.

**Unregistration**
* **FR-RC-009**: The system MUST allow Members to unregister from contests.
* **FR-RC-010**: The system MUST allow unregistration only for contests with status SCHEDULED.
* **FR-RC-011**: The system MUST reject unregistration attempts for contests with status ACTIVE or FINISHED.
* **FR-RC-012**: The system MUST return 204 No Content on successful unregistration.

**Automatic Unregistration**
* **FR-RC-013**: The system MUST automatically unregister users who are removed from a group if the contest has not started (status SCHEDULED).
* **FR-RC-014**: The system MUST preserve registrations for users removed from a group if the contest has started (status ACTIVE or FINISHED).
* **FR-RC-015**: Users removed from a group after contest starts remain registered but cannot access the contest or make submissions.

**Registration Status**
* **FR-RC-016**: The system MUST allow any authenticated user to check their registration status.
* **FR-RC-017**: The system MUST return `registered: false` for non-members (they cannot register).
* **FR-RC-018**: The system MUST return registration timestamp if user is registered.

**Registrations List**
* **FR-RC-019**: The system MUST allow Members and Leads to view the list of registered participants.
* **FR-RC-020**: The system MUST return only nickname for each registered user.
* **FR-RC-021**: The system MUST order registrations by registration date (oldest first).
* **FR-RC-022**: The system MUST paginate the registrations list if there are many registrations.
* **FR-RC-023**: The system MUST include pagination metadata (page, limit, total, totalPages, hasMore).

**Contest Modifications**
* **FR-RC-024**: The system MUST preserve registrations when contest dates are modified.
* **FR-RC-025**: The system MUST preserve registrations when contest is modified but remains SCHEDULED.

**Contest Deletion**
* **FR-RC-026**: The system MUST delete all registrations when a contest is deleted (cascade delete).

**Validation**
* **FR-RC-036**: The system MUST validate that the contest exists before allowing registration.
* **FR-RC-037**: The system MUST validate that the group exists before allowing registration.
* **FR-RC-038**: The system MUST validate user role (Member, not Lead, not Admin) before allowing registration.

**Response**
* **FR-RC-039**: The system MUST return appropriate error codes for validation and authorization failures.
* **FR-RC-040**: The system MUST NOT return internal IDs except where needed as identifiers.

## Data Architecture *(mandatory)*

### NoSQL Document Structure

Registration and Standing data are stored together in a NoSQL document database (e.g., MongoDB, DynamoDB) using a **one collection per contest** approach for optimal performance.

**Collection Naming**: `contest_{contestId}_standings`

**Unified Document Structure** (supports both individuals and teams):
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "type": "INDIVIDUAL",
  "displayName": "speed_coder",
  "registeredAt": "2026-01-10T10:00:00Z",
  "problemsSolved": 2,
  "penalty": 45,
  "problems": {
    "sum-of-two-numbers": {
      "attempts": 3,
      "acceptedAt": "2026-01-10T11:30:00Z",
      "penalty": 20
    },
    "longest-path": {
      "attempts": 2,
      "acceptedAt": "2026-01-10T12:15:00Z",
      "penalty": 20
    },
    "matrix-multiplication": {
      "attempts": 1,
      "acceptedAt": null,
      "penalty": 0
    }
  },
  "lastUpdated": "2026-01-10T12:15:00Z"
}
```

**Team Document Example** (for TEAM/MIXED modes):
```json
{
  "id": "team-550e8400-e29b-41d4",
  "type": "TEAM",
  "displayName": "Team Alpha",
  "members": ["alice", "bob", "carol"],
  "registeredAt": "2026-01-10T10:00:00Z",
  "problemsSolved": 3,
  "penalty": 80,
  "problems": {...},
  "lastUpdated": "2026-01-10T12:15:00Z"
}
```

**Field Descriptions**:
* `id` (string, UUID): Unique identifier - **either userId or teamId** (document key)
* `type` (enum): `INDIVIDUAL` or `TEAM` - identifies what the id represents
* `displayName` (string): User's nickname or team name (for standings display)
* `members` (array, optional): Team member nicknames (only for TEAM type, if showTeamMembers is true)
* `registeredAt` (timestamp): When the participant registered to the contest
* `problemsSolved` (integer): Total number of problems solved (accepted submissions)
* `penalty` (integer): Total penalty in minutes (sum of penalties from all solved problems)
* `problems` (object): Map of problem slugs to problem-specific data
  * `attempts` (integer): Number of submission attempts for this problem
  * `acceptedAt` (timestamp, nullable): When the first accepted submission occurred (null if not solved)
  * `penalty` (integer): Penalty for this problem = (attempts - 1) * contestPenalty (only set when accepted)
* `lastUpdated` (timestamp): Last time the document was updated

**Penalty Calculation**:
* Penalty is only added when a problem is **first accepted** (acceptedAt is set)
* Problem penalty = (attempts - 1) * contestPenalty
* Total penalty = sum of all problem penalties for solved problems
* Example: If contestPenalty = 20 and user has 3 attempts before acceptance, problem penalty = (3-1) * 20 = 40 minutes

### Collection Management

**Active Collection**: `contest_{contestId}_standings`
* Created when first participant registers (individual or team)
* Updated in real-time during contest
* Used for live standings and ranking queries
* Deleted when contest is deleted

**Final Snapshot**: `contest_{contestId}_standings_final`
* Created when contest ends (FINISHED status)
* Contains immutable snapshot of final standings
* Deleted when contest is deleted (if exists)

### Indexes

**Required Indexes**:
* Primary Key: `id` (unique)
* Ranking Index: `problemsSolved` (descending) + `penalty` (ascending) for efficient ranking queries
* Registration Index: `registeredAt` (ascending) for ordered registration list
* Type Index: `type` for filtering by participant type (optional)

### Atomic Operations

All standing updates use atomic operations to ensure consistency:
* **Increment attempts**: `$inc: { "problems.{slug}.attempts": 1 }`
* **Set accepted**: `$set: { "problems.{slug}.acceptedAt": timestamp, "problems.{slug}.penalty": value }` (only if acceptedAt is null)
* **Update totals**: `$inc: { problemsSolved: 1, penalty: value }` (only on first acceptance)

### Performance Considerations

* **One collection per contest**: Enables efficient queries and horizontal scaling
* **Document-based**: All standing data for a participant in one document (no joins needed)
* **Atomic updates**: Ensures consistency during concurrent submissions
* **Snapshot on finish**: Final standings preserved without affecting active collection performance

---

### Key Entities

* **Standing Document**: Represents a participant's (individual or team) registration and standing data in a NoSQL document.
  * `id` (string, UUID, document key) - **userId OR teamId**
  * `type` (enum: INDIVIDUAL, TEAM)
  * `displayName` (string, nickname or team name)
  * `members` (array, optional, team member nicknames)
  * `registeredAt` (timestamp, when registration was created)
  * `problemsSolved` (integer, total problems solved)
  * `penalty` (integer, total penalty in minutes)
  * `problems` (object, map of problem slugs to problem data)
  * `lastUpdated` (timestamp, last update time)
  * **Storage**: NoSQL collection `contest_{contestId}_standings`
  * **Deletion**: Collection deleted when contest is deleted
  * **Snapshot**: Final snapshot created in `contest_{contestId}_standings_final` when contest ends (deleted when contest is deleted)

> **Registration Rules (Individual)**:
> * Only Members can register (not Leads, not Admin)
> * Registration only allowed for SCHEDULED contests
> * Registration is idempotent
> * Unregistration only allowed for SCHEDULED contests
> * If user removed from group before contest starts: automatic unregistration (document deleted)
> * If user removed from group after contest starts: document preserved but access denied
> * Document created on registration, updated with standing data when contest starts

> **Registration Rules (Team)**:
> * See [Team management specs](../../Team%20management/README.md) for full team registration rules
> * Teams register via separate endpoint with selectedMembers
> * Standing document uses teamId as the `id` field

### Permission Matrix

| Role | Can Register | Can Unregister | Can View List | Can Check Status |
|------|--------------|----------------|---------------|------------------|
| Member | ✅ | ✅ (before start) | ✅ | ✅ |
| Lead | ❌ | ❌ | ✅ | ✅ |
| Admin | ❌ | ❌ | ✅ (if member) | ✅ |
| Non-member | ❌ | ❌ | ❌ | ✅ (returns false) |

### Registration Flow

```
POST /api/groups/{groupId}/contests/{contestId}/register
    ↓
Validate user is authenticated
    ↓
Validate user is Member (not Lead, not Admin)
    ↓
Validate user is member of group
    ↓
Validate contest exists and is SCHEDULED
    ↓
Check if document exists in contest_{contestId}_standings (idempotent)
    ↓
Create document in contest_{contestId}_standings (if not exists):
  {
    contestantId: userId,
    registeredAt: now(),
    problemsSolved: 0,
    penalty: 0,
    problems: {},
    lastUpdated: now()
  }
    ↓
Return 204 No Content
```

### Unregistration Flow

```
DELETE /api/groups/{groupId}/contests/{contestId}/register
    ↓
Validate user is authenticated
    ↓
Validate user is Member
    ↓
Validate contest exists and is SCHEDULED
    ↓
Check if document exists in contest_{contestId}_standings
    ↓
Delete document from contest_{contestId}_standings
    ↓
Return 204 No Content
```

### Standing Update Flow (when contest starts)

```
Contest status changes to ACTIVE
    ↓
For each registered participant (document in contest_{contestId}_standings):
    ↓
Initialize problems map with all contest problems:
  problems: {
    "problem-slug-1": { attempts: 0, acceptedAt: null, penalty: 0 },
    "problem-slug-2": { attempts: 0, acceptedAt: null, penalty: 0 },
    ...
  }
    ↓
Document ready for standing updates
```

### Standing Update Flow (on submission)

```
User submits solution
    ↓
If submission is accepted AND problem not yet solved:
    ↓
Atomic update in contest_{contestId}_standings:
  {
    $inc: {
      "problems.{slug}.attempts": 1,
      problemsSolved: 1,
      penalty: (attempts - 1) * contestPenalty
    },
    $set: {
      "problems.{slug}.acceptedAt": now(),
      "problems.{slug}.penalty": (attempts - 1) * contestPenalty,
      lastUpdated: now()
    }
  }
    ↓
If submission is NOT accepted:
    ↓
Atomic update:
  {
    $inc: { "problems.{slug}.attempts": 1 },
    $set: { lastUpdated: now() }
  }
```

### Snapshot Creation Flow (when contest ends)

```
Contest status changes to FINISHED
    ↓
Copy entire collection contest_{contestId}_standings
    ↓
Create collection contest_{contestId}_standings_final
    ↓
All documents copied to final snapshot
    ↓
Final standings preserved for historical records
```

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-RC-001**: Members can register to SCHEDULED contests via `POST /api/groups/{groupId}/contests/{contestId}/register` with HTTP 204.
* **SC-RC-002**: Leads cannot register to contests - HTTP 403.
* **SC-RC-003**: Admins cannot register to contests - HTTP 403.
* **SC-RC-004**: Registration is rejected for ACTIVE or FINISHED contests - HTTP 400.
* **SC-RC-005**: Registration is idempotent (registering twice returns HTTP 204).
* **SC-RC-006**: Members can unregister from SCHEDULED contests via `DELETE /api/groups/{groupId}/contests/{contestId}/register` with HTTP 204.
* **SC-RC-007**: Unregistration is rejected for ACTIVE or FINISHED contests - HTTP 400.
* **SC-RC-008**: Users can check registration status via `GET /api/groups/{groupId}/contests/{contestId}/register/status` with HTTP 200.
* **SC-RC-009**: Members and Leads can view registrations list via `GET /api/groups/{groupId}/contests/{contestId}/registrations` with HTTP 200.
* **SC-RC-010**: Registrations list is paginated when there are many registrations.
* **SC-RC-011**: Registrations list returns only nickname and ordered by registration date.
* **SC-RC-012**: Non-members cannot view registrations list - HTTP 403.
* **SC-RC-013**: Users removed from group before contest starts are automatically unregistered.
* **SC-RC-014**: Users removed from group after contest starts remain registered but cannot access.
* **SC-RC-015**: Registrations are preserved when contest dates are modified.
* **SC-RC-016**: Collection `contest_{contestId}_standings` is deleted when contest is deleted.
* **SC-RC-017**: Final snapshot collection `contest_{contestId}_standings_final` is deleted when contest is deleted (if exists).
* **SC-RC-018**: Documents are created in NoSQL format with correct structure on registration.
* **SC-RC-019**: Documents are updated atomically when standings change.
* **SC-RC-020**: Final snapshot is created when contest ends.
* **SC-RC-021**: Penalty is calculated correctly (only on first acceptance, (attempts-1) * contestPenalty).
* **SC-RC-022**: Non-existent groups or contests return HTTP 404.
* **SC-RC-023**: Unauthorized requests return HTTP 401.

---

## Optional Notes

* **Idempotency**: Registration is idempotent - registering multiple times has the same effect as registering once.
* **Pagination**: Default page size is 50, maximum is 100. Consider performance implications for contests with thousands of registrations.
* **Automatic Unregistration**: When a user is removed from a group, the system should check for SCHEDULED contests and automatically delete their documents from the standings collection.
* **Access Control**: Users removed from group after contest starts cannot access the contest but their document remains in the collection (for historical accuracy in standings).
* **NoSQL Performance**: One collection per contest enables efficient queries and horizontal scaling. Consider TTL policies for old collections.
* **Atomic Operations**: All standing updates use atomic operations (`$inc`, `$set`) to ensure consistency during concurrent submissions.
* **Snapshot Strategy**: Final snapshot preserves standings without affecting active collection performance. Snapshot is deleted when contest is deleted.
* **Indexing**: Ensure proper indexes on `problemsSolved` (desc) + `penalty` (asc) for efficient ranking queries.
* **Future enhancements**:
  * Registration deadline (register before a specific time)
  * Registration approval (Leads approve registrations)
  * Team registration (register as a team)
  * Registration notifications (notify when contest is about to start)
* **Related specs**:
  * Create Contest: Contest creation
  * Update Contest: Contest modifications (preserves registrations)
  * Delete Contest: Contest deletion (removes registrations)
  * View Contest: Contest details
  * Submit Solution: Making submissions (requires registration)

