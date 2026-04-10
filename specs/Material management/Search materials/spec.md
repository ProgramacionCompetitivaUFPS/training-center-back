# Feature Specification: Search Materials

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Search materials by text (Priority: P2)

As a group member, I want to search materials by text content so that I can quickly find relevant information without browsing through all materials.

**Why this priority**: Improves user experience significantly for groups with many materials, but not critical since basic listing already exists.

**Independent Test**: Authenticated group member performs GET `/groups/{groupId}/materials?q=algorithm` and verifies only materials containing "algorithm" in title or content are returned, ordered by relevance.

**Acceptance Scenarios**:

1. **Scenario**: Search with text term in title
   * **Given** a group has materials with titles "Algorithm Basics" and "Data Structures"
   * **And** a user is authenticated and is a member of the group
   * **When** they request GET /groups/{groupId}/materials?q=algorithm
   * **Then** the system returns materials with "algorithm" in the title
   * **And** results are ordered by relevance (search score)
   * **And** only PUBLISHED materials are included

2. **Scenario**: Search with text term in content
   * **Given** a group has materials with "sorting algorithms" mentioned in content
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=sorting
   * **Then** the system returns materials with "sorting" in the content
   * **And** results are ordered by relevance

3. **Scenario**: Search is case-insensitive
   * **Given** a group has materials with "Algorithm" in title
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=ALGORITHM
   * **Then** the system returns the same results as searching for "algorithm"

4. **Scenario**: Search with no results
   * **Given** a group has materials but none match the search term
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=nonexistent
   * **Then** the system returns HTTP 200 with empty materials array
   * **And** pagination shows totalCount=0

5. **Scenario**: Search without query parameter (list all)
   * **Given** a group has multiple PUBLISHED materials
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials (no `q` parameter)
   * **Then** the system returns all PUBLISHED materials
   * **And** results are ordered by publishedAt DESC (not by relevance)

6. **Scenario**: Title matches rank higher than content matches
   * **Given** material A has "algorithm" in title, material B has "algorithm" only in content
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm
   * **Then** material A appears before material B in results

---

### User Story 2 - Filter search results (Priority: P2)

As a group member, I want to combine search with filters so that I can narrow down results to specific topics, authors, or time periods.

**Why this priority**: Enhances search usability for large groups with diverse content.

**Independent Test**: Perform search with multiple filters and verify all filters are applied correctly with AND logic.

**Acceptance Scenarios**:

1. **Scenario**: Filter by tags (AND logic)
   * **Given** a group has materials with various tags
   * **And** material A has tags ["algorithm", "sorting"]
   * **And** material B has tags ["algorithm", "graph"]
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&tags=algorithm,sorting
   * **Then** only material A is returned (has BOTH tags)

2. **Scenario**: Filter by author nickname
   * **Given** a group has materials from multiple authors
   * **And** user "coach_john" authored some materials
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&author=coach_john
   * **Then** only materials by coach_john matching "algorithm" are returned

3. **Scenario**: Filter by date range (from)
   * **Given** a group has materials published on different dates
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&publishedFrom=2026-02-01
   * **Then** only materials published on or after 2026-02-01 are returned

4. **Scenario**: Filter by date range (to)
   * **Given** a group has materials published on different dates
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&publishedTo=2026-02-15
   * **Then** only materials published on or before 2026-02-15 are returned

5. **Scenario**: Filter by date range (from and to)
   * **Given** a group has materials published on different dates
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?publishedFrom=2026-02-01&publishedTo=2026-02-15
   * **Then** only materials published between those dates (inclusive) are returned

6. **Scenario**: Combine all filters
   * **Given** a group has diverse materials
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&tags=sorting&author=coach_john&publishedFrom=2026-02-01
   * **Then** only materials matching ALL criteria are returned

---

### User Story 3 - Control search result ordering (Priority: P2)

As a user, I want to control how search results are sorted so that I can view results in the most useful order for my needs.

**Why this priority**: Different use cases require different sorting (relevance for search, date for recent content, title for alphabetical browsing).

**Independent Test**: Perform searches with different sort parameters and verify correct ordering.

