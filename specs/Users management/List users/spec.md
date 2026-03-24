# Feature Specification: List Users (Admin)

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 - List all users with pagination (Priority: P2)

As a system administrator, I want to list all users in the platform with pagination so that I can browse and manage the user base efficiently.

**Why this priority**: Useful for user administration and management, but not critical since individual user operations already exist.

**Independent Test**: Authenticated Admin performs GET `/admin/users` and verifies all users are returned with correct pagination metadata.

**Acceptance Scenarios**:

1. **Scenario**: Successful user listing by admin
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users
   * **Then** the system returns HTTP 200 with all users
   * **And** results include id, email, name, nickname, country, city, institution, role, status, createdAt, updatedAt, deactivatedAt
   * **And** results are paginated with default page=1 and limit=20
   * **And** results are ordered by createdAt DESC (most recent first)

2. **Scenario**: Non-admin attempts to list users
   * **Given** an authenticated user has role CONTESTANT or COACH
   * **When** they attempt to list users
   * **Then** the system rejects with HTTP 403 Forbidden (ADMIN_REQUIRED)

3. **Scenario**: Unauthenticated request
   * **Given** the request does not include valid authentication credentials
   * **When** a list users request is submitted
   * **Then** the system rejects with HTTP 401 Unauthorized

4. **Scenario**: Custom pagination
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?page=2&limit=50
   * **Then** the system returns users 51-100
   * **And** pagination metadata shows currentPage=2, totalPages calculated correctly

5. **Scenario**: Deactivated users included in list
   * **Given** some users have status DEACTIVATED
   * **And** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users
   * **Then** deactivated users are included in the results
   * **And** deactivated users show email=null and anonymized nickname

---

### User Story 2 - Filter users by role and status (Priority: P2)

As an administrator, I want to filter users by role and status so that I can find specific user groups quickly.

**Why this priority**: Enhances admin efficiency when managing different user types.

**Independent Test**: Perform requests with different filter combinations and verify correct filtering with OR logic for multiple values.

**Acceptance Scenarios**:

1. **Scenario**: Filter by single role
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?role=COACH
   * **Then** only users with role COACH are returned

2. **Scenario**: Filter by multiple roles (OR logic)
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?role=COACH,ADMIN
   * **Then** users with role COACH OR ADMIN are returned

3. **Scenario**: Filter by status
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?status=ACTIVE
   * **Then** only users with status ACTIVE are returned

4. **Scenario**: Filter by country
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?country=Colombia
   * **Then** only users from Colombia are returned

5. **Scenario**: Filter by city
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?city=Bogotá
   * **Then** only users from Bogotá are returned

6. **Scenario**: Filter by institution
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?institution=Universidad Nacional
   * **Then** only users from Universidad Nacional are returned

7. **Scenario**: Combine multiple filters (AND logic)
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?role=COACH&status=ACTIVE&country=Colombia
   * **Then** only users matching ALL criteria are returned

---

### User Story 3 - Search users by text (Priority: P2)

As an administrator, I want to search users by name, nickname, email, or institution so that I can quickly find specific users.

**Why this priority**: Essential for efficient user lookup and management.

**Independent Test**: Perform searches with different search fields and verify correct case-insensitive matching.

**Acceptance Scenarios**:

1. **Scenario**: Search by name
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?searchField=name&searchTerm=john
   * **Then** only users with "john" in their name are returned (case-insensitive)

2. **Scenario**: Search by nickname
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?searchField=nickname&searchTerm=coach
   * **Then** only users with "coach" in their nickname are returned (case-insensitive)

3. **Scenario**: Search by email
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?searchField=email&searchTerm=example.com
   * **Then** only users with "example.com" in their email are returned (case-insensitive)

4. **Scenario**: Search by institution
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?searchField=institution&searchTerm=universidad
   * **Then** only users with "universidad" in their institution are returned (case-insensitive)

5. **Scenario**: Search with no results
   * **Given** an authenticated user has the ADMIN role
   * **When** they search for a term that matches no users
   * **Then** the system returns HTTP 200 with empty users array
   * **And** pagination shows totalCount=0

6. **Scenario**: Search without specifying field (searches all)
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?searchTerm=john (no searchField)
   * **Then** the system searches in name, nickname, email, and institution simultaneously
   * **And** returns users matching in ANY of those fields

7. **Scenario**: Combine search with filters
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?searchTerm=john&role=COACH&status=ACTIVE
   * **Then** only users matching the search term AND filters are returned

