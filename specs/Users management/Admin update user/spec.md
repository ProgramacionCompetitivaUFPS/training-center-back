# Feature Specification: Admin Update User

**Created**: 2025-12-13  

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Update user as admin (Priority: P1)

As a system administrator, I want to update another user's information (including role and email) so that I can manage permissions, recover access, and keep user data consistent across the platform.

**Why this priority**: Administrators are responsible for maintaining the integrity and operability of the system. The ability to update user roles is required to promote competitors to coaches or administrators, and updating emails is necessary to recover user access in cases where the original email is lost.

**Independent Test**: This user story can be tested independently by consuming the `PUT /admin/users/{id}` endpoint, validating successful updates, validation errors, authorization rules, and business constraints. The target user is resolved from the `{id}` path parameter, and the acting user is resolved from the authentication token.

**Acceptance Scenarios**:

1. **Scenario**: Successful admin user update
   - **Given** a user exists in the system
   - **And** the authenticated user has the ADMIN role
   - **When** valid updated user data is submitted
   - **Then** the system updates the target user information and returns the updated data including `updatedAt`

2. **Scenario**: Target user not found
   - **Given** no user exists with the provided identifier
   - **And** the authenticated user has the ADMIN role
   - **When** an update request is submitted
   - **Then** the system rejects the operation with a not found error

3. **Scenario**: Non-admin attempts update
   - **Given** a user exists in the system
   - **And** the authenticated user does not have the ADMIN role
   - **When** an admin update request is submitted
   - **Then** the system rejects the operation with a forbidden error

4. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** an admin update request is submitted
   - **Then** the system rejects the operation with an authentication error

5. **Scenario**: Empty update payload
   - **Given** the target user exists
   - **And** the authenticated user has the ADMIN role
   - **When** the update request does not contain any updatable fields
   - **Then** the system rejects the operation with a validation error indicating that at least one field must be provided

6. **Scenario**: Update user role
   - **Given** a user exists with role CONTESTANT
   - **And** the authenticated user has the ADMIN role
   - **When** the update request sets the role to COACH
   - **Then** the system updates the user role accordingly

7. **Scenario**: Attempt to assign ADMIN role
   - **Given** a user exists in the system
   - **And** the authenticated user has the ADMIN role
   - **When** the update request attempts to set the role to ADMIN
   - **Then** the system rejects the operation with a validation error

8. **Scenario**: Update user email
   - **Given** a user exists in the system
   - **And** the authenticated user has the ADMIN role
   - **When** the update request includes a new valid email
   - **Then** the system updates the user email and persists the change

9. **Scenario**: Duplicate email update
   - **Given** another user already exists with the provided email
   - **And** the authenticated user has the ADMIN role
   - **When** the admin attempts to update the target user's email
   - **Then** the system rejects the operation with a validation error indicating that the email is already in use

10. **Scenario**: Payload contains extra non-updatable fields
    - **Given** the target user exists
    - **And** the authenticated user has the ADMIN role
    - **When** the update request includes additional fields that are not supported by the admin update operation
    - **Then** the system ignores those fields and updates only the allowed attributes

11. **Scenario**: Idempotent admin update
    - **Given** the target user exists
    - **And** the authenticated user submits the same values already stored
    - **When** the update request is submitted
    - **Then** the system returns success without modifying the resource state

---

### Edge Cases

- Attempt to update the user with an empty name or a name containing only whitespace.
- Attempt to update the email to the same current value.
- Attempt to assign an unsupported role value.
- Use of Unicode characters in name, nickname, or institution fields.
- Concurrent admin update requests for the same target user.
- Partial updates where only one allowed field is modified.
- Admin attempts to update their own profile through this endpoint.
- Invalid UUID format in the path parameter.

## API Contract

### PUT /admin/users/{id}

Update a user's information as an administrator.

> **Important**: The acting user is resolved from the authentication token and MUST have the ADMIN role. The target user is resolved from the `{id}` path parameter. Extra fields in the payload are ignored.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for admin authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string (UUID) | Yes | The unique identifier of the user to update |

