# Feature Specification: Register Team to Contest

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Register team to contest (Priority: P1)

As a team member, I want to register my team to a contest so that we can compete together.

**Acceptance Scenarios**:

1. **Scenario**: Successful team registration
   * **Given** Team X has members [A, B, C, D, E]
   * **And** Contest Z allows teams with size 2-3
   * **And** Contest Z is SCHEDULED
   * **When** member A registers Team X with selectedMembers [A, B, C]
   * **Then** system creates ContestTeamParticipant
   * **And** returns 201 Created

2. **Scenario**: Member already registered individually
   * **Given** User A is already registered individually to Contest Z
   * **When** team tries to register with User A in selectedMembers
   * **Then** system rejects with 409 (`MEMBER_ALREADY_REGISTERED`)

3. **Scenario**: Member in another team
   * **Given** User A is selected in Team Y already registered to Contest Z
   * **When** Team X tries to register with User A in selectedMembers
   * **Then** system rejects with 409 (`MEMBER_IN_ANOTHER_TEAM`)

4. **Scenario**: Selected user not team member
   * **Given** User F is NOT a member of Team X
   * **When** Team X tries to register with User F in selectedMembers
   * **Then** system rejects with 400 (`INVALID_SELECTED_MEMBER`)

5. **Scenario**: Team size violation
   * **Given** Contest Z requires exactly 3 members (min=3, max=3)
   * **When** Team X tries to register with only 2 selectedMembers
   * **Then** system rejects with 400 (`INVALID_TEAM_SIZE`)

6. **Scenario**: Group membership required
   * **Given** Contest Z belongs to Group G
   * **And** User A is NOT a member of Group G
   * **When** Team X tries to register with User A in selectedMembers
   * **Then** system rejects with 403 (`MEMBER_NOT_IN_GROUP`)

7. **Scenario**: Contest doesn't allow teams
   * **Given** Contest Z has participationMode = INDIVIDUAL
   * **When** Team X tries to register
   * **Then** system rejects with 400 (`TEAMS_NOT_ALLOWED`)

8. **Scenario**: Contest not SCHEDULED
   * **Given** Contest Z is ACTIVE or FINISHED
   * **When** Team X tries to register
   * **Then** system rejects with 409 (`CONTEST_NOT_OPEN_FOR_REGISTRATION`)

9. **Scenario**: Non-member tries to register team
   * **Given** User F is NOT a member of Team X
   * **When** User F tries to register Team X
   * **Then** system rejects with 403 (`NOT_TEAM_MEMBER`)

---

### User Story 2 - Modify selected members (Priority: P1)

As a team member, I want to change which team members participate before the contest starts.

**Acceptance Scenarios**:

1. **Scenario**: Change selected members while SCHEDULED
   * **Given** Team X is registered to Contest Z (SCHEDULED)
   * **And** current selectedMembers = [A, B, C]
   * **When** member updates to selectedMembers = [A, B, D]
   * **Then** system updates ContestTeamParticipant
   * **And** returns 200 OK

2. **Scenario**: Cannot change during ACTIVE contest
   * **Given** Contest Z is ACTIVE
   * **When** team tries to change selectedMembers
   * **Then** system rejects with 409 (`CONTEST_ALREADY_STARTED`)

---

### User Story 3 - Unregister team from contest (Priority: P2)

As a team member, I want to unregister my team from a contest.

**Acceptance Scenarios**:

1. **Scenario**: Unregister while SCHEDULED
   * **Given** Team X is registered to Contest Z (SCHEDULED)
   * **When** member unregisters Team X
   * **Then** system deletes ContestTeamParticipant
   * **And** returns 204 No Content

2. **Scenario**: Cannot unregister during ACTIVE contest
   * **Given** Contest Z is ACTIVE
   * **When** team tries to unregister
   * **Then** system rejects with 409 (`CONTEST_ALREADY_STARTED`)

---

## Requirements *(mandatory)*

### Functional Requirements

* **FR-001**: System MUST allow team registration to contests that allow teams.
* **FR-002**: System MUST validate selectedMembers are all members of the team.
* **FR-003**: System MUST validate selectedMembers count is within [teamSizeMin, teamSizeMax].
* **FR-004**: System MUST verify no selectedMember is registered individually.
* **FR-005**: System MUST verify no selectedMember is in another registered team.
* **FR-006**: System MUST verify all selectedMembers are group members (for group contests).
* **FR-007**: System MUST allow modifying selectedMembers only for SCHEDULED contests.
* **FR-008**: System MUST lock participation when contest becomes ACTIVE.

### Key Entities

* **ContestTeamParticipant**
  * `id` (UUID)
  * `contestId` (UUID)
  * `teamId` (UUID)
  * `selectedMembers` (UUID[], max contest.teamSizeMax)
  * `registeredAt` (timestamp)

---

## API Contract

### POST /api/contests/{contestId}/team-registrations

Register a team to a contest.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| contestId | UUID | Contest ID |

**Request Body**:

```json
{
  "teamId": "team-uuid",
  "selectedMembers": ["user-a-uuid", "user-b-uuid", "user-c-uuid"]
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| teamId | UUID | Yes | Must be a team the requester belongs to |
| selectedMembers | UUID[] | Yes | Must be team members, within contest size limits |

**Response 201 Created**:

```json
{
  "id": "registration-uuid",
  "contestId": "contest-uuid",
  "team": {
    "id": "team-uuid",
    "name": "Team Alpha"
  },
  "selectedMembers": [
    { "id": "user-a-uuid", "nickname": "alice" },
    { "id": "user-b-uuid", "nickname": "bob" },
    { "id": "user-c-uuid", "nickname": "carol" }
  ],
  "registeredAt": "2026-02-07T16:00:00Z"
}
```

**Response 409 Conflict** (various):

```json
{
  "error": "MEMBER_ALREADY_REGISTERED",
  "message": "User 'alice' is already registered individually to this contest",
  "details": {
    "userId": "user-a-uuid",
    "nickname": "alice",
    "registrationType": "INDIVIDUAL"
  }
}
```

---

### PUT /api/contests/{contestId}/team-registrations/{teamId}

Update selected members for a registered team.

**Request Body**:

```json
{
  "selectedMembers": ["user-a-uuid", "user-b-uuid", "user-d-uuid"]
}
```

**Response 200 OK**: Updated registration.

**Response 409 Conflict**: Contest already started.

---

### DELETE /api/contests/{contestId}/team-registrations/{teamId}

Unregister team from contest.

**Response 204 No Content**

**Response 409 Conflict**: Contest already started.

---

### GET /api/contests/{contestId}/team-registrations

List all registered teams for a contest.

**Response 200 OK**:

```json
{
  "teams": [
    {
      "team": { "id": "team-uuid", "name": "Team Alpha" },
      "selectedMembers": [...],
      "registeredAt": "2026-02-07T16:00:00Z"
    }
  ],
  "total": 15
}
```

---

## Notes / Implementation hints

* Use transactions to atomically check all validations and create registration
* Consider adding an index on `ContestTeamParticipant(contestId, selectedMembers)` for fast lookup
* When contest transitions to ACTIVE, no more modifications allowed
* Validate group membership for ALL selectedMembers if contest belongs to a group

