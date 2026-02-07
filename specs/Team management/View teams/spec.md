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

### User Story 2 - View pending team invitations (Priority: P1)

As a user, I want to see team invitations I've received so that I can accept or reject them.

**Why this priority**: Users need to see and respond to invitations to join teams.

**Acceptance Scenarios**:

1. **Scenario**: User views pending invitations
   * **Given** user has pending team invitations
   * **When** they request their invitations
   * **Then** system returns all pending invitations
   * **And** includes team info and inviter info

2. **Scenario**: User has no pending invitations
   * **Given** user has no pending invitations
   * **When** they request their invitations
   * **Then** system returns an empty list

3. **Scenario**: Invitation was already accepted
   * **Given** user accepted an invitation
   * **When** they request pending invitations
   * **Then** that invitation is NOT in the list

---

### User Story 3 - View team details (Priority: P2)

As a team member, I want to view details of my team so that I can see all members.

**Why this priority**: Team members need to know who else is on the team.

**Acceptance Scenarios**:

1. **Scenario**: Team member views team details
   * **Given** user is a member of a team
   * **When** they request team details
   * **Then** system returns team name, members, and creation info

2. **Scenario**: Non-member tries to view team details
   * **Given** user is NOT a member of the team
   * **When** they request team details
   * **Then** system rejects with 403 Forbidden

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

**Pending Invitations**

* **FR-VT-005**: System MUST provide endpoint for users to list pending team invitations.
* **FR-VT-006**: System MUST include team info (id, name) for each invitation.
* **FR-VT-007**: System MUST include inviter info (id, nickname) for each invitation.
* **FR-VT-008**: System MUST include invitation date and expiration (if applicable).
* **FR-VT-009**: System MUST NOT return accepted or rejected invitations.

**Team Details**

* **FR-VT-010**: System MUST provide endpoint to view team details.
* **FR-VT-011**: Only team members MUST be able to view team details.
* **FR-VT-012**: Team details MUST include all members with their join dates.
* **FR-VT-013**: Team details MUST include pending invitations (for members only).
* **FR-VT-014**: Admin MUST be able to view any team's details.

**Privacy**

* **FR-VT-015**: Teams MUST NOT be searchable publicly.
* **FR-VT-016**: Team access MUST only be through membership or invitation.

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

### GET /api/users/me/team-invitations

List pending team invitations for the authenticated user.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Success Response (200 OK)**:

```json
{
  "invitations": [
    {
      "id": "invitation-uuid",
      "team": {
        "id": "team-uuid",
        "name": "Team Beta"
      },
      "invitedBy": {
        "id": "user-uuid",
        "nickname": "alice_coder"
      },
      "invitedAt": "2026-02-05T10:00:00Z",
      "expiresAt": null
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 1,
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

#### 403 Forbidden

```json
{
  "error": "NOT_TEAM_MEMBER",
  "message": "Only team members can view team details"
}
```

#### 404 Not Found

```json
{
  "error": "TEAM_NOT_FOUND",
  "message": "Team not found"
}
```

---

## Notes / Implementation hints

* Teams are private by design - no public search/discovery
* Users can only find teams through invitations
* Consider caching "my teams" as this is frequently accessed
* Pending invitations should be filtered server-side (not including expired ones)
* Admin access to team details is for support/moderation purposes
