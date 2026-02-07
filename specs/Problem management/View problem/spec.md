# Feature Specification: View Problem

**Created**: 2026-02-06

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View single problem details (Priority: P1)

As an authenticated user, I want to view detailed information about a specific problem so that I can understand the problem requirements, constraints, and metadata before attempting to solve it or add it to a contest.

**Why this priority**: Viewing problem details is essential for users to understand what they need to solve. Without this feature, users cannot see problem statements, time/memory limits, or any other critical information needed to work with problems.

**Independent Test**: This user story can be tested independently by consuming the `GET /problems/{slug}` endpoint with different user types (Admin, modifier, regular user) and problem states (DRAFT, PUBLISHED), validating correct responses and access control without depending on other system features.

**Acceptance Scenarios**:

1. **Scenario**: View published problem as any authenticated user
   - **Given** a problem with status PUBLISHED exists
   - **And** a user is authenticated (any role)
   - **When** the user requests GET /problems/{slug}
   - **Then** the system returns HTTP 200 with complete problem metadata
   - **And** the response includes title, statement, limits, tags, status, accessibility, and author information
   - **And** the response does NOT include modifiers list or file indicators (user is not a modifier)

2. **Scenario**: View draft problem as modifier
   - **Given** a problem with status DRAFT exists
   - **And** a user is authenticated and is a modifier (author or assigned modifier) for that problem
   - **When** the user requests GET /problems/{slug}
   - **Then** the system returns HTTP 200 with complete problem metadata
   - **And** the response includes modifiers list and file availability indicators

3. **Scenario**: View draft problem as Admin
   - **Given** a problem with status DRAFT exists
   - **And** a user is authenticated with ADMIN role
   - **When** the user requests GET /problems/{slug}
   - **Then** the system returns HTTP 200 with complete problem metadata
   - **And** the response includes modifiers list and file availability indicators

4. **Scenario**: Non-modifier attempts to view draft problem
   - **Given** a problem with status DRAFT exists
   - **And** a user is authenticated but is NOT a modifier or Admin
   - **When** the user requests GET /problems/{slug}
   - **Then** the system rejects with HTTP 403 Forbidden
   - **And** returns error code INSUFFICIENT_PERMISSIONS

5. **Scenario**: Unauthenticated request
   - **Given** a problem exists (any status)
   - **When** a request is made without valid authentication credentials
   - **Then** the system rejects with HTTP 401 Unauthorized

6. **Scenario**: Problem not found
   - **Given** no problem exists with the provided slug
   - **When** a user requests GET /problems/{slug}
   - **Then** the system returns HTTP 404 Not Found

---

### User Story 2 – List problems with filters (Priority: P1)

As an authenticated user, I want to list problems with optional filters and pagination so that I can discover problems relevant to my needs without overwhelming the system or UI.

**Why this priority**: Users need to browse and search for problems efficiently. Pagination and filters are essential for usability when dealing with hundreds or thousands of problems.

**Independent Test**: This user story can be tested independently by consuming the `GET /problems` endpoint with various filter combinations and pagination parameters, validating correct filtering, visibility rules, and pagination metadata.

**Acceptance Scenarios**:

1. **Scenario**: List problems without filters
   - **Given** multiple problems exist (both DRAFT and PUBLISHED)
   - **And** a user is authenticated
   - **When** the user requests GET /problems without filters
   - **Then** the system returns HTTP 200 with PUBLISHED problems only
   - **And** results are paginated with default page=1 and limit=20
   - **And** response includes pagination metadata (totalCount, currentPage, totalPages, itemsPerPage)

2. **Scenario**: Modifier sees own draft problems in list
   - **Given** multiple problems exist including DRAFT problems where user is a modifier
   - **And** a user is authenticated and is a modifier for some DRAFT problems
   - **When** the user requests GET /problems
   - **Then** the system returns PUBLISHED problems AND DRAFT problems where user is a modifier
   - **And** results respect pagination

3. **Scenario**: Filter by status
   - **Given** multiple problems exist with different statuses
   - **And** a user is authenticated
   - **When** the user requests GET /problems?status=PUBLISHED
   - **Then** the system returns only PUBLISHED problems
   - **And** visibility rules still apply (no DRAFT problems even if user is modifier)

4. **Scenario**: Filter by accessibility
   - **Given** multiple problems exist with different accessibility levels
   - **And** a user is authenticated
   - **When** the user requests GET /problems?accessibility=PUBLIC
   - **Then** the system returns only problems with accessibility PUBLIC

5. **Scenario**: Filter by tags (AND logic)
   - **Given** multiple problems exist with different tags
   - **And** a user is authenticated
   - **When** the user requests GET /problems?tags=algorithms,dp
   - **Then** the system returns only problems containing BOTH "algorithms" AND "dp" tags

6. **Scenario**: Filter by author
   - **Given** multiple problems exist from different authors
   - **And** a user is authenticated
   - **When** the user requests GET /problems?author=coach_john
   - **Then** the system returns only problems created by author with nickname "coach_john"

7. **Scenario**: Multiple filters combined
   - **Given** multiple problems exist
   - **And** a user is authenticated
   - **When** the user requests GET /problems?status=PUBLISHED&accessibility=PUBLIC&tags=beginner
   - **Then** the system returns only problems matching ALL criteria (AND logic)

8. **Scenario**: Custom pagination
   - **Given** 50 problems exist
   - **And** a user is authenticated
   - **When** the user requests GET /problems?page=2&limit=10
   - **Then** the system returns problems 11-20
   - **And** pagination metadata shows currentPage=2, totalPages=5, itemsPerPage=10

9. **Scenario**: Invalid pagination parameters
   - **Given** a user is authenticated
   - **When** the user requests GET /problems?page=0 or page=-1
   - **Then** the system returns HTTP 400 Bad Request with validation error

10. **Scenario**: Limit exceeds maximum
    - **Given** a user is authenticated
    - **When** the user requests GET /problems?limit=150
    - **Then** the system returns HTTP 400 Bad Request with validation error indicating maximum limit is 100

11. **Scenario**: Empty page
    - **Given** only 10 problems exist
    - **And** a user is authenticated
    - **When** the user requests GET /problems?page=5&limit=10
    - **Then** the system returns HTTP 200 with empty problems array
    - **And** pagination metadata shows totalCount=10, currentPage=5, totalPages=1

---

### Edge Cases

- Request with invalid slug format (too short, too long, invalid characters)
- Request with expired authentication token
- Request with malformed authentication token
- Problem with null statement, timeLimit, or memoryLimit fields
- Problem with empty tags or languageOverrides arrays
- Problem with null problemJudgingUpdatedAt
- Concurrent requests for same problem
- Very large pagination requests (limit=100, page=1000)
- Filter with non-existent author nickname
- Filter with non-existent tags
- Unicode characters in problem title and statement

## API Contract

### GET /problems/{slug}

View detailed information about a specific problem.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | Unique problem identifier (3-70 chars, lowercase alphanumeric with hyphens) |

**Responses**:

#### 200 OK
Problem details retrieved successfully.

```json
{
  "slug": "sum-two-numbers",
  "title": "Sum of Two Numbers",
  "statement": "Given two integers a and b, return their sum.",
  "timeLimit": 2000,
  "memoryLimit": 256,
  "languageOverrides": [
    { "language": "python310", "timeLimit": 4000 },
    { "language": "java17", "memoryLimit": 512 }
  ],
  "tags": ["math", "beginner"],
  "status": "PUBLISHED",
  "accessibility": "PUBLIC",
  "author": {
    "nickname": "coach_john",
    "name": "John Smith"
  },
  "modifiers": [
    {
      "nickname": "coach_john",
      "name": "John Smith"
    },
    {
      "nickname": "coach_mary",
      "name": "Mary Johnson"
    }
  ],
  "files": {
    "testCases": true,
    "solutions": ["solution.cpp", "solution.py"],
    "checker": false,
    "validator": true
  },
  "createdAt": "2025-12-20T10:00:00Z",
  "updatedAt": "2025-12-21T15:30:00Z",
  "problemJudgingUpdatedAt": "2025-12-21T15:30:00Z"
}
```

