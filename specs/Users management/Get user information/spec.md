# Feature Specification: Get User Profile

**Created**: 2025-12-20

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View my own profile (Priority: P1)

As an authenticated user, I want to view my own profile information so that I can see my current account details and verify my information is correct.

**Why this priority**: Users need to access their own profile information to verify account details, check their role, and understand their account status. This is fundamental for user self-service and account management.

**Independent Test**: This user story can be tested independently by consuming the `GET /users/me` endpoint with valid authentication, validating that the authenticated user's complete profile information is returned.

**Acceptance Scenarios**:

1. **Scenario**: Successful retrieval of own profile
   - **Given** a user exists in the system with ACTIVE status and is authenticated
   - **When** they request their own profile information
   - **Then** the system returns complete user information including email, name, nickname, institution, role, createdAt, updatedAt (if applicable)
   - **And** password and id are never included in the response

2. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a profile request is submitted
   - **Then** the system rejects the operation with 401 Unauthorized

3. **Scenario**: User not found
   - **Given** the token is valid but does not resolve an existing user
   - **When** a profile request is submitted
   - **Then** the system rejects with 404 Not Found

---

### User Story 2 – View other users' profiles (Priority: P2)

As an authenticated user (Coach, Contestant, or Admin), I want to view other users' profile information by their nickname so that I can see public user details for collaboration, verification, or administrative purposes, while respecting privacy restrictions.

**Why this priority**: Enables users to discover and verify other users on the platform, which is essential for collaboration features (groups, contests, teams). However, this is lower priority than viewing one's own profile as it's not essential for basic account management.

**Independent Test**: This user story can be tested independently by consuming the `GET /users/{nickname}` endpoint with valid authentication, validating that appropriate user information is returned based on the requester's role and the target user's status, with proper access restrictions enforced.

**Acceptance Scenarios**:

1. **Scenario**: Coach views another Coach/Contestant profile
   - **Given** a Coach is authenticated
   - **And** a target user exists with role COACH or CONTESTANT, ACTIVE status, and a unique nickname
   - **When** the Coach requests the target user's profile by nickname
   - **Then** the system returns public user information (name, nickname, institution, role, createdAt)
   - **And** sensitive information (email, password, id) is not included
   - **And** the response is 200 OK

2. **Scenario**: Contestant views another Contestant/Coach profile
   - **Given** a Contestant is authenticated
   - **And** a target user exists with role CONTESTANT or COACH, ACTIVE status, and a unique nickname
   - **When** the Contestant requests the target user's profile by nickname
   - **Then** the system returns public user information (name, nickname, institution, role, createdAt)
   - **And** sensitive information (email, password, id) is not included
   - **And** the response is 200 OK

3. **Scenario**: Non-admin attempts to view Admin profile
   - **Given** a Coach or Contestant is authenticated
   - **And** a target user exists with role ADMIN and a unique nickname
   - **When** they attempt to view the Admin's profile by nickname
   - **Then** the system rejects with 403 Forbidden (ADMIN_PROFILE_RESTRICTED)
   - **And** no user information is returned

4. **Scenario**: Admin views any active user profile
   - **Given** an Admin is authenticated
   - **And** a target user exists with any role, ACTIVE status, and a unique nickname
   - **When** the Admin requests the target user's profile by nickname
   - **Then** the system returns complete user information including email and all fields (except password and id)
   - **And** the response is 200 OK

5. **Scenario**: Non-admin attempts to view deactivated user profile
   - **Given** a Coach or Contestant is authenticated
   - **And** a target user has status DEACTIVATED (nickname anonymized to `user_anonimo_{uuid}`)
   - **When** they attempt to request the user's profile by the anonymized nickname
   - **Then** the system rejects with 404 Not Found
   - **And** no user information is returned

6. **Scenario**: Admin attempts to view deactivated user profile
   - **Given** an Admin is authenticated
   - **And** a target user has status DEACTIVATED
   - **When** the Admin requests the user's profile by the anonymized nickname
   - **Then** the system rejects with 404 Not Found
   - **And** deactivated user profiles are not accessible even for admins via this endpoint

7. **Scenario**: Search by original nickname of deactivated user
   - **Given** a user was deactivated and their nickname was anonymized
   - **When** anyone searches for the original (pre-deactivation) nickname
   - **Then** the system returns 404 Not Found (original nickname no longer exists)

