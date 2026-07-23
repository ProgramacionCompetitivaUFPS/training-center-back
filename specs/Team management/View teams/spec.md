# Feature Specification: View Teams

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View my teams (Priority: P1)

As a user, I want to see a list of teams I belong to so that I can manage my team memberships.

**Why this priority**: Users need to know which teams they are part of to participate in contests.

**Independent Test**: Authenticated user GET `/api/users/me/teams`. Verify list of teams where user is a member.

**Acceptance Scenarios**:

1. **Scenario**: User views their teams
   * **Given** user is a member of multiple teams
   * **When** they request their teams list
   * **Then** system returns all teams where they are a member
   * **And** includes team name, member count, and join date

2. **Scenario**: User has no teams
   * **Given** user is not a member of any team
   * **When** they request their teams list
   * **Then** system returns an empty list

3. **Scenario**: Unauthenticated request
   * **Given** request has no authentication
   * **When** teams list is requested
   * **Then** system rejects with 401 Unauthorized

---

### User Story 2 - View team details (Priority: P2)

As an authenticated user, I want to view the details of any team so that I can see its members before accepting an invitation or checking contest standings.

**Why this priority**: Team composition is public information in a competitive programming context — any authenticated user should be able to look up who is on a team.

**Acceptance Scenarios**:

1. **Scenario**: Any authenticated user views team details
   * **Given** user is authenticated (member or non-member)
   * **When** they request team details
   * **Then** system returns team name, members, and creation info

2. **Scenario**: Unauthenticated request
   * **Given** request has no authentication
   * **When** team details are requested
   * **Then** system rejects with 401 Unauthorized

3. **Scenario**: Team not found
   * **Given** no team exists with the provided ID
   * **When** user requests team details
   * **Then** system rejects with 404 Not Found

---

## Requirements *(mandatory)*

### Functional Requirements

**My Teams**

* **FR-VT-001**: System MUST provide endpoint for users to list teams they belong to.
* **FR-VT-002**: System MUST return team name and member count for each team.
* **FR-VT-003**: System MUST return the user's join date for each team.
* **FR-VT-004**: System MUST sort teams by join date descending by default.

**Team Details**

* **FR-VT-010**: System MUST provide endpoint to view team details.
* **FR-VT-011**: Any authenticated user MUST be able to view team details.
* **FR-VT-012**: Team details MUST include all members with their join dates.
* **FR-VT-013**: Team details MUST include pending invitations (for members only).
* **FR-VT-014**: Unauthenticated requests MUST be rejected with 401.

**Privacy**

* **FR-VT-015**: Teams MUST NOT be searchable or discoverable publicly (no list-all endpoint).
* **FR-VT-016**: Users can only find teams by direct ID (e.g., from an invitation or standings).

---

## API Contract

### GET /api/users/me/teams

List teams where the authenticated user is a member.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Query Parameters**:

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| page | integer | No | 1 | Page number |
| limit | integer | No | 20 | Items per page |

**Success Response (200 OK)**:

```json
{
  "teams": [
    {
      "id": "team-uuid",
      "name": "Team Alpha",
      "memberCount": 5,
      "joinedAt": "2026-01-15T10:00:00Z",
      "createdAt": "2026-01-10T08:00:00Z"
    },
    {
      "id": "team-uuid-2",
      "name": "Competitive Coders",
      "memberCount": 3,
      "joinedAt": "2026-02-01T14:00:00Z",
      "createdAt": "2026-01-20T12:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 2,
    "totalPages": 1
  }
}
```

---

### GET /api/teams/{teamId}

View details of a specific team.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| teamId | UUID | Yes | The team ID |

**Success Response (200 OK)**:

```json
{
  "id": "team-uuid",
  "name": "Team Alpha",
  "createdBy": {
    "id": "user-uuid",
    "nickname": "john_doe"
  },
  "createdAt": "2026-01-10T08:00:00Z",
  "members": [
    {
      "id": "user-uuid-1",
      "nickname": "john_doe",
      "joinedAt": "2026-01-10T08:00:00Z"
    },
    {
      "id": "user-uuid-2",
      "nickname": "alice_coder",
      "joinedAt": "2026-01-15T10:00:00Z"
    },
    {
      "id": "user-uuid-3",
      "nickname": "bob_dev",
      "joinedAt": "2026-01-20T14:00:00Z"
    }
  ],
  "pendingInvitations": [
    {
      "id": "invitation-uuid",
      "invitee": {
        "id": "user-uuid-4",
        "nickname": "carol_prog"
      },
      "invitedBy": {
        "id": "user-uuid-1",
        "nickname": "john_doe"
      },
      "invitedAt": "2026-02-05T10:00:00Z"
    }
  ]
}
```

**Error Responses**:

#### 404 Not Found

```json
{
  "error": "TEAM_NOT_FOUND",
  "message": "Team not found"
}
```

---

## Notes / Implementation hints

* Team details are visible to any authenticated user — team composition is treated as public information (similar to user profiles)
* Teams are not publicly searchable/discoverable; a user must know the team ID (from an invitation, standings, etc.)
* Consider caching "my teams" as this is frequently accessed
* Pending invitations in the detail response should be filtered server-side (not including expired ones) — implemented in T-2
