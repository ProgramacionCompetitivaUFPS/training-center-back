# Feature Specification: Create User (Self-Registration)

**Created**: 2025-12-13  

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Create user (Priority: P1)

As a system contestant, I want to register by creating an account so that I can access the platform and participate in the available functionalities.

> **Note**: This specification covers the public self-registration flow, which only allows creating users with the CONTESTANT role. Other user roles (ADMIN, COACH) are assigned through administrative processes outside of this flow.

**Why this priority**: User registration is the entry point to the system. Without this feature, there is no way to identify users or enable access to other application capabilities (competitions, problems, groups, etc.).

**Independent Test**: This user story can be tested independently by consuming the `POST /users` endpoint, validating the correct creation of the user and associated validation errors, without depending on other system features.

**Acceptance Scenarios**:

1. **Scenario**: Successful user creation
   - **Given** no user is registered with the provided email
   - **When** valid user registration data is submitted
   - **Then** the system creates the user and returns their data with a unique identifier and creation date
   - **And** the user is automatically added as a member of the global group

2. **Scenario**: Duplicate email
   - **Given** a user already exists with the provided email
   - **When** another user is attempted to be created with the same email
   - **Then** the system rejects the operation with a validation error indicating the email is already in use

3. **Scenario**: Required fields missing
   - **Given** the request omits one or more required fields
   - **When** the user creation request is submitted
   - **Then** the system rejects the operation with a structured validation error indicating the missing fields

4. **Scenario**: Invalid email format
   - **Given** the provided email does not comply with a valid format
   - **When** the user creation request is submitted
   - **Then** the system rejects the operation with a validation error indicating the email format is invalid

---

### Edge Cases

- Attempt to create a user with an empty or null email.
- Attempt to create a user with an empty name or name containing only spaces.
- Use of Unicode characters in the name, nickname, or institution.
- Concurrent requests attempting to create users with the same email.
- Payload with additional fields not recognized by the API.

## API Contract

### POST /users

Register a new user in the system.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Content-Type | string | Yes | application/json |

**Request Body**:
```json
{
  "email": "string",
  "password": "string",
  "name": "string",
  "nickname": "string",
  "country": "string",
  "city": "string",
  "institution": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| email | string | Yes | User's email address (must be unique) |
| password | string | Yes | User's password for authentication (must meet security requirements) |
| name | string | Yes | User's full name |
| nickname | string | No | User's display name or alias |
| country | string | Yes | User's country |
| city | string | Yes | User's city |
| institution | string | Yes | User's institution or organization |

**Password Requirements**:
- Minimum 8 characters
- At least 1 uppercase letter (A-Z)
- At least 1 special character (!@#$%^&*()_+-=[]{}|;:',.<>?/)
- At least 1 number (0-9)

**Responses**:

#### 201 Created
User created successfully.

```json
{
  "email": "user@example.com",
  "name": "John Doe",
  "nickname": "johnd",
  "country": "United States",
  "city": "Cambridge",
  "institution": "MIT",
  "role": "CONTESTANT",
  "createdAt": "2025-12-13T10:30:00Z"
}
```

> **Note**: The `id` is generated internally but not returned in the response. The `nickname` is stored in lowercase regardless of input case.

#### 400 Bad Request
Validation error in the request (missing fields, invalid format).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "email",
      "message": "Invalid email format"
    },
    {
      "field": "name",
      "message": "Name is required"
    }
  ]
}
```

#### 409 Conflict
The email is already registered.

```json
{
  "error": "EMAIL_ALREADY_EXISTS",
  "message": "The email address is already in use"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST allow users to register via a public registration endpoint.
- **FR-002**: The system MUST ensure that each user email is unique across the whole system.
- **FR-003**: The system MUST validate the email format before persisting the user.
- **FR-004**: The system MUST validate the presence of required fields (email, password, name, country, city, institution).
- **FR-004.1**: The system MUST securely hash the password before storing it.
- **FR-004.2**: The system MUST enforce password security rules: minimum 8 characters, at least 1 uppercase letter, at least 1 special character, at least 1 number.
- **FR-005**: The system MUST automatically generate a unique identifier (id) for each user (not returned in responses).
- **FR-005.1**: The system MUST store nicknames in lowercase, regardless of the input case provided.
- **FR-006**: The system MUST assign the CONTESTANT role to the new user. Other roles (ADMIN, COACH) are assigned through separate administrative processes.
- **FR-006.1**: The system MUST automatically add the new user as a member of the global group upon creation.
- **FR-007**: The system MUST persist the user's creation date.
- **FR-008**: The system MUST return validation errors with a consistent structure and clear messages.

### Key Entities

- **User**: Represents a registered person in the system.  
  Key attributes:
  - `id` (string, UUID)
  - `email` (string, unique)
  - `password` (string, hashed, never returned in responses)
  - `name` (string)
  - `nickname` (string, optional, stored in lowercase)
  - `country` (string, required)
  - `city` (string, required)
  - `institution` (string, required)
  - `role` (enum: ADMIN | COACH | CONTESTANT, default: CONTESTANT)
  - `status` (enum: ACTIVE | DEACTIVATED, default: ACTIVE, immutable via user actions except self-deactivation)
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp, nullable, null on creation)

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: The system allows the creation of a valid user, returning HTTP 201 with user data (id not included in response).
- **SC-002**: The system rejects creation attempts with duplicate emails, returning a validation error.
- **SC-003**: 100% of created users have a unique email.
- **SC-004**: Validation errors include clear messages and a consistent structure.
- **SC-005**: User creation can be successfully completed in a single call to the API with no external dependencies.