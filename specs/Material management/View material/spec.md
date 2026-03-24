# Feature Specification: View Materials

**Created**: 2026-02-06

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View materials in a group (Priority: P1)

As a group member, I want to view a list of materials in my group with filters and sorting so that I can find relevant announcements, resources, and information shared by group leads.

**Why this priority**: Viewing materials is essential for group members to access shared content, announcements, and resources. Without this feature, users cannot see the information that leads share with the group.

**Independent Test**: This user story can be tested independently by consuming the `GET /groups/{groupId}/materials` endpoint with different user types (Lead, Member, non-member) and filter combinations, validating correct visibility rules, sorting, and pagination.

**Acceptance Scenarios**:

1. **Scenario**: Group member views published materials
   - **Given** a group exists with multiple PUBLISHED materials
   - **And** a user is authenticated and is a member of the group
   - **When** the user requests GET /groups/{groupId}/materials
   - **Then** the system returns HTTP 200 with all PUBLISHED materials
   - **And** pinned materials appear first, sorted by pinnedAt DESC
   - **And** non-pinned materials follow, sorted by publishedAt DESC (or createdAt if never published)
   - **And** results are paginated with default page=1 and limit=20

2. **Scenario**: Author sees own draft materials in list
   - **Given** a group exists with DRAFT and PUBLISHED materials
   - **And** a user is authenticated and is the author of some DRAFT materials
   - **When** the user requests GET /groups/{groupId}/materials
   - **Then** the system returns PUBLISHED materials AND the user's own DRAFT materials
   - **And** DRAFT materials from other authors are NOT included

3. **Scenario**: Admin sees all materials including drafts
   - **Given** a group exists with DRAFT and PUBLISHED materials from multiple authors
   - **And** a user is authenticated with ADMIN role
   - **When** the user requests GET /groups/{groupId}/materials
   - **Then** the system returns ALL materials (DRAFT and PUBLISHED) from all authors

4. **Scenario**: Filter by pinned status
   - **Given** a group exists with pinned and non-pinned materials
   - **And** a user is authenticated and is a member
   - **When** the user requests GET /groups/{groupId}/materials?pinned=true
   - **Then** the system returns only pinned materials
   - **And** results are sorted by pinnedAt DESC

5. **Scenario**: Filter by tags (AND logic)
   - **Given** a group exists with materials having different tags
   - **And** a user is authenticated and is a member
   - **When** the user requests GET /groups/{groupId}/materials?tags=announcement,important
   - **Then** the system returns only materials containing BOTH "announcement" AND "important" tags

6. **Scenario**: Non-member views materials in visible group
   - **Given** a group with visibility=VISIBLE exists with PUBLISHED materials
   - **And** a user is authenticated but is NOT a member of the group
   - **When** the user requests GET /groups/{groupId}/materials
   - **Then** the system returns HTTP 200 with PUBLISHED materials only (read-only access)
   - **And** DRAFT materials are NOT included

7. **Scenario**: Non-member attempts to view materials in non-visible group
   - **Given** a group with visibility=NOT_VISIBLE exists
   - **And** a user is authenticated but is NOT a member of the group
   - **When** the user requests GET /groups/{groupId}/materials
   - **Then** the system rejects with HTTP 403 Forbidden

8. **Scenario**: Anonymous user views materials in visible group
   - **Given** a group with visibility=VISIBLE exists with PUBLISHED materials
   - **When** an unauthenticated request is made to GET /groups/{groupId}/materials
   - **Then** the system returns HTTP 200 with PUBLISHED materials only
   - **And** DRAFT materials are NOT included

9. **Scenario**: Anonymous user attempts to view materials in non-visible group
   - **Given** a group with visibility=NOT_VISIBLE exists
   - **When** an unauthenticated request is made to GET /groups/{groupId}/materials
   - **Then** the system rejects with HTTP 401 Unauthorized

10. **Scenario**: Group not found
   - **Given** no group exists with the provided groupId
   - **When** a user requests GET /groups/{groupId}/materials
   - **Then** the system returns HTTP 404 Not Found

11. **Scenario**: Custom pagination
    - **Given** a group exists with 50 materials
    - **And** a user is authenticated and is a member
    - **When** the user requests GET /groups/{groupId}/materials?page=2&limit=10
    - **Then** the system returns materials 11-20
    - **And** pagination metadata shows currentPage=2, totalPages=5, itemsPerPage=10

12. **Scenario**: Invalid pagination parameters
    - **Given** a user is authenticated and is a member
    - **When** the user requests GET /groups/{groupId}/materials?page=0 or limit=150
    - **Then** the system returns HTTP 400 Bad Request with validation error

---

### Edge Cases

- Request with invalid groupId format (not a valid UUID)
- Request with expired authentication token
- Request with malformed authentication token
- Material with empty content field
- Material with empty tags array
- Material with null publishedAt (never published DRAFT)
- Material with null pinnedAt (not pinned)
- Concurrent requests for same group
- Very large pagination requests (limit=100, page=1000)
- Filter with non-existent tags
- Unicode characters in material title and content
- Group with no materials (empty list)

