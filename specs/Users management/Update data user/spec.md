# Feature Specification: Update User Profile

**Created**: 2025-12-13  

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Update user profile (Priority: P1)

As a system contestant, I want to update my personal information so that my profile data remains accurate and up to date.

**Why this priority**: Keeping user information updated is essential for correct identification within the platform and for features that rely on user metadata (rankings, group membership, contest registrations, etc.). This functionality is required for a usable system after registration.

**Independent Test**: This user story can be tested independently by consuming the `PUT /users` endpoint, validating successful updates, validation errors, authorization rules, and business constraints. The user to be updated is always resolved from the authentication token.

**Acceptance Scenarios**:

1. **Scenario**: Successful user update
   - **Given** a user exists in the system and is authenticated
   - **When** valid updated user data is submitted
   - **Then** the system updates the user information and returns the updated data including `updatedAt`

2. **Scenario**: User not found
   - **Given** no user exists for the authenticated token
   - **When** an update request is submitted
   - **Then** the system rejects the operation with a not found error

3. **Scenario**: Empty update payload
   - **Given** the update request does not contain any updatable fields
   - **When** the request is submitted
   - **Then** the system rejects the operation with a validation error indicating that at least one field must be provided

4. **Scenario**: Payload contains extra non-updatable fields
   - **Given** the user exists and is authenticated
   - **When** the update request includes additional fields that are not used for profile updates
   - **Then** the system ignores those fields and updates only the allowed attributes

5. **Scenario**: Idempotent update
   - **Given** the user exists and submits the same values already stored
   - **When** the update request is submitted
   - **Then** the system returns success without modifying the resource state

6. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** an update request is submitted
   - **Then** the system rejects the operation with an authentication error

---

### Edge Cases

- Attempt to update the user with an empty name or a name containing only whitespace.
- Payload includes the email field, but it is ignored since the user identity is resolved from the token.
- Payload includes additional, unknown fields which are safely ignored.
- Use of Unicode characters in name, nickname, or institution fields.
- Concurrent update requests for the same authenticated user.
- Partial updates where only one allowed field is modified.

## API Contract

### PUT /users

Update the authenticated user's profile information.

> **Important**: The system resolves the user identity from the authentication token. The `email`, `id`, `role`, and `createdAt` fields are never accepted from the request and cannot be modified. Any extra fields in the payload are ignored.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for user authentication |

**Request Body** (at least one field required):
```json
{
  "name": "string",
  "nickname": "string",
  "institution": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | No* | User's full name |
| nickname | string | No* | User's display name or alias |
| institution | string | No* | User's institution or organization |

> *At least one field must be provided in the request.

**Responses**:

#### 200 OK
User profile updated successfully.

```json
{
  "email": "user@example.com",
  "name": "Juan Pérez Updated",
  "nickname": "juan_updated",
  "institution": "Updated University",
  "role": "CONTESTANT",
  "createdAt": "2025-12-13T10:00:00Z",
  "updatedAt": "2025-12-14T09:30:00Z"
}
```

> **Note**: The `id` is not returned in the response. The `nickname` is stored in lowercase regardless of input case.

#### 400 Bad Request
Validation error (no updatable fields provided, invalid data).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "At least one updatable field must be provided"
}
```

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 404 Not Found
User not found for the authenticated token.

```json
{
  "error": "NOT_FOUND",
  "message": "User not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow users to update their own profile information.
- **FR-002**: The system MUST identify the user to be updated using the authentication token, not request payload data.
- **FR-003**: The system MUST validate that the authenticated user exists before applying updates.
- **FR-004**: The system MUST allow partial updates of user information.
- **FR-005**: The system MUST ignore non-updatable fields present in the request payload (`id`, `email`, `role`, `createdAt`, and any other unknown fields).
- **FR-006**: The system MUST validate that at least one updatable field is provided in the request.
- **FR-007**: The system MUST persist the original `createdAt` value and MUST NOT modify it during updates.
- **FR-008**: The system MUST update and persist the `updatedAt` timestamp on every successful modification.
- **FR-009**: The system MUST return validation and business errors with a consistent structure and clear messages.
- **FR-010**: The system MUST store nicknames in lowercase, regardless of the input case provided.
- **FR-011**: The system MUST NOT return the user's `id` in the response.

### Key Entities

- **User**: Represents a registered person in the system.  
  Key attributes:
  - `id` (string, UUID, immutable)
  - `email` (string, unique, immutable via this endpoint)
  - `password` (string, hashed, immutable via this endpoint)
  - `name` (string, **mutable**)
  - `nickname` (string, optional, **mutable**, stored in lowercase)
  - `institution` (string, optional, **mutable**)
  - `role` (enum: ADMIN | COACH | CONTESTANT, immutable via this endpoint)
  - `status` (enum: ACTIVE | DEACTIVATED, immutable via this endpoint)
  - `createdAt` (timestamp, immutable)
  - `updatedAt` (timestamp, nullable, updated on modification)

> **Note**: Only mutable fields (`name`, `nickname`, `institution`) can be updated through this feature.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system successfully updates user information and returns HTTP 200 with the updated data.
- **SC-002**: The system rejects update attempts when the authenticated user does not exist.
- **SC-003**: The system ignores extra non-updatable fields (`email`, `id`, `role`, `createdAt`, and others) if present in the request payload.
- **SC-004**: Validation errors include clear messages and a consistent structure.
- **SC-005**: User profile updates can be completed successfully in a single API call.
- **SC-006**: The `createdAt` value remains unchanged after user updates.
- **SC-007**: The `updatedAt` value is updated on every successful modification.

