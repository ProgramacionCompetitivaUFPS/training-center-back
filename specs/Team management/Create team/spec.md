# Feature Specification: Create Team

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a new team (Priority: P1)

As a user, I want to create a new team so that I can participate in contests with other users.

**Why this priority**: Team creation is the foundation for all team functionality.

**Independent Test**: Authenticated user POST `/api/teams` with team name. Verify team created with user as member.

**Acceptance Scenarios**:

1. **Scenario**: User creates team successfully
   * **Given** requesting user is authenticated
   * **When** user provides a valid team name
   * **Then** system creates the team
   * **And** user is automatically added as a member
   * **And** returns 201 Created with team details

2. **Scenario**: Team name already exists
   * **Given** a team with name "Alpha" already exists
   * **When** user tries to create a team with name "Alpha"
   * **Then** system rejects with 409 (`TEAM_NAME_EXISTS`)

3. **Scenario**: Invalid team name
   * **Given** user provides an empty or too long name
   * **When** team creation is attempted
   * **Then** system rejects with 400 (`VALIDATION_ERROR`)

4. **Scenario**: Unauthenticated user
   * **Given** request has no authentication
   * **When** team creation is attempted
   * **Then** system rejects with 401 (`UNAUTHORIZED`)

---

## Requirements *(mandatory)*

### Functional Requirements

* **FR-001**: System MUST allow any authenticated user to create a team.
* **FR-002**: System MUST require a team name (1-100 characters).
* **FR-003**: System MUST automatically add the creator as a team member.
* **FR-004**: System MUST store the creator's ID (for reference, no special privileges).
* **FR-005**: System MUST validate team name uniqueness.
* **FR-006**: System MUST apply NFKC normalization to team name.
* **FR-007**: System MUST reject team names that are only whitespace.

### Key Entities

* **Team**
  * **Description**: Global entity representing a group of users who compete together.
  * **Core attributes**:
    * `id` (UUID)
    * `name` (string, 1-100 chars, unique)
    * `createdBy` (UUID, reference only)
    * `createdAt` (timestamp)

* **TeamMember**
  * **Description**: Represents membership in a team.
  * **Core attributes**:
    * `id` (UUID)
    * `teamId` (UUID)
    * `userId` (UUID)
    * `joinedAt` (timestamp)

---

## API Contract

### POST /api/teams

Create a new team.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Request Body**:

```json
{
  "name": "string"
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| name | string | Yes | 1-100 characters, unique, cannot be only whitespace |

**Responses**:

#### 201 Created

Team created successfully.

```json
{
  "id": "team-uuid",
  "name": "Team Alpha",
  "createdBy": "user-uuid",
  "createdAt": "2026-02-07T16:00:00Z",
  "members": [
    {
      "userId": "user-uuid",
      "nickname": "john_doe",
      "joinedAt": "2026-02-07T16:00:00Z"
    }
  ]
}
```

#### 400 Bad Request

Invalid team name.

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Team name must be between 1 and 100 characters",
  "details": {
    "field": "name",
    "constraint": "length"
  }
}
```

#### 401 Unauthorized

User is not authenticated.

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 409 Conflict

Team name already exists.

```json
{
  "error": "TEAM_NAME_EXISTS",
  "message": "A team with this name already exists"
}
```

---

## Notes / Implementation hints

* Apply NFKC normalization before uniqueness check
* Trim whitespace from team name
* Creator is added as member in the same transaction
* Team name comparison should be case-insensitive for uniqueness