**Acceptance Scenarios**:

1. **Scenario**: Default sort with search term (relevance)
   * **Given** a group has materials matching a search term
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm (no sort parameter)
   * **Then** results are ordered by relevance (search score) descending

2. **Scenario**: Default sort without search term (publishedAt)
   * **Given** a group has multiple PUBLISHED materials
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials (no q, no sort)
   * **Then** results are ordered by publishedAt DESC

3. **Scenario**: Explicit sort by relevance
   * **Given** a group has materials matching a search term
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&sort=relevance
   * **Then** results are ordered by search score descending

4. **Scenario**: Sort by publishedAt
   * **Given** a group has materials with different publication dates
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&sort=publishedAt
   * **Then** results are ordered by publishedAt DESC (most recent first)

5. **Scenario**: Sort by title
   * **Given** a group has materials with different titles
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm&sort=title
   * **Then** results are ordered by title ASC (alphabetically)

6. **Scenario**: Pinned materials mixed by relevance
   * **Given** a group has pinned and non-pinned materials matching search
   * **And** a user is authenticated and is a member
   * **When** they request GET /groups/{groupId}/materials?q=algorithm
   * **Then** pinned materials are NOT automatically at the top
   * **And** all materials are ranked by relevance regardless of pinned status

---

### User Story 4 - Search with proper permissions (Priority: P2)

As a system, I want to enforce the same visibility rules for search as for listing so that users only see materials they have permission to view.

**Why this priority**: Security and consistency with existing View material permissions.

**Independent Test**: Test search with different user types and group visibilities, verify correct access control.

**Acceptance Scenarios**:

1. **Scenario**: Member searches in their group
   * **Given** a user is authenticated and is a member of a group
   * **When** they search materials in that group
   * **Then** the system returns matching PUBLISHED materials

2. **Scenario**: Non-member searches in VISIBLE group
   * **Given** a group has visibility=VISIBLE
   * **And** a user is authenticated but NOT a member
   * **When** they search materials in that group
   * **Then** the system returns matching PUBLISHED materials (read-only)

3. **Scenario**: Non-member attempts to search in NOT_VISIBLE group
   * **Given** a group has visibility=NOT_VISIBLE
   * **And** a user is authenticated but NOT a member
   * **When** they attempt to search materials in that group
   * **Then** the system rejects with HTTP 403 Forbidden

4. **Scenario**: Admin searches in any group
   * **Given** a user has Admin role
   * **When** they search materials in any group
   * **Then** the system returns matching PUBLISHED materials

5. **Scenario**: Anonymous user searches in VISIBLE group
   * **Given** a group has visibility=VISIBLE
   * **When** an unauthenticated user searches materials
   * **Then** the system returns matching PUBLISHED materials

6. **Scenario**: Anonymous user attempts to search in NOT_VISIBLE group
   * **Given** a group has visibility=NOT_VISIBLE
   * **When** an unauthenticated user attempts to search
   * **Then** the system rejects with HTTP 401 Unauthorized