8. **Scenario**: Target user not found
   - **Given** an authenticated user
   - **And** no user exists with the provided nickname
   - **When** they request the user profile
   - **Then** the system rejects with 404 Not Found

9. **Scenario**: Invalid nickname format
   - **Given** an authenticated user
   - **When** they request a profile with an invalid or empty nickname
   - **Then** the system rejects with 400 Bad Request (INVALID_NICKNAME_FORMAT)

10. **Scenario**: Unauthenticated request to view other user
    - **Given** the request does not include valid authentication credentials
    - **When** a profile request for another user is submitted
    - **Then** the system rejects with 401 Unauthorized

11. **Scenario**: User requests own profile via nickname endpoint
    - **Given** an authenticated user with nickname "johnd"
    - **When** they request their own profile via `GET /users/johnd`
    - **Then** the system returns the same complete information as `GET /users/me` (including email)
    - **And** password and id are never included in the response

12. **Scenario**: Case-insensitive nickname lookup
    - **Given** a user exists with nickname "johnd" (stored in lowercase)
    - **And** an authenticated user requests the profile
    - **When** they request the profile using "JohnD", "JOHND", or "johnd"
    - **Then** the system returns the same user profile in all cases
    - **And** nickname lookups are case-insensitive

---

### Edge Cases

- Attempt to view profile with empty or whitespace-only nickname.
- Concurrent requests to view the same user profile by nickname.
- User changes nickname between request and response (old nickname lookup should fail).
- Requesting profile of a user that was just created.
- Admin viewing their own profile through `GET /users/{nickname}` vs `GET /users/me`.
- Unicode characters in user names, nicknames, or institutions.
- Very long names or nicknames in profile responses.
- Nickname stored in lowercase: lookups are case-insensitive.
- Deactivated user profiles are not accessible by anyone (404 Not Found).
- Original nickname of deactivated user is freed and can be used by new accounts.

## API Contract

### GET /users/me

Retrieve the authenticated user's own profile information.

> **Important**: User identity is resolved exclusively from the authentication token. This endpoint returns complete profile information including sensitive fields (email) as the user is viewing their own data. Password and id are never returned.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Responses**:

#### 200 OK
User profile retrieved successfully.