> **Note**: The `modifiers` and `files` fields are only included if the requesting user is a modifier or Admin. For regular users viewing PUBLISHED problems, these fields are excluded.

**Field Descriptions**:

| Field | Type | Description |
|-------|------|-------------|
| slug | string | Unique problem identifier |
| title | string | Problem title (normalized NFKC) |
| statement | string \| null | Problem statement in LaTeX format |
| timeLimit | integer \| null | Default time limit in milliseconds |
| memoryLimit | integer \| null | Default memory limit in MiB |
| languageOverrides | array | Language-specific limit overrides |
| tags | string[] | Problem tags (empty array if none) |
| status | enum | DRAFT or PUBLISHED |
| accessibility | enum | PRIVATE or PUBLIC |
| author | object | Author information (nickname, name) |
| modifiers | array | List of modifiers (only for modifiers/Admin) |
| files | object | File availability indicators (only for modifiers/Admin) |
| createdAt | string | Creation timestamp (ISO 8601) |
| updatedAt | string | Last update timestamp (ISO 8601) |
| problemJudgingUpdatedAt | string \| null | Last judging component update (ISO 8601 or null) |

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission to view this DRAFT problem.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only the problem author, Admin, or assigned modifiers can view this DRAFT problem"
}
```

#### 404 Not Found
Problem with the specified slug does not exist.

```json
{
  "error": "NOT_FOUND",
  "message": "Problem not found"
}
```

---

### GET /problems

List problems with optional filters and pagination.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| status | string | No | Filter by status (DRAFT or PUBLISHED) |
| accessibility | string | No | Filter by accessibility (PRIVATE or PUBLIC) |
| tags | string | No | Comma-separated tags (AND logic, e.g., "algorithms,dp") |
| author | string | No | Filter by author nickname |
| page | integer | No | Page number (default: 1, minimum: 1) |
| limit | integer | No | Items per page (default: 20, minimum: 1, maximum: 100) |

**Responses**:

#### 200 OK
Problems list retrieved successfully.

```json
{
  "problems": [
    {
      "slug": "sum-two-numbers",
      "title": "Sum of Two Numbers",
      "tags": ["math", "beginner"],
      "status": "PUBLISHED",
      "accessibility": "PUBLIC",
      "author": {
        "nickname": "coach_john",
        "name": "John Smith"
      },
      "createdAt": "2025-12-20T10:00:00Z",
      "updatedAt": "2025-12-21T15:30:00Z"
    },
    {
      "slug": "binary-search",
      "title": "Binary Search",
      "tags": ["algorithms", "search"],
      "status": "PUBLISHED",
      "accessibility": "PRIVATE",
      "author": {
        "nickname": "coach_mary",
        "name": "Mary Johnson"
      },
      "createdAt": "2025-12-19T14:00:00Z",
      "updatedAt": "2025-12-19T14:00:00Z"
    }
  ],
  "pagination": {
    "totalCount": 42,
    "currentPage": 1,
    "totalPages": 3,
    "itemsPerPage": 20
  }
}
```

> **Note**: The list response includes a subset of fields compared to the single problem view. Statement, limits, modifiers, and files are NOT included in list responses.

**Field Descriptions**:

| Field | Type | Description |
|-------|------|-------------|
| problems | array | Array of problem summaries |
| pagination | object | Pagination metadata |
| pagination.totalCount | integer | Total number of problems matching filters |
| pagination.currentPage | integer | Current page number |
| pagination.totalPages | integer | Total number of pages |
| pagination.itemsPerPage | integer | Number of items per page |

#### 400 Bad Request
Invalid query parameters.

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request parameters",
  "details": [
    {
      "field": "page",
      "message": "Page must be a positive integer"
    },
    {
      "field": "limit",
      "message": "Limit must be between 1 and 100"
    }
  ]
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

---

## Requirements *(mandatory)*

### Functional Requirements

**Single Problem View**
- **FR-001**: The system MUST allow authenticated users to view PUBLISHED problems via GET /problems/{slug}
- **FR-002**: The system MUST allow modifiers (author or assigned modifiers) and Admin to view DRAFT problems via GET /problems/{slug}
- **FR-003**: The system MUST reject requests from non-modifiers attempting to view DRAFT problems with HTTP 403 Forbidden
- **FR-004**: The system MUST reject unauthenticated requests with HTTP 401 Unauthorized
- **FR-005**: The system MUST return HTTP 404 Not Found for non-existent problem slugs
- **FR-006**: The system MUST include author information (nickname and name) in all problem responses
- **FR-007**: The system MUST include modifiers list in responses when the requesting user is a modifier or Admin
- **FR-008**: The system MUST exclude modifiers list from responses when the requesting user is NOT a modifier or Admin
- **FR-009**: The system MUST include file availability indicators in responses when the requesting user is a modifier or Admin
- **FR-010**: The system MUST exclude file availability indicators from responses when the requesting user is NOT a modifier or Admin
- **FR-011**: The system MUST include timestamps (createdAt, updatedAt, problemJudgingUpdatedAt) in all problem responses
- **FR-012**: The system MUST NOT return internal database IDs in any response

**Problem List**
- **FR-013**: The system MUST return PUBLISHED problems by default when listing problems
- **FR-014**: The system MUST include DRAFT problems in list results only for users who are modifiers or Admin for those problems
- **FR-015**: The system MUST support filtering by status (DRAFT or PUBLISHED)
- **FR-016**: The system MUST support filtering by accessibility (PRIVATE or PUBLIC)
- **FR-017**: The system MUST support filtering by tags using AND logic (all specified tags must be present)
- **FR-018**: The system MUST support filtering by author nickname
- **FR-019**: The system MUST apply all filters using AND logic when multiple filters are provided
- **FR-020**: The system MUST enforce visibility rules on filtered results (DRAFT problems only visible to modifiers/Admin)
- **FR-021**: The system MUST support pagination with page and limit query parameters
- **FR-022**: The system MUST return pagination metadata (totalCount, currentPage, totalPages, itemsPerPage)
- **FR-023**: The system MUST default to page=1 and limit=20 if pagination parameters are not provided
- **FR-024**: The system MUST validate that page is a positive integer (minimum 1)
- **FR-025**: The system MUST validate that limit is between 1 and 100 (inclusive)
- **FR-026**: The system MUST enforce a maximum limit of 100 items per page
- **FR-027**: The system MUST sort results by createdAt in descending order (newest first) by default
- **FR-028**: The system MUST return an empty problems array with correct pagination metadata when requested page exceeds available pages

**Data Completeness**
- **FR-029**: The system MUST return slug, title, statement, timeLimit, memoryLimit, languageOverrides, tags, status, and accessibility for all problems
- **FR-030**: The system MUST return timestamps in ISO 8601 format
- **FR-031**: The system MUST return null for problemJudgingUpdatedAt when it has never been set
- **FR-032**: The system MUST normalize all text fields using Unicode NFKC normalization
- **FR-033**: The system MUST return empty arrays for tags and languageOverrides when no values exist
- **FR-034**: The system MUST return consistent field names and data types across all endpoints

**Error Handling**
- **FR-035**: The system MUST return HTTP 404 Not Found with error code NOT_FOUND when problem doesn't exist
- **FR-036**: The system MUST return HTTP 401 Unauthorized with error code UNAUTHORIZED when authentication fails
- **FR-037**: The system MUST return HTTP 403 Forbidden with error code INSUFFICIENT_PERMISSIONS when authorization fails
- **FR-038**: The system MUST return HTTP 400 Bad Request with error code VALIDATION_ERROR and field-level details when validation fails
- **FR-039**: The system MUST return error responses in consistent JSON format with error code and message fields
- **FR-040**: The system MUST validate authentication before checking problem visibility rules

### Key Entities

- **Problem**: Represents a programming problem.  
  Key attributes:
  - `id` (string, UUID, internal only, never exposed in API responses)
  - `slug` (string, unique, user-provided, 3-70 chars, lowercase alphanumeric with hyphens)
  - `title` (string, normalized NFKC)
  - `statement` (string, LaTeX format, nullable)
  - `timeLimit` (integer, milliseconds, nullable)
  - `memoryLimit` (integer, MiB, nullable)
  - `languageOverrides` (array, language-specific limit overrides)
  - `tags` (array of strings, from predefined list)
  - `status` (enum: DRAFT | PUBLISHED)
  - `accessibility` (enum: PRIVATE | PUBLIC)
  - `authorId` (string, UUID, FK to User) - JOIN with User table required to retrieve author.nickname and author.name for API response
  - `modifierIds` (array of UUIDs, FK to User) - JOIN with User table required to retrieve modifiers[].nickname and modifiers[].name for API response
  - `testCasesFileKey` (string, nullable)
  - `solutionFileKeys` (array of strings)
  - `checkerFileKey` (string, nullable)
  - `validatorFileKey` (string, nullable)
  - `problemJudgingUpdatedAt` (timestamp, nullable)
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp)

- **User**: Represents a user (author or modifier).  
  Key attributes:
  - `id` (string, UUID, internal only, never exposed in API responses)
  - `nickname` (string, unique, lowercase)
  - `name` (string)
  - `role` (enum: ADMIN | COACH | CONTESTANT)

> **Note on Data Retrieval**: To return author and modifiers information in API responses, the system must JOIN the Problem table with the User table using `authorId` and each UUID in `modifierIds` array. The authenticated user's `userId` (from JWT token) is used for authorization checks without requiring additional JOINs.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Authenticated users can view PUBLISHED problems via GET /problems/{slug} with HTTP 200
- **SC-002**: Modifiers and Admin can view DRAFT problems via GET /problems/{slug} with HTTP 200
- **SC-003**: Non-modifiers attempting to view DRAFT problems receive HTTP 403 Forbidden
- **SC-004**: Unauthenticated requests receive HTTP 401 Unauthorized
- **SC-005**: Non-existent problem slugs return HTTP 404 Not Found
- **SC-006**: Problem responses include author information (nickname and name)
- **SC-007**: Modifiers and Admin see modifiers list and file indicators in responses
- **SC-008**: Regular users do NOT see modifiers list or file indicators in responses
- **SC-009**: All timestamps are returned in ISO 8601 format
- **SC-010**: No internal database IDs are exposed in any response
- **SC-011**: Authenticated users can list problems via GET /problems with HTTP 200
- **SC-012**: List results include only PUBLISHED problems by default (plus DRAFT where user is modifier)
- **SC-013**: Filters work correctly (status, accessibility, tags, author) with AND logic
- **SC-014**: Pagination works correctly with default and custom page/limit values
- **SC-015**: Pagination metadata accurately reflects total results and current page
- **SC-016**: Invalid pagination parameters return HTTP 400 Bad Request with validation errors
- **SC-017**: Empty pages return empty problems array with correct pagination metadata
- **SC-018**: List responses are sorted by createdAt descending (newest first)
- **SC-019**: Visibility rules are enforced consistently across both endpoints
- **SC-020**: Error responses follow consistent JSON format with error code and message