7. **Scenario**: DRAFT materials excluded from search
   * **Given** a group has DRAFT and PUBLISHED materials
   * **And** a user is authenticated and is a member
   * **When** they search materials
   * **Then** only PUBLISHED materials are returned
   * **And** DRAFT materials are NOT included (even author's own drafts)

---

### Edge Cases

* Search term with special characters (properly escaped)
* Search term with Unicode characters
* Very long search terms (>100 chars)
* Empty search term (treated as no search, list all)
* Search with only whitespace (treated as empty)
* Invalid date format in publishedFrom/publishedTo
* publishedFrom > publishedTo (invalid range)
* Author nickname that doesn't exist (returns empty results)
* Tags that don't exist (returns empty results)
* Invalid sort parameter (returns 400 Bad Request)
* Pagination with search (page beyond available results)
* Concurrent searches by multiple users
* Search in group with no materials
* Search in non-existent group (404 Not Found)

---

## API Contract

### Endpoint

```
GET /groups/{groupId}/materials
```

> **Note**: This extends the existing View material endpoint with additional query parameters for search functionality.

### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | No | Bearer token for authentication (optional for VISIBLE groups, required for NOT_VISIBLE groups) |

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group to search materials in |

### Query Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| q | string | No | Search term (case-insensitive, searches title and content) |
| tags | string | No | Comma-separated tags (AND logic, e.g., "algorithm,sorting") |
| author | string | No | Author nickname to filter by |
| publishedFrom | string | No | Start date for publication range (ISO 8601 date: YYYY-MM-DD) |
| publishedTo | string | No | End date for publication range (ISO 8601 date: YYYY-MM-DD) |
| sort | string | No | Sort order: `relevance`, `publishedAt`, `title` (default: `relevance` if q provided, else `publishedAt`) |
| page | integer | No | Page number (default: 1, minimum: 1) |
| limit | integer | No | Items per page (default: 20, minimum: 1, maximum: 100) |

> **Backward Compatibility**: The existing `pinned` parameter from View material is still supported and can be combined with search parameters.

### Success Response (200 OK)

```json
{
  "materials": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Algorithm Basics",
      "content": "# Introduction to Algorithms\n\nAlgorithms are step-by-step procedures...",
      "tags": ["algorithm", "beginner"],
      "status": "PUBLISHED",
      "pinned": false,
      "pinnedAt": null,
      "author": {
        "nickname": "coach_john",
        "name": "John Smith"
      },
      "createdAt": "2026-02-01T10:00:00Z",
      "updatedAt": "2026-02-01T10:00:00Z",
      "publishedAt": "2026-02-01T12:00:00Z"
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "title": "Data Structures Overview",
      "content": "This material covers various data structures including algorithms for traversal...",
      "tags": ["algorithm", "data-structures"],
      "status": "PUBLISHED",
      "pinned": true,
      "pinnedAt": "2026-02-05T10:00:00Z",
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
    "totalCount": 2,
    "currentPage": 1,
    "totalPages": 1,
    "itemsPerPage": 20
  }
}
```

### Field Descriptions

| Field | Type | Description |
|-------|------|-------------|
| materials | array | Array of material objects matching search criteria |
| materials[].id | string | Material identifier (UUID) |
| materials[].title | string | Material title |
| materials[].content | string | Markdown content |
| materials[].tags | string[] | User-defined tags |
| materials[].status | enum | Always PUBLISHED (DRAFT excluded from search) |
| materials[].pinned | boolean | Whether material is pinned |
| materials[].pinnedAt | string \| null | When material was pinned (ISO 8601 or null) |
| materials[].author | object | Author information (nickname, name) |
| materials[].createdAt | string | Creation timestamp (ISO 8601) |
| materials[].updatedAt | string | Last update timestamp (ISO 8601) |
| materials[].publishedAt | string | Publication timestamp (ISO 8601, never null for PUBLISHED) |
| pagination | object | Pagination metadata |
| pagination.totalCount | integer | Total number of materials matching search |
| pagination.currentPage | integer | Current page number |
| pagination.totalPages | integer | Total number of pages |
| pagination.itemsPerPage | integer | Number of items per page |

### Error Responses

#### 400 Bad Request - Invalid Parameters

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request parameters",
  "details": [
    {
      "field": "sort",
      "message": "Sort must be one of: relevance, publishedAt, title"
    }
  ]
}
```

#### 400 Bad Request - Invalid Date Format

```json
{
  "error": "INVALID_DATE_FORMAT",
  "message": "Date must be in ISO 8601 format (YYYY-MM-DD)",
  "field": "publishedFrom",
  "providedValue": "2026/02/01"
}
```

#### 400 Bad Request - Invalid Date Range

```json
{
  "error": "INVALID_DATE_RANGE",
  "message": "publishedFrom must be before or equal to publishedTo",
  "publishedFrom": "2026-02-15",
  "publishedTo": "2026-02-01"
}
```

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 403 Forbidden

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "You do not have permission to view materials in this group"
}
```

#### 404 Not Found

```json
{
  "error": "NOT_FOUND",
  "message": "Group not found"
}
```

---

## Functional Requirements

### Search Functionality