## API Contract

### GET /groups/{groupId}/materials

List materials in a group with optional filters and pagination.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | No | Bearer token for authentication (optional for VISIBLE groups, required for NOT_VISIBLE groups) |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | Group identifier (UUID) |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| pinned | boolean | No | Filter by pinned status (true or false) |
| tags | string | No | Comma-separated tags (AND logic, e.g., "announcement,important") |
| page | integer | No | Page number (default: 1, minimum: 1) |
| limit | integer | No | Items per page (default: 20, minimum: 1, maximum: 100) |

**Responses**:

#### 200 OK
Materials list retrieved successfully.

```json
{
  "materials": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Welcome to the Group!",
      "content": "# Welcome\n\nThis is our group's introduction...",
      "tags": ["announcement", "welcome"],
      "status": "PUBLISHED",
      "pinned": true,
      "pinnedAt": "2026-02-05T10:00:00Z",
      "author": {
        "nickname": "coach_john",
        "name": "John Smith"
      },
      "createdAt": "2026-02-01T10:00:00Z",
      "updatedAt": "2026-02-05T10:00:00Z",
      "publishedAt": "2026-02-01T12:00:00Z"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "title": "Week 1 Resources",
      "content": "## Resources\n\n- [Lecture Notes](https://example.com/notes.pdf)",
      "tags": ["resources", "week1"],
      "status": "PUBLISHED",
      "pinned": false,
      "pinnedAt": null,
      "author": {
        "nickname": "coach_mary",
        "name": "Mary Johnson"
      },
      "createdAt": "2026-02-03T14:00:00Z",
      "updatedAt": "2026-02-03T14:00:00Z",
      "publishedAt": "2026-02-03T14:30:00Z"
    }
  ],
  "pagination": {
    "totalCount": 15,
    "currentPage": 1,
    "totalPages": 1,
    "itemsPerPage": 20
  }
}
```

> **Note**: The `id` field is included in the response for materials (unlike problems which use slug). Materials are identified by UUID.

**Field Descriptions**:

| Field | Type | Description |
|-------|------|-------------|
| materials | array | Array of material objects |
| materials[].id | string | Material identifier (UUID) |
| materials[].title | string | Material title (normalized NFKC, max 200 chars) |
| materials[].content | string | Markdown content with embedded media (max 50000 chars) |
| materials[].tags | string[] | User-defined tags (empty array if none) |
| materials[].status | enum | DRAFT or PUBLISHED |
| materials[].pinned | boolean | Whether material is pinned |
| materials[].pinnedAt | string \| null | When material was pinned (ISO 8601 or null) |
| materials[].author | object | Author information (nickname, name) |
| materials[].createdAt | string | Creation timestamp (ISO 8601) |
| materials[].updatedAt | string | Last update timestamp (ISO 8601) |
| materials[].publishedAt | string \| null | First publication timestamp (ISO 8601 or null) |
| pagination | object | Pagination metadata |
| pagination.totalCount | integer | Total number of materials matching filters |
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
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission to view materials in this group.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "You do not have permission to view materials in this group"
}
```

#### 404 Not Found
Group with the specified ID does not exist.

```json
{
  "error": "NOT_FOUND",
  "message": "Group not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Material Listing**
- **FR-001**: The system MUST allow authenticated group members to view PUBLISHED materials via GET /groups/{groupId}/materials
- **FR-002**: The system MUST allow authors to view their own DRAFT materials in addition to all PUBLISHED materials
- **FR-003**: The system MUST allow Admin to view ALL materials (DRAFT and PUBLISHED) in any group
- **FR-004**: The system MUST allow non-members to view PUBLISHED materials in VISIBLE groups (read-only access)
- **FR-005**: The system MUST reject non-members attempting to view materials in NOT_VISIBLE groups with HTTP 403 Forbidden
- **FR-006**: The system MUST allow anonymous (unauthenticated) users to view PUBLISHED materials in VISIBLE groups
- **FR-007**: The system MUST reject anonymous users attempting to view materials in NOT_VISIBLE groups with HTTP 401 Unauthorized
- **FR-008**: The system MUST return HTTP 404 Not Found for non-existent group IDs

**Sorting**
- **FR-009**: The system MUST sort pinned materials first, ordered by pinnedAt DESC (most recently pinned first)
- **FR-010**: The system MUST sort non-pinned materials by publishedAt DESC, then by createdAt DESC if publishedAt is null
- **FR-011**: The system MUST maintain sort order consistently across pagination

**Filtering**
- **FR-012**: The system MUST support filtering by pinned status (true or false)
- **FR-013**: The system MUST support filtering by tags using AND logic (all specified tags must be present)
- **FR-014**: The system MUST apply all filters using AND logic when multiple filters are provided
- **FR-015**: The system MUST enforce visibility rules on all results (DRAFT materials only visible to author/Admin)

**Pagination**
- **FR-016**: The system MUST support pagination with page and limit query parameters
- **FR-017**: The system MUST return pagination metadata (totalCount, currentPage, totalPages, itemsPerPage)
- **FR-018**: The system MUST default to page=1 and limit=20 if pagination parameters are not provided
- **FR-019**: The system MUST validate that page is a positive integer (minimum 1)
- **FR-020**: The system MUST validate that limit is between 1 and 100 (inclusive)
- **FR-021**: The system MUST enforce a maximum limit of 100 items per page
- **FR-022**: The system MUST return an empty materials array with correct pagination metadata when requested page exceeds available pages

**Data Completeness**
- **FR-023**: The system MUST return id, title, content, tags, status, pinned, pinnedAt, author, createdAt, updatedAt, and publishedAt for all materials
- **FR-024**: The system MUST return author information with nickname and name fields
- **FR-025**: The system MUST return timestamps in ISO 8601 format
- **FR-026**: The system MUST return null for pinnedAt when material is not pinned
- **FR-027**: The system MUST return null for publishedAt when material has never been published
- **FR-028**: The system MUST normalize all text fields using Unicode NFKC normalization
- **FR-029**: The system MUST return empty arrays for tags when no values exist
- **FR-030**: The system MUST return consistent field names and data types

**Error Handling**
- **FR-031**: The system MUST return HTTP 404 Not Found with error code NOT_FOUND when group doesn't exist
- **FR-032**: The system MUST return HTTP 401 Unauthorized with error code UNAUTHORIZED when anonymous users attempt to access NOT_VISIBLE groups
- **FR-033**: The system MUST return HTTP 403 Forbidden with error code INSUFFICIENT_PERMISSIONS when authenticated non-members attempt to access NOT_VISIBLE groups
- **FR-034**: The system MUST return HTTP 400 Bad Request with error code VALIDATION_ERROR and field-level details when validation fails
- **FR-035**: The system MUST return error responses in consistent JSON format with error code and message fields
- **FR-036**: The system MUST validate group visibility before checking group membership

### Key Entities

- **Material**: Represents a post/announcement in a group.  
  Key attributes:
  - `id` (string, UUID, exposed in API responses)
  - `title` (string, max 200 chars, normalized NFKC)
  - `content` (string, Markdown format, max 50000 chars)
  - `tags` (array of strings, user-defined, lowercase + numbers + hyphens + underscores)
  - `status` (enum: DRAFT | PUBLISHED)
  - `pinned` (boolean, default: false)
  - `pinnedAt` (timestamp, nullable)
  - `group_id` (string, UUID, FK to Group) - Used to filter materials by group
  - `author_id` (string, UUID, FK to User) - JOIN with User table required to retrieve author.nickname and author.name for API response
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp)
  - `publishedAt` (timestamp, nullable)

