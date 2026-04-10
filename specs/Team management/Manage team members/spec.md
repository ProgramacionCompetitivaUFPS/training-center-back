# Feature Specification: Manage Team Members

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Invite user to team (Priority: P1)

As a team member, I want to invite other users to my team so that we can compete together.

**Acceptance Scenarios**:

1. **Scenario**: Member invites user successfully
   * **Given** requesting user is a member of Team X
   * **When** they invite User B by nickname
   * **Then** system creates a TeamInvitation
   * **And** User B receives a notification
   * **And** returns 201 Created

2. **Scenario**: Invite already-member user
   * **Given** User B is already a member of Team X
   * **When** member tries to invite User B
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)

3. **Scenario**: Invite already-invited user
   * **Given** User B already has a pending invitation to Team X
   * **When** another member tries to invite User B
   * **Then** system rejects with 409 (`ALREADY_INVITED`)

4. **Scenario**: Non-member tries to invite
   * **Given** requesting user is NOT a member of Team X
   * **When** they try to invite someone to Team X
   * **Then** system rejects with 403 (`NOT_TEAM_MEMBER`)

5. **Scenario**: User not found
   * **Given** the specified nickname doesn't exist
   * **When** invitation is attempted
   * **Then** system rejects with 404 (`USER_NOT_FOUND`)

---

### User Story 2 - Accept invitation (Priority: P1)

As a user, I want to accept a team invitation so that I can join the team.

**Acceptance Scenarios**:

1. **Scenario**: User accepts invitation successfully
   * **Given** User B has a pending invitation to Team X
   * **When** User B accepts the invitation
   * **Then** system creates TeamMember record
   * **And** deletes the TeamInvitation
   * **And** returns 200 OK with team details

2. **Scenario**: Accept non-existent invitation
   * **Given** the invitation ID doesn't exist
   * **When** user tries to accept
   * **Then** system rejects with 404 (`INVITATION_NOT_FOUND`)

3. **Scenario**: Accept another user's invitation
   * **Given** invitation belongs to User B
   * **When** User C tries to accept it
   * **Then** system rejects with 403 (`NOT_YOUR_INVITATION`)

---

### User Story 3 - Reject invitation (Priority: P2)

As a user, I want to reject a team invitation if I don't want to join.

**Acceptance Scenarios**:

1. **Scenario**: User rejects invitation successfully
   * **Given** User B has a pending invitation to Team X
   * **When** User B rejects the invitation
   * **Then** system deletes the TeamInvitation
   * **And** returns 204 No Content

---

### User Story 4 - Leave team (Priority: P1)

As a team member, I want to leave a team so that I can join other teams.

**Acceptance Scenarios**:

1. **Scenario**: Member leaves team successfully
   * **Given** User A is a member of Team X
   * **And** Team X is not registered to any ACTIVE contest with User A selected
   * **When** User A leaves Team X
   * **Then** system deletes TeamMember record
   * **And** returns 204 No Content

2. **Scenario**: Cannot leave during active contest
   * **Given** User A is selected for Team X in an ACTIVE contest
   * **When** User A tries to leave Team X
   * **Then** system rejects with 409 (`CANNOT_LEAVE_DURING_ACTIVE_CONTEST`)

3. **Scenario**: Leave removes from scheduled contest selection
   * **Given** User A is selected for Team X in a SCHEDULED contest
   * **When** User A leaves Team X
   * **Then** User A is removed from selectedMembers for that contest
   * **And** returns 204 No Content

4. **Scenario**: Non-member tries to leave
   * **Given** User A is NOT a member of Team X
   * **When** they try to leave
   * **Then** system rejects with 404 (`NOT_TEAM_MEMBER`)

---

### User Story 5 - View pending invitations (Priority: P2)

As a user, I want to view my pending team invitations.

**Acceptance Scenarios**:

1. **Scenario**: User views their invitations
   * **Given** User B has pending invitations to Teams X and Y
   * **When** User B requests their invitations
   * **Then** system returns list of invitations with team details

---

## Requirements *(mandatory)*

### Functional Requirements

* **FR-001**: Any team member CAN invite users to the team.
* **FR-002**: Users MUST accept invitations to become members.
* **FR-003**: Users CAN reject invitations.
* **FR-004**: Users CAN leave teams at any time (except during ACTIVE contests).
* **FR-005**: If user leaves while selected for SCHEDULED contest, MUST be removed from selection.
* **FR-006**: System MUST prevent duplicate invitations.
* **FR-007**: System MUST prevent inviting existing members.

### Key Entities

* **TeamInvitation**
  * `id` (UUID)
  * `teamId` (UUID)
  * `inviteeUserId` (UUID)
  * `invitedBy` (UUID)
  * `createdAt` (timestamp)

---

## API Contract

### POST /api/teams/{teamId}/invitations

Invite a user to the team.

**Path Parameters**:
| Parameter | Type | Description |
|-----------|------|-------------|
| teamId | UUID | Team ID |

**Request Body**:

```json
{
  "nickname": "string"
}
```

**Response 201 Created**:

```json
{
  "id": "invitation-uuid",
  "teamId": "team-uuid",
  "inviteeUser": {
    "id": "user-uuid",
    "nickname": "john_doe"
  },
  "invitedBy": {
    "id": "inviter-uuid",
    "nickname": "jane_doe"
  },
  "createdAt": "2026-02-07T16:00:00Z"
}
```

---

### GET /api/me/team-invitations

View user's pending invitations.

**Response 200 OK**:

```json
{
  "invitations": [
    {
      "id": "invitation-uuid",
      "team": {
        "id": "team-uuid",
        "name": "Team Alpha"
      },
      "invitedBy": {
        "id": "inviter-uuid",
        "nickname": "jane_doe"
      },
      "createdAt": "2026-02-07T16:00:00Z"
    }
  ]
}
```

---

### POST /api/team-invitations/{invitationId}/accept

Accept a team invitation.

**Response 200 OK**:

```json
{
  "team": {
    "id": "team-uuid",
    "name": "Team Alpha",
    "members": [...]
  },
  "joinedAt": "2026-02-07T16:30:00Z"
}
```

---

### DELETE /api/team-invitations/{invitationId}

Reject a team invitation.

**Response 204 No Content**

---

### DELETE /api/teams/{teamId}/members/me

Leave a team.

**Response 204 No Content**

**Response 409 Conflict**:

```json
{
  "error": "CANNOT_LEAVE_DURING_ACTIVE_CONTEST",
  "message": "You are selected to participate in an active contest with this team",
  "details": {
    "contestId": "contest-uuid",
    "contestName": "Regional 2026"
  }
}
```

---

### GET /api/teams/{teamId}/members

View team members.

**Response 200 OK**:

```json
{
  "members": [
    {
      "userId": "user-uuid",
      "nickname": "john_doe",
      "joinedAt": "2026-02-07T16:00:00Z"
    }
  ]
}
```

---

## Notes / Implementation hints

* Use transactions for accept invitation (create member + delete invitation)
* Check for ACTIVE contest participation before allowing leave
* When leaving during SCHEDULED contest, update selectedMembers array atomically
* Consider sending notifications for invitations and team changes