```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "nickname": "johnd",
  "institution": "MIT",
  "role": "CONTESTANT",
  "createdAt": "2025-12-13T10:00:00Z",
  "updatedAt": "2025-12-14T09:30:00Z"
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

### GET /users/{nickname}

Retrieve another user's profile information by their unique nickname.

> **Important**: Access restrictions apply based on the requester's role and the target user's role/status:
> - Admins can view all **active** user profiles
> - Non-admins cannot view Admin profiles
> - **Deactivated users are not accessible by anyone** (returns 404)
> - Nickname lookups are case-insensitive (all nicknames stored in lowercase)
> - Password and id are never returned

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| nickname | string | Yes | The unique nickname of the user to retrieve (case-insensitive) |

**Responses**:

#### 200 OK
User profile retrieved successfully. Response structure varies based on requester role.

**For non-admin viewing active non-admin user**:
```json
{
  "name": "Jane Smith",
  "nickname": "janes",
  "institution": "Stanford",
  "role": "COACH",
  "createdAt": "2025-12-10T08:00:00Z"
}
```

**For Admin viewing any active user**:
```json
{
  "email": "user@example.com",
  "name": "Jane Smith",
  "nickname": "janes",
  "institution": "Stanford",
  "role": "COACH",
  "createdAt": "2025-12-10T08:00:00Z",
  "updatedAt": "2025-12-14T09:30:00Z"
}
```

**For user viewing their own profile via nickname**:
```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "nickname": "johnd",
  "institution": "MIT",
  "role": "CONTESTANT",
  "createdAt": "2025-12-13T10:00:00Z",
  "updatedAt": "2025-12-14T09:30:00Z"
}
```

#### 400 Bad Request
Invalid nickname format.

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid nickname format",
  "details": [
    {
      "field": "nickname",
      "message": "Nickname cannot be empty"
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
Non-admin user attempting to view Admin profile.

```json
{
  "error": "ADMIN_PROFILE_RESTRICTED",
  "message": "Admin profiles are not accessible to non-admin users"
}
```

#### 404 Not Found
Target user not found (including deactivated users).

```json
{
  "error": "NOT_FOUND",
  "message": "User not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Own Profile (GET /users/me)**
- **FR-001**: The system MUST allow authenticated users with ACTIVE status to retrieve their own profile information via `GET /users/me`.
- **FR-002**: User identity for `GET /users/me` MUST be resolved exclusively from the authentication token.
- **FR-003**: The system MUST return complete profile information (including email) when a user views their own profile.
- **FR-004**: The system MUST never return `password` or `id` in any profile response.

**Other Users' Profiles (GET /users/{nickname})**
- **FR-005**: The system MUST allow authenticated users with ACTIVE status to retrieve other ACTIVE users' profile information via `GET /users/{nickname}`.
- **FR-006**: The system MUST validate that the nickname in the path parameter is not empty and follows valid format.
- **FR-007**: The system MUST perform nickname lookups in a case-insensitive manner (nicknames are stored in lowercase).
- **FR-008**: The system MUST restrict non-admin users (Coach, Contestant) from viewing Admin profiles, returning 403 Forbidden.
- **FR-009**: The system MUST allow non-admin users to view profiles of other active non-admin users (Coach, Contestant).
- **FR-010**: The system MUST allow Admin users to view profiles of all active users regardless of role.

**Deactivated Users**
- **FR-011**: The system MUST NOT allow access to deactivated user profiles via `GET /users/{nickname}`, returning 404 Not Found.
- **FR-012**: Deactivated users are not accessible by anyone, including Admins, through this endpoint.
- **FR-013**: Original nicknames of deactivated users become available for registration by new accounts.

**Response Rules**
- **FR-014**: The system MUST NOT include `email` in profile responses for non-admin users viewing other users' profiles.
- **FR-015**: When a user requests their own profile via `GET /users/{nickname}`, the system MUST return complete information (same as `GET /users/me`).
- **FR-016**: The system MUST return consistent response structures based on requester role and ownership.
- **FR-017**: The system MUST return appropriate HTTP status codes (200, 400, 401, 403, 404) with clear error messages.

### Key Entities

- **User**: Registered person in the system.  
  Relevant attributes for this feature:
  - `email` (string, sensitive - only visible to self or Admin for active users)
  - `password` (string, hashed, **never returned in any response**)
  - `name` (string, public for active users)
  - `institution` (string, optional, public for active users)
  - `nickname` (string, UNIQUE, lowercase, public for active users)
  - `role` (enum: ADMIN | COACH | CONTESTANT)
  - `status` (enum: ACTIVE | DEACTIVATED)
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp, nullable)

> **Profile Visibility Rules**:
> - **Own profile**: Active users can see complete information including email. Password and id never returned.
> - **Other users (non-admin requester)**: Can see public fields (name, nickname, institution, role, createdAt) for active non-admin users. Cannot see Admin profiles. Deactivated users return 404.
> - **Other users (Admin requester)**: Can see complete information (including email) for all active users. Deactivated users return 404.
> - **Deactivated users**: Not accessible by anyone. Returns 404 Not Found.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Authenticated users with ACTIVE status can successfully retrieve their own profile via `GET /users/me` with HTTP 200.
- **SC-002**: Authenticated users with ACTIVE status can retrieve other active non-admin users' profiles via `GET /users/{nickname}` with HTTP 200.
- **SC-003**: Non-admin users attempting to view Admin profiles receive HTTP 403.
- **SC-004**: Admin users can view profiles of all active users with complete information (except password and id).
- **SC-005**: Deactivated user profiles return HTTP 404 for all requesters (including Admins).
- **SC-006**: Email addresses are only included when viewing own profile or when Admin views active users.
- **SC-007**: Password and id are never included in any profile response.
- **SC-008**: Invalid or empty nicknames return HTTP 400 with validation error.
- **SC-009**: Non-existent nicknames return HTTP 404.
- **SC-010**: Unauthenticated requests return HTTP 401.
- **SC-011**: Nickname lookups are case-insensitive.
- **SC-012**: When a user requests their own profile via `GET /users/{nickname}`, they receive complete information (same as `GET /users/me`).