- **FR-001**: The system MUST perform case-insensitive full-text search when `q` parameter is provided.
- **FR-002**: The system MUST search simultaneously in both `title` and `content` fields.
- **FR-003**: The system MUST assign higher weight to matches in `title` than matches in `content`.
- **FR-004**: The system MUST rank exact matches higher than partial matches.
- **FR-005**: The system MUST consider term frequency in relevance scoring.
- **FR-006**: The system MUST treat empty or whitespace-only `q` parameter as no search term.
- **FR-007**: The system MUST return all materials when no `q` parameter is provided (backward compatible with View material).

### Material Status Filtering

- **FR-008**: The system MUST include only materials with `PUBLISHED` status in search results.
- **FR-009**: The system MUST exclude all materials with `DRAFT` status from search results.
- **FR-010**: The system MUST exclude author's own DRAFT materials from search results.

### Tag Filtering

- **FR-011**: The system MUST support filtering by tags using AND logic (all specified tags must be present).
- **FR-012**: The system MUST accept comma-separated tag list in `tags` parameter.
- **FR-013**: The system MUST return empty results when filtering by non-existent tags (not an error).

### Author Filtering

- **FR-014**: The system MUST support filtering by author nickname via `author` parameter.
- **FR-015**: The system MUST return empty results when filtering by non-existent author (not an error).

### Date Range Filtering

- **FR-016**: The system MUST support filtering by publication date range via `publishedFrom` and `publishedTo` parameters.
- **FR-017**: The system MUST accept ISO 8601 date format (YYYY-MM-DD) for date parameters.
- **FR-018**: The system MUST validate that `publishedFrom` is before or equal to `publishedTo`.
- **FR-019**: The system MUST return HTTP 400 Bad Request for invalid date formats.
- **FR-020**: The system MUST return HTTP 400 Bad Request for invalid date ranges.
- **FR-021**: The system MUST apply date filtering inclusively (materials published on boundary dates are included).

### Filter Combination

- **FR-022**: The system MUST support combining all filters simultaneously (q, tags, author, date range).
- **FR-023**: The system MUST apply all filters using AND logic.

### Result Ordering

- **FR-024**: The system MUST order results by relevance (search score) descending when `q` parameter is provided and no `sort` parameter is specified.
- **FR-025**: The system MUST order results by `publishedAt` descending when no `q` parameter is provided and no `sort` parameter is specified.
- **FR-026**: The system MUST support explicit sorting via `sort` parameter with values: `relevance`, `publishedAt`, `title`.
- **FR-027**: The system MUST order by search score descending when `sort=relevance`.
- **FR-028**: The system MUST order by publication date descending when `sort=publishedAt`.
- **FR-029**: The system MUST order by title ascending (alphabetically) when `sort=title`.
- **FR-030**: The system MUST return HTTP 400 Bad Request for invalid `sort` values.

### Pinned Material Handling

- **FR-031**: The system MUST rank pinned materials by relevance along with other materials when searching.
- **FR-032**: The system MUST NOT automatically place pinned materials at the top of search results.

### Pagination

- **FR-033**: The system MUST support pagination with `page` and `limit` query parameters.
- **FR-034**: The system MUST default to page=1 and limit=20 if pagination parameters are not provided.
- **FR-035**: The system MUST validate that `page` is a positive integer (minimum 1).
- **FR-036**: The system MUST validate that `limit` is between 1 and 100 (inclusive).
- **FR-037**: The system MUST enforce a maximum limit of 100 items per page.
- **FR-038**: The system MUST return pagination metadata (totalCount, currentPage, totalPages, itemsPerPage).
- **FR-039**: The system MUST return empty materials array with correct pagination metadata when requested page exceeds available pages.

### Permission Enforcement

- **FR-040**: The system MUST allow authenticated group members to search materials in their group.
- **FR-041**: The system MUST allow non-members to search materials in VISIBLE groups.
- **FR-042**: The system MUST reject non-members attempting to search materials in NOT_VISIBLE groups with HTTP 403 Forbidden.
- **FR-043**: The system MUST allow Admin to search materials in any group regardless of membership or visibility.
- **FR-044**: The system MUST allow anonymous users to search materials in VISIBLE groups.
- **FR-045**: The system MUST reject anonymous users attempting to search materials in NOT_VISIBLE groups with HTTP 401 Unauthorized.
- **FR-046**: The system MUST return HTTP 404 Not Found for non-existent group IDs.