---

### User Story 4 - Sort users by different fields (Priority: P2)

As an administrator, I want to sort users by different fields so that I can view users in the most useful order.

**Why this priority**: Flexibility in viewing user data for different administrative tasks.

**Independent Test**: Perform requests with different sort parameters and verify correct ordering.

**Acceptance Scenarios**:

1. **Scenario**: Default sort (createdAt DESC)
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users (no sort parameter)
   * **Then** users are ordered by createdAt DESC (most recent first)

2. **Scenario**: Sort by createdAt ascending
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?sort=createdAt&order=asc
   * **Then** users are ordered by createdAt ASC (oldest first)

3. **Scenario**: Sort by name
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?sort=name&order=asc
   * **Then** users are ordered by name ASC (alphabetically)

4. **Scenario**: Sort by nickname
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?sort=nickname&order=asc
   * **Then** users are ordered by nickname ASC (alphabetically)

5. **Scenario**: Sort by email
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?sort=email&order=asc
   * **Then** users are ordered by email ASC (alphabetically)
   * **And** users with null email (deactivated) appear at the end

6. **Scenario**: Sort by deactivatedAt
   * **Given** an authenticated user has the ADMIN role
   * **When** they request GET /admin/users?sort=deactivatedAt&order=desc
   * **Then** users are ordered by deactivatedAt DESC
   * **And** users with null deactivatedAt (active) appear at the end

---

### Edge Cases

- Empty user database (returns empty array with correct pagination)
- All users deactivated (all show anonymized data)
- Search term with special characters (properly escaped)
- Search term with Unicode characters
- Invalid sort field (returns 400 Bad Request)
- Invalid order value (returns 400 Bad Request)
- Invalid role filter value (returns 400 Bad Request)
- Invalid status filter value (returns 400 Bad Request)
- Page beyond available results (returns empty array)
- Limit exceeds maximum (capped at 100)
- Concurrent admin requests (handled independently)
- Very large result sets (pagination handles efficiently)

---

## API Contract

### GET /admin/users

List all users in the platform with optional filters, search, sorting, and pagination.

> **Important**: 
> - Only ADMIN users can access this endpoint
> - Deactivated users show email=null and anonymized nickname
> - All user fields are returned (except password and internal id)

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for admin authentication |

**Query Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| role | string | No | Filter by role (comma-separated for OR logic): ADMIN, COACH, CONTESTANT |
| status | string | No | Filter by status: ACTIVE or DEACTIVATED |
| country | string | No | Filter by country (exact match, case-insensitive) |
| city | string | No | Filter by city (exact match, case-insensitive) |
| institution | string | No | Filter by institution (exact match, case-insensitive) |
| searchField | string | No | Field to search in: name, nickname, email, institution, all (default: all) |
| searchTerm | string | No | Search term (case-insensitive, partial match) |
| sort | string | No | Sort field: createdAt, name, nickname, email, deactivatedAt (default: createdAt) |
| order | string | No | Sort order: asc, desc (default: desc) |
| page | integer | No | Page number (default: 1, minimum: 1) |
| limit | integer | No | Items per page (default: 20, minimum: 1, maximum: 100) |

**Success Response (200 OK)**:

```json
{
  "users": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "email": "john.doe@example.com",
      "name": "John Doe",
      "nickname": "john_doe",
      "country": "Colombia",
      "city": "Bogotá",
      "institution": "Universidad Nacional",
      "role": "COACH",
      "status": "ACTIVE",
      "createdAt": "2026-01-15T10:00:00Z",
      "updatedAt": "2026-02-01T14:30:00Z",
      "deactivatedAt": null
    },
    {
      "id": "660e8400-e29b-41d4-a716-446655440001",
      "email": null,
      "name": "Jane Smith",
      "nickname": "user_anonimo_a1b2c3d4e5",
      "country": "Colombia",
      "city": "Medellín",
      "institution": "Universidad de Antioquia",
      "role": "CONTESTANT",
      "status": "DEACTIVATED",
      "createdAt": "2025-12-20T08:00:00Z",
      "updatedAt": "2026-01-10T16:45:00Z",
      "deactivatedAt": "2026-01-10T16:45:00Z"
    }
  ],
  "pagination": {
    "totalCount": 150,
    "currentPage": 1,
    "totalPages": 8,
    "itemsPerPage": 20
  }
}
```

**Field Descriptions**:

| Field | Type | Description |
|-------|------|-------------|
| users | array | Array of user objects |
| users[].id | string | User identifier (UUID) |
| users[].email | string \| null | User email (null for deactivated users) |
| users[].name | string | User's full name |
| users[].nickname | string | User's nickname (anonymized for deactivated users) |
| users[].country | string | User's country |
| users[].city | string | User's city |
| users[].institution | string | User's institution |
| users[].role | enum | ADMIN, COACH, or CONTESTANT |
| users[].status | enum | ACTIVE or DEACTIVATED |
| users[].createdAt | string | Account creation timestamp (ISO 8601) |
| users[].updatedAt | string | Last modification timestamp (ISO 8601, nullable) |
| users[].deactivatedAt | string \| null | Deactivation timestamp (ISO 8601 or null) |
| pagination | object | Pagination metadata |
| pagination.totalCount | integer | Total number of users matching filters |
| pagination.currentPage | integer | Current page number |
| pagination.totalPages | integer | Total number of pages |
| pagination.itemsPerPage | integer | Number of items per page |

**Error Responses**:

#### 400 Bad Request - Invalid Parameters

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request parameters",
  "details": [
    {
      "field": "role",
      "message": "Role must be one of: ADMIN, COACH, CONTESTANT"
    }
  ]
}
```

```json
{
  "error": "INVALID_SORT_FIELD",
  "message": "Sort field must be one of: createdAt, name, nickname, email, deactivatedAt"
}
```

```json
{
  "error": "INVALID_SORT_ORDER",
  "message": "Sort order must be: asc or desc"
}
```

```json
{
  "error": "INVALID_SEARCH_FIELD",
  "message": "Search field must be one of: name, nickname, email, institution, all"
}
```

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden

```json
{
  "error": "ADMIN_REQUIRED",
  "message": "Admin privileges required"
}
```

---

## Functional Requirements

### Authorization

- **FR-001**: The system MUST allow only users with ADMIN role to access this endpoint.
- **FR-002**: The system MUST reject non-admin users with HTTP 403 Forbidden.
- **FR-003**: The system MUST reject unauthenticated requests with HTTP 401 Unauthorized.

### User Listing

- **FR-004**: The system MUST return all users in the platform when no filters are applied.
- **FR-005**: The system MUST include deactivated users in the results.
- **FR-006**: The system MUST return all user fields except password.
- **FR-007**: For deactivated users, the system MUST return email as null and nickname in anonymized format.

### Filtering

- **FR-008**: The system MUST support filtering by role with OR logic for multiple values.
- **FR-009**: The system MUST support filtering by status (single value: ACTIVE or DEACTIVATED).
- **FR-010**: The system MUST support filtering by country (exact match, case-insensitive).
- **FR-011**: The system MUST support filtering by city (exact match, case-insensitive).
- **FR-012**: The system MUST support filtering by institution (exact match, case-insensitive).
- **FR-013**: The system MUST apply all filters using AND logic when multiple filters are provided.
- **FR-014**: The system MUST validate role filter values (ADMIN, COACH, CONTESTANT).
- **FR-015**: The system MUST validate status filter value (ACTIVE or DEACTIVATED).

### Search

- **FR-016**: The system MUST support case-insensitive partial match search.
- **FR-017**: The system MUST support searching in specific fields: name, nickname, email, institution.
- **FR-018**: When searchField is "all" or not specified, the system MUST search in name, nickname, email, and institution simultaneously.
- **FR-019**: The system MUST return users matching the search term in ANY of the specified fields (OR logic).
- **FR-020**: The system MUST allow combining search with filters (AND logic).
- **FR-021**: The system MUST validate searchField values.

### Sorting

- **FR-022**: The system MUST support sorting by: createdAt, name, nickname, email, deactivatedAt.
- **FR-023**: The system MUST support ascending (asc) and descending (desc) order.
- **FR-024**: The system MUST default to createdAt DESC when no sort parameters are provided.
- **FR-025**: The system MUST default to DESC order when sort field is provided but order is not.
- **FR-026**: The system MUST handle null values in sorting (place at end for ASC, at end for DESC).
- **FR-027**: The system MUST validate sort field and order values.

### Pagination

- **FR-028**: The system MUST support pagination with page and limit query parameters.
- **FR-029**: The system MUST default to page=1 and limit=20 if pagination parameters are not provided.
- **FR-030**: The system MUST validate that page is a positive integer (minimum 1).
- **FR-031**: The system MUST validate that limit is between 1 and 100 (inclusive).
- **FR-032**: The system MUST enforce a maximum limit of 100 items per page.
- **FR-033**: The system MUST return pagination metadata (totalCount, currentPage, totalPages, itemsPerPage).
- **FR-034**: The system MUST return empty users array with correct pagination metadata when requested page exceeds available pages.

### Data Completeness

- **FR-035**: The system MUST return id, email, name, nickname, country, city, institution, role, status, createdAt, updatedAt, and deactivatedAt for all users.
- **FR-036**: The system MUST return timestamps in ISO 8601 format.
- **FR-037**: The system MUST return null for email and deactivatedAt when applicable.
- **FR-038**: The system MUST NOT return password field.

---

## Non-Functional Requirements

- **NFR-001**: User listing MUST complete within 2 seconds for up to 10,000 users.
- **NFR-002**: The system MUST use database indexes for efficient filtering and sorting.
- **NFR-003**: The system MUST handle concurrent admin requests gracefully.
- **NFR-004**: The system SHOULD cache frequently accessed user lists for improved performance.

---

## Data Model

### Key Entities

- **User**: Registered person in the system.
  Key attributes:
  - `id` (UUID, primary key)
  - `email` (string, UNIQUE, nullable - NULL for deactivated users, indexed)
  - `name` (string, indexed)
  - `nickname` (string, UNIQUE, lowercase, indexed)
  - `country` (string, indexed)
  - `city` (string, indexed)
  - `institution` (string, indexed)
  - `role` (enum: ADMIN | COACH | CONTESTANT, indexed)
  - `status` (enum: ACTIVE | DEACTIVATED, indexed)
  - `createdAt` (timestamp, indexed)
  - `updatedAt` (timestamp, nullable)
  - `deactivatedAt` (timestamp, nullable, indexed)

### Database Indexes

For optimal query performance, the following indexes are recommended:

```sql
-- Filter indexes
CREATE INDEX idx_user_role ON User (role);
CREATE INDEX idx_user_status ON User (status);
CREATE INDEX idx_user_country ON User (country);
CREATE INDEX idx_user_city ON User (city);
CREATE INDEX idx_user_institution ON User (institution);