- **Group**: Represents a group that contains materials.  
  Key attributes:
  - `id` (string, UUID)
  - `name` (string)
  - `isGlobal` (boolean)
  - `visibility` (inferred from context, not stored directly)

- **User**: Represents a user (author or group member).  
  Key attributes:
  - `id` (string, UUID, internal only)
  - `nickname` (string, unique, lowercase)
  - `name` (string)
  - `role` (enum: ADMIN | COACH | CONTESTANT)

> **Note on Data Retrieval**: To return author information in API responses, the system must JOIN the Material table with the User table using `author_id`. The authenticated user's `userId` (from JWT token) is used for authorization checks without requiring additional JOINs. Group membership is verified by checking the GroupMember table.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Group members can view PUBLISHED materials via GET /groups/{groupId}/materials with HTTP 200
- **SC-002**: Authors can see their own DRAFT materials in addition to PUBLISHED materials
- **SC-003**: Admin can see ALL materials (DRAFT and PUBLISHED) in any group
- **SC-004**: Non-members can view PUBLISHED materials in VISIBLE groups with HTTP 200
- **SC-005**: Non-members attempting to view materials in NOT_VISIBLE groups receive HTTP 403 Forbidden
- **SC-006**: Anonymous users can view PUBLISHED materials in VISIBLE groups with HTTP 200
- **SC-007**: Anonymous users attempting to view materials in NOT_VISIBLE groups receive HTTP 401 Unauthorized
- **SC-008**: Non-existent group IDs return HTTP 404 Not Found
- **SC-009**: Pinned materials appear first, sorted by pinnedAt DESC
- **SC-010**: Non-pinned materials are sorted by publishedAt DESC, then createdAt DESC
- **SC-011**: Filters work correctly (pinned, tags) with AND logic
- **SC-012**: Pagination works correctly with default and custom page/limit values
- **SC-013**: Pagination metadata accurately reflects total results and current page
- **SC-014**: Invalid pagination parameters return HTTP 400 Bad Request with validation errors
- **SC-015**: Empty pages return empty materials array with correct pagination metadata
- **SC-016**: All timestamps are returned in ISO 8601 format
- **SC-017**: Null values are correctly returned for pinnedAt and publishedAt when applicable
- **SC-018**: Author information includes nickname and name for all materials
- **SC-019**: Visibility rules are enforced consistently (DRAFT only for author/Admin)
- **SC-020**: Error responses follow consistent JSON format with error code and message
- **SC-021**: Material IDs (UUIDs) are exposed in responses for client-side operations