### Backward Compatibility

- **FR-047**: The system MUST maintain backward compatibility with existing View material functionality when no search parameters are provided.
- **FR-048**: The system MUST support the existing `pinned` filter parameter in combination with search parameters.

---

## Non-Functional Requirements

- **NFR-001**: Search queries MUST complete within 1 second for groups with up to 1000 materials.
- **NFR-002**: The system MUST use database full-text search capabilities (not application-level filtering).
- **NFR-003**: The system MUST properly index `title` and `content` fields for full-text search performance.
- **NFR-004**: The system MUST handle concurrent search requests gracefully.
- **NFR-005**: The system SHOULD cache frequently searched terms for improved performance.

---

## Data Model

### Key Entities

- **Material**: The content being searched.
  Key attributes:
  - `id` (UUID, primary key)
  - `title` (string, 1-200 chars, normalized NFKC, indexed for full-text search)
  - `content` (string, 0-50000 chars, Markdown format, indexed for full-text search)
  - `tags` (string[], each 2-50 chars)
  - `status` (enum: `DRAFT` | `PUBLISHED`)
  - `pinned` (boolean)
  - `pinnedAt` (timestamp, nullable)
  - `group_id` (UUID, FK to Group, indexed)
  - `author_id` (UUID, FK to User, indexed)
  - `publishedAt` (timestamp, nullable, indexed)

- **Group**: The group containing materials.
  Key attributes:
  - `id` (UUID)
  - `visibility` (enum: `VISIBLE` | `NOT_VISIBLE`)

- **User**: The author of materials.
  Key attributes:
  - `id` (UUID)
  - `nickname` (string, unique, indexed)
  - `role` (enum: `ADMIN` | `COACH` | `CONTESTANT`)

### Database Indexes

For optimal search performance, the following indexes are required:

```sql
-- Full-text search indexes
CREATE INDEX idx_material_title_fts ON Material USING GIN (to_tsvector('english', title));
CREATE INDEX idx_material_content_fts ON Material USING GIN (to_tsvector('english', content));

-- Filter indexes
CREATE INDEX idx_material_group_id ON Material (group_id);
CREATE INDEX idx_material_author_id ON Material (author_id);
CREATE INDEX idx_material_status ON Material (status);
CREATE INDEX idx_material_published_at ON Material (publishedAt);

-- Composite index for common queries
CREATE INDEX idx_material_group_status_published ON Material (group_id, status, publishedAt DESC);
```

> **Note**: The exact index syntax depends on the database system used (PostgreSQL, MySQL, MongoDB, etc.). The above example uses PostgreSQL syntax.

---

## Security Considerations

- **SEC-001**: The system MUST enforce the same visibility rules for search as for View material.
- **SEC-002**: The system MUST validate all input parameters to prevent SQL injection.
- **SEC-003**: The system MUST sanitize search terms to prevent malicious queries.
- **SEC-004**: The system SHOULD implement rate limiting to prevent search abuse.
- **SEC-005**: The system MUST NOT expose DRAFT materials in search results.

---

## Optional Notes

- **Search Highlighting**: Future enhancement could return snippets with highlighted search terms.
- **Search Suggestions**: Future enhancement could provide autocomplete suggestions based on popular searches.
- **Advanced Search**: Future enhancement could support boolean operators (AND, OR, NOT) in search queries.
- **Search Analytics**: Consider tracking popular search terms for content improvement insights.
- **Performance**: For very large groups (>10,000 materials), consider implementing search result caching.
- **Relevance Tuning**: The relevance scoring algorithm may need tuning based on user feedback.
- **Related Specs**:
  - View material: Base functionality for listing materials
  - Create material: For creating searchable content
  - Update material: For modifying searchable content
  - Change material visibility: For publishing materials (making them searchable)