-- Sort indexes
CREATE INDEX idx_user_created_at ON User (createdAt DESC);
CREATE INDEX idx_user_name ON User (name);
CREATE INDEX idx_user_nickname ON User (nickname);
CREATE INDEX idx_user_email ON User (email);
CREATE INDEX idx_user_deactivated_at ON User (deactivatedAt DESC);

-- Search indexes (for text search)
CREATE INDEX idx_user_name_text ON User USING GIN (to_tsvector('english', name));
CREATE INDEX idx_user_nickname_text ON User USING GIN (to_tsvector('english', nickname));
CREATE INDEX idx_user_email_text ON User USING GIN (to_tsvector('english', email));
CREATE INDEX idx_user_institution_text ON User USING GIN (to_tsvector('english', institution));

-- Composite indexes for common queries
CREATE INDEX idx_user_role_status ON User (role, status);
CREATE INDEX idx_user_status_created_at ON User (status, createdAt DESC);
```

> **Note**: The exact index syntax depends on the database system used. The above example uses PostgreSQL syntax.

---

## Security Considerations

- **SEC-001**: Only ADMIN users can access this endpoint.
- **SEC-002**: Password field MUST NEVER be returned in responses.
- **SEC-003**: The system MUST validate all input parameters to prevent injection attacks.
- **SEC-004**: The system SHOULD implement rate limiting to prevent abuse.
- **SEC-005**: Deactivated users' email and original nickname are not exposed (shown as null and anonymized).

---

## Optional Notes

- **Audit Log Access**: Future enhancement could provide access to DeactivationAuditLog to see original email/nickname of deactivated users.
- **Export Functionality**: Future enhancement could allow exporting user list as CSV/Excel.
- **Bulk Operations**: Future enhancement could support bulk user operations (bulk role change, bulk deactivation).
- **Advanced Filters**: Future enhancement could add filters by date range (createdAt, deactivatedAt).
- **User Statistics**: Future enhancement could include statistics (submission count, group count, etc.).
- **Related Specs**:
  - Create user: For user registration
  - Admin update user: For modifying user data
  - Admin deactivate user: For deactivating users
  - Get user information: For viewing individual user details