**Request Body** (at least one field required):
```json
{
  "email": "string",
  "name": "string",
  "nickname": "string",
  "institution": "string",
  "role": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | No* | New email address (must be unique) |
| name | string | No* | User's full name |
| nickname | string | No* | User's display name or alias |
| institution | string | No* | User's institution or organization |
| role | string | No* | User's role (CONTESTANT, COACH). Cannot be set to ADMIN. |

> *At least one field must be provided in the request.

**Responses**:

#### 200 OK
User updated successfully by admin.

```json
{
  "email": "newmail@example.com",
  "name": "Juan Pérez Updated",
  "nickname": "juan_updated",
  "institution": "Updated University",
  "role": "COACH",
  "createdAt": "2025-12-13T10:00:00Z",
  "updatedAt": "2025-12-14T09:30:00Z"
}
```

> **Note**: The `id` is not returned in the response. The `nickname` is stored in lowercase regardless of input case.

#### 400 Bad Request
Validation error (no updatable fields, invalid email format, duplicate email, invalid role).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "email",
      "message": "Email already exists"
    }
  ]
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "role",
      "message": "Cannot assign ADMIN role through this endpoint"
    }
  ]
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

#### 403 Forbidden
The authenticated user does not have admin privileges.

```json
{
  "error": "FORBIDDEN",
  "message": "Admin privileges required"
}
```

#### 404 Not Found
Target user not found.

```json
{
  "error": "NOT_FOUND",
  "message": "User not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow administrators to update other users' profile information.
- **FR-002**: The system MUST authorize the operation only for users with the ADMIN role.
- **FR-003**: The system MUST validate that the target user exists before applying updates.
- **FR-004**: The system MUST allow partial updates of user information.
- **FR-005**: The system MUST allow administrators to update the user's email and role.
- **FR-006**: The system MUST validate email format and ensure email uniqueness when updating the email.
- **FR-007**: The system MUST validate that the role value is supported by the system and MUST NOT allow assigning the ADMIN role through this feature.
- **FR-008**: The system MUST ignore non-updatable or unknown fields present in the request payload (`id`, `password`, `createdAt`, and any unknown fields).
- **FR-009**: The system MUST persist the original `createdAt` value and MUST NOT modify it during updates.
- **FR-010**: The system MUST update and persist the `updatedAt` timestamp on every successful modification.
- **FR-011**: The system MUST return validation, authorization, and business errors with a consistent structure and clear messages.
- **FR-012**: The system MUST store nicknames in lowercase, regardless of the input case provided.
- **FR-013**: The system MUST NOT return the user's `id` in the response.

### Key Entities

- **User**: Represents a registered person in the system.  
  Key attributes:
  - `id` (string, UUID, immutable)
  - `email` (string, unique, **mutable by admin**)
  - `password` (string, hashed, immutable via this endpoint)
  - `name` (string, **mutable**)
  - `nickname` (string, optional, **mutable**)
  - `institution` (string, optional, **mutable**)
  - `role` (string, **mutable by admin**, except ADMIN)
  - `createdAt` (timestamp, immutable)
  - `updatedAt` (timestamp, nullable, updated on modification)

> **Note**: Through this feature, administrators can update mutable fields (`name`, `nickname`, `institution`) and privileged fields (`email`, `role`), except assigning the ADMIN role.

### Supported Roles

| Role | Description | Assignable via this endpoint |
|------|-------------|------------------------------|
| CONTESTANT | Regular participant | ✅ Yes |
| COACH | Team coach/mentor | ✅ Yes |
| ADMIN | System administrator | ❌ No |

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system successfully updates user information as an admin and returns HTTP 200 with the updated data.
- **SC-002**: The system rejects admin update attempts for non-existent users with a not found error.
- **SC-003**: The system enforces email uniqueness when admins update user emails.
- **SC-004**: The system allows role changes only between non-admin roles (e.g., CONTESTANT → COACH).
- **SC-005**: The system prevents assigning the ADMIN role through user updates.
- **SC-006**: Non-admin users cannot update other users' information (HTTP 403).
- **SC-007**: Validation and authorization errors include clear messages and a consistent structure.
- **SC-008**: The `createdAt` value remains unchanged after admin updates.
- **SC-009**: The `updatedAt` value is updated on every successful modification.

