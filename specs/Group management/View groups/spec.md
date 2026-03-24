# Feature Specification: View Groups

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – List all visible groups (Priority: P2)

As a user, I want to browse all visible groups so that I can discover groups to join and see what's available on the platform.

**Why this priority**: Improves discoverability and user engagement, but not critical for basic functionality (users can still be invited to groups).

**Independent Test**: This user story can be tested independently by consuming the `GET /api/groups` endpoint with valid authentication, validating that visible groups are returned with proper filtering and pagination.

**Acceptance Scenarios**:

1. **Scenario**: User lists all visible groups
   * **Given** multiple groups exist with different visibility settings
   * **And** the authenticated user is a regular member
   * **When** they request the list of groups
   * **Then** the system returns all VISIBLE groups
   * **And** does NOT include NOT_VISIBLE groups (unless user is member)
   * **And** includes the global group
   * **And** results are paginated

2. **Scenario**: User sees their NOT_VISIBLE groups in the list
   * **Given** user is a member of a NOT_VISIBLE group
   * **When** they request the list of groups
   * **Then** the system includes that NOT_VISIBLE group in the results
   * **And** also includes all VISIBLE groups

3. **Scenario**: Admin sees all groups
   * **Given** the authenticated user is Admin
   * **When** they request the list of groups
   * **Then** the system returns ALL groups (VISIBLE and NOT_VISIBLE)
   * **And** includes groups where Admin is not a member

4. **Scenario**: Search groups by name
   * **Given** multiple groups exist
   * **When** user searches with `?search=programming`
   * **Then** only groups with "programming" in name or description are returned

5. **Scenario**: Filter by visibility
   * **Given** user is member of both VISIBLE and NOT_VISIBLE groups
   * **When** they request with `?visibility=VISIBLE`
   * **Then** only VISIBLE groups are returned

6. **Scenario**: Filter by join policy
   * **Given** multiple groups with different join policies
   * **When** user requests with `?joinPolicy=OPEN`
   * **Then** only groups with OPEN policy are returned

7. **Scenario**: Filter by active contests
   * **Given** some groups have active contests
   * **When** user requests with `?hasActiveContests=true`
   * **Then** only groups with at least one ACTIVE contest are returned

8. **Scenario**: Sort by member count
   * **Given** multiple groups with different member counts
   * **When** user requests with `?sortBy=memberCount&order=desc`
   * **Then** groups are ordered by member count descending

9. **Scenario**: Empty results
   * **Given** no groups match the filters
   * **When** user requests with specific filters
   * **Then** system returns empty array with 200 OK

---

### User Story 2 – View group details (Priority: P2)

As a user, I want to view detailed information about a specific group so that I can decide if I want to join it.

**Why this priority**: Essential for informed decision-making about joining groups.

**Independent Test**: This user story can be tested independently by consuming the `GET /api/groups/{id}` endpoint, validating that group details and statistics are returned correctly.

**Acceptance Scenarios**:

1. **Scenario**: User views details of a VISIBLE group
   * **Given** a VISIBLE group exists
   * **And** the authenticated user is not a member
   * **When** they request the group details
   * **Then** the system returns full group information
   * **And** includes statistics (member count, contest count, material count)
   * **And** includes list of leads

2. **Scenario**: User views details of a NOT_VISIBLE group they're member of
   * **Given** a NOT_VISIBLE group exists
   * **And** the authenticated user is a member
   * **When** they request the group details
   * **Then** the system returns full group information

3. **Scenario**: User tries to view NOT_VISIBLE group they're not member of
   * **Given** a NOT_VISIBLE group exists
   * **And** the authenticated user is NOT a member
   * **And** the user is NOT Admin
   * **When** they request the group details
   * **Then** the system rejects with 404 Not Found

4. **Scenario**: Admin views any group
   * **Given** any group exists (VISIBLE or NOT_VISIBLE)
   * **And** the authenticated user is Admin
   * **When** they request the group details
   * **Then** the system returns full group information

5. **Scenario**: View global group details
   * **Given** the global group exists
   * **When** any authenticated user requests its details
   * **Then** the system returns full information
   * **And** indicates it's the global group (isGlobal = true)

6. **Scenario**: Group not found
   * **Given** no group exists with the provided ID
   * **When** user requests the group details
   * **Then** the system rejects with 404 Not Found

---

### User Story 3 – View my groups dashboard (Priority: P2)

As a user, I want to see a personalized dashboard of my groups so that I can quickly access the groups I'm part of.

**Why this priority**: Significantly improves UX by providing quick access to relevant groups.

**Independent Test**: This user story can be tested independently by consuming the `GET /api/users/me/groups` endpoint, validating that only user's groups are returned with proper role indication.

**Acceptance Scenarios**:

1. **Scenario**: User views their groups
   * **Given** user is a member of 3 groups and lead of 2 groups
   * **When** they request their groups dashboard
   * **Then** the system returns all 5 groups
   * **And** indicates their role in each group (MEMBER or LEAD)
   * **And** does NOT include the global group (unless preference says otherwise)

2. **Scenario**: User with hideGlobalGroup = false
   * **Given** user has preference `hideGlobalGroup = false` (or not set)
   * **When** they request their groups dashboard
   * **Then** the system includes the global group in the results

3. **Scenario**: User with hideGlobalGroup = true
   * **Given** user has preference `hideGlobalGroup = true`
   * **When** they request their groups dashboard
   * **Then** the system does NOT include the global group

4. **Scenario**: Filter my groups by role
   * **Given** user is member of some groups and lead of others
   * **When** they request with `?role=LEAD`
   * **Then** only groups where user is LEAD are returned

5. **Scenario**: Search within my groups
   * **Given** user is member of multiple groups
   * **When** they search with `?search=competitive`
   * **Then** only their groups matching the search are returned

6. **Scenario**: New user with no groups
   * **Given** user just registered and hasn't joined any groups
   * **When** they request their groups dashboard
   * **Then** system returns only the global group (if not hidden)

7. **Scenario**: Admin views their groups
   * **Given** user is Admin
   * **When** they request their groups dashboard
   * **Then** system returns groups where Admin is explicitly a member
   * **And** does NOT return all groups (Admin's implicit permissions don't count as membership)

---

### Edge Cases

* User is member of 100+ groups - pagination handles large datasets
* Group with only leads (no regular members) - memberCount includes leads
* Group with 0 contests and 0 materials - statistics show zeros
* Concurrent group creation while listing - eventual consistency acceptable
* Search with special characters - properly escaped and handled
* Filter combinations that return no results - empty array with 200 OK
* Global group always has isGlobal = true flag
* Deleted/deactivated users in lead lists - show anonymized nicknames
* Sort by field that has ties (e.g., same member count) - secondary sort by name
* User preferences not set - use default values (hideGlobalGroup = false)

---

## API Contract

### GET /api/groups

List all groups accessible to the authenticated user.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Query Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| search | string | No | Search in group name and description (case-insensitive) |
| visibility | enum | No | Filter by visibility: `VISIBLE` or `NOT_VISIBLE` |
| joinPolicy | enum | No | Filter by join policy: `INVITE`, `REQUEST`, or `OPEN` |
| hasActiveContests | boolean | No | Filter groups with active contests |
| sortBy | enum | No | Sort field: `name`, `createdAt`, `memberCount` (default: `name`) |
| order | enum | No | Sort order: `asc` or `desc` (default: `asc`) |
| page | integer | No | Page number for pagination (default: 1) |
| limit | integer | No | Items per page (default: 20, max: 50) |

**Responses**:

#### 200 OK - Regular user

```json
{
  "groups": [
    {
      "id": "group-uuid-1",
      "name": "Competitive Programming Club",
      "description": "Weekly contests and practice sessions",
      "visibility": "VISIBLE",
      "joinPolicy": "REQUEST",
      "isGlobal": false,
      "memberCount": 45,
      "leadCount": 3,
      "contestCount": 12,
      "materialCount": 8,
      "activeContestCount": 2,
      "userRole": null,
      "createdAt": "2026-01-15T10:00:00Z"
    },
    {
      "id": "global-group-uuid",
      "name": "Global Training Center",
      "description": "Public contests and materials for all users",
      "visibility": "VISIBLE",
      "joinPolicy": "OPEN",
      "isGlobal": true,
      "memberCount": 1523,
      "leadCount": 5,
      "contestCount": 45,
      "materialCount": 120,
      "activeContestCount": 3,
      "userRole": "MEMBER",
      "createdAt": "2025-01-01T00:00:00Z"
    },
    {
      "id": "group-uuid-2",
      "name": "Advanced Algorithms Study Group",
      "description": null,
      "visibility": "NOT_VISIBLE",
      "joinPolicy": "INVITE",
      "isGlobal": false,
      "memberCount": 12,
      "leadCount": 2,
      "contestCount": 5,
      "materialCount": 15,
      "activeContestCount": 0,
      "userRole": "MEMBER",
      "createdAt": "2026-02-01T14:30:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 3,
    "totalPages": 1,
    "hasNextPage": false,
    "hasPrevPage": false
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| groups | array | List of groups |
| groups[].id | UUID | Group identifier |
| groups[].name | string | Group name |
| groups[].description | string | Group description (nullable) |
| groups[].visibility | enum | VISIBLE or NOT_VISIBLE |
| groups[].joinPolicy | enum | INVITE, REQUEST, or OPEN |
| groups[].isGlobal | boolean | Whether this is the global group |
| groups[].memberCount | integer | Total number of members (including leads) |
| groups[].leadCount | integer | Number of leads |
| groups[].contestCount | integer | Total number of contests |
| groups[].materialCount | integer | Total number of materials |
| groups[].activeContestCount | integer | Number of currently ACTIVE contests |
| groups[].userRole | enum | User's role in this group: MEMBER, LEAD, or null (not a member) |
| groups[].createdAt | timestamp | When the group was created |
| pagination | object | Pagination information |

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 400 Bad Request

```json
{
  "error": "INVALID_PARAMETER",
  "message": "Invalid sortBy value. Must be: name, createdAt, or memberCount"
}
```

---

### GET /api/groups/{groupId}

Get detailed information about a specific group.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The unique identifier of the group |

**Responses**:

#### 200 OK

```json
{
  "id": "group-uuid-1",
  "name": "Competitive Programming Club",
  "description": "Weekly contests and practice sessions for all skill levels",
  "visibility": "VISIBLE",
  "joinPolicy": "REQUEST",
  "isGlobal": false,
  "statistics": {
    "memberCount": 45,
    "leadCount": 3,
    "contestCount": 12,
    "materialCount": 8,
    "activeContestCount": 2,
    "scheduledContestCount": 3,
    "finishedContestCount": 7
  },
  "leads": [
    {
      "userId": "user-uuid-1",
      "nickname": "coach_john",
      "name": "John Smith"
    },
    {
      "userId": "user-uuid-2",
      "nickname": "prof_alice",
      "name": "Alice Johnson"
    }
  ],
  "userMembership": {
    "isMember": false,
    "role": null,
    "joinedAt": null,
    "hasPendingRequest": false,
    "hasPendingInvitation": true
  },
  "createdAt": "2026-01-15T10:00:00Z",
  "updatedAt": "2026-02-10T15:30:00Z"
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Group identifier |
| name | string | Group name |
| description | string | Group description (nullable) |
| visibility | enum | VISIBLE or NOT_VISIBLE |
| joinPolicy | enum | INVITE, REQUEST, or OPEN |
| isGlobal | boolean | Whether this is the global group |
| statistics | object | Group statistics |
| statistics.memberCount | integer | Total members (including leads) |
| statistics.leadCount | integer | Number of leads |
| statistics.contestCount | integer | Total contests |
| statistics.materialCount | integer | Total materials |
| statistics.activeContestCount | integer | Currently ACTIVE contests |
| statistics.scheduledContestCount | integer | SCHEDULED contests |
| statistics.finishedContestCount | integer | FINISHED contests |
| leads | array | List of group leads |
| leads[].userId | UUID | Lead's user ID |
| leads[].nickname | string | Lead's nickname |
| leads[].name | string | Lead's full name |
| userMembership | object | Current user's relationship with the group |
| userMembership.isMember | boolean | Whether user is a member |
| userMembership.role | enum | User's role: MEMBER, LEAD, or null |
| userMembership.joinedAt | timestamp | When user joined (null if not member) |
| userMembership.hasPendingRequest | boolean | Whether user has a pending join request |
| userMembership.hasPendingInvitation | boolean | Whether user has a pending invitation |
| createdAt | timestamp | When the group was created |
| updatedAt | timestamp | When the group was last updated |

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 404 Not Found

```json
{
  "error": "GROUP_NOT_FOUND",
  "message": "Group not found"
}
```

> **Note**: NOT_VISIBLE groups return 404 for non-members (not 403) to avoid leaking group existence.

---

### GET /api/users/me/groups

Get the authenticated user's groups dashboard.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Query Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| role | enum | No | Filter by user's role: `MEMBER` or `LEAD` |
| search | string | No | Search in group name and description |
| sortBy | enum | No | Sort field: `name`, `joinedAt`, `memberCount` (default: `name`) |
| order | enum | No | Sort order: `asc` or `desc` (default: `asc`) |
| page | integer | No | Page number for pagination (default: 1) |
| limit | integer | No | Items per page (default: 20, max: 50) |

**Responses**:

#### 200 OK

```json
{
  "groups": [
    {
      "id": "group-uuid-1",
      "name": "Competitive Programming Club",
      "description": "Weekly contests and practice sessions",
      "visibility": "VISIBLE",
      "joinPolicy": "REQUEST",
      "isGlobal": false,
      "myRole": "LEAD",
      "joinedAt": "2026-01-20T10:00:00Z",
      "memberCount": 45,
      "contestCount": 12,
      "materialCount": 8,
      "activeContestCount": 2,
      "unreadNotifications": 3
    },
    {
      "id": "group-uuid-2",
      "name": "Advanced Algorithms Study Group",
      "description": null,
      "visibility": "NOT_VISIBLE",
      "joinPolicy": "INVITE",
      "isGlobal": false,
      "myRole": "MEMBER",
      "joinedAt": "2026-02-01T14:30:00Z",
      "memberCount": 12,
      "contestCount": 5,
      "materialCount": 15,
      "activeContestCount": 0,
      "unreadNotifications": 0
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 2,
    "totalPages": 1,
    "hasNextPage": false,
    "hasPrevPage": false
  }
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| groups | array | List of user's groups |
| groups[].id | UUID | Group identifier |
| groups[].name | string | Group name |
| groups[].description | string | Group description (nullable) |
| groups[].visibility | enum | VISIBLE or NOT_VISIBLE |
| groups[].joinPolicy | enum | INVITE, REQUEST, or OPEN |
| groups[].isGlobal | boolean | Whether this is the global group |
| groups[].myRole | enum | User's role: MEMBER or LEAD |
| groups[].joinedAt | timestamp | When user joined this group |
| groups[].memberCount | integer | Total members |
| groups[].contestCount | integer | Total contests |
| groups[].materialCount | integer | Total materials |
| groups[].activeContestCount | integer | Currently ACTIVE contests |
| groups[].unreadNotifications | integer | Number of unread notifications for this group |
| pagination | object | Pagination information |

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**List All Groups (GET /api/groups)**

* **FR-VG-001**: System MUST return all VISIBLE groups to any authenticated user.
* **FR-VG-002**: System MUST include NOT_VISIBLE groups where the user is a member.
* **FR-VG-003**: Admin MUST see all groups (VISIBLE and NOT_VISIBLE) regardless of membership.
* **FR-VG-004**: System MUST include the global group in the list.
* **FR-VG-005**: System MUST support search by group name and description (case-insensitive).
* **FR-VG-006**: System MUST support filtering by visibility, joinPolicy, and hasActiveContests.
* **FR-VG-007**: System MUST support sorting by name, createdAt, and memberCount.
* **FR-VG-008**: System MUST indicate user's role in each group (MEMBER, LEAD, or null).
* **FR-VG-009**: System MUST include statistics: memberCount, leadCount, contestCount, materialCount, activeContestCount.

**View Group Details (GET /api/groups/{id})**

* **FR-VG-010**: System MUST return full details for VISIBLE groups to any authenticated user.
* **FR-VG-011**: System MUST return full details for NOT_VISIBLE groups to members and Admin.
* **FR-VG-012**: System MUST return 404 for NOT_VISIBLE groups to non-members (not 403).
* **FR-VG-013**: System MUST include detailed statistics (active, scheduled, finished contests).
* **FR-VG-014**: System MUST include list of group leads with nickname and name.
* **FR-VG-015**: System MUST include user's membership status (isMember, role, joinedAt).
* **FR-VG-016**: System MUST indicate if user has pending request or invitation.
* **FR-VG-017**: System MUST indicate if group is the global group (isGlobal flag).

**My Groups Dashboard (GET /api/users/me/groups)**

* **FR-VG-018**: System MUST return all groups where user is a member or lead.
* **FR-VG-019**: System MUST exclude global group if user preference `hideGlobalGroup = true`.
* **FR-VG-020**: System MUST include global group if preference is false or not set.
* **FR-VG-021**: System MUST support filtering by user's role (MEMBER or LEAD).
* **FR-VG-022**: System MUST support search within user's groups.
* **FR-VG-023**: System MUST indicate user's role in each group (myRole field).
* **FR-VG-024**: System MUST include joinedAt timestamp for each group.
* **FR-VG-025**: Admin's dashboard MUST only show groups where Admin is explicitly a member (implicit permissions don't count as membership).

**Pagination**

* **FR-VG-026**: System MUST support pagination with page and limit parameters.
* **FR-VG-027**: Default limit MUST be 20, maximum limit MUST be 50.
* **FR-VG-028**: System MUST return total count, total pages, hasNextPage, hasPrevPage.

**Statistics Calculation**

* **FR-VG-029**: memberCount MUST include both members and leads.
* **FR-VG-030**: activeContestCount MUST only count contests with status ACTIVE.
* **FR-VG-031**: Statistics MUST be calculated in real-time (not cached).

**User Preferences**

* **FR-VG-032**: System MUST read `hideGlobalGroup` from User.preferences JSON field.
* **FR-VG-033**: Default value for `hideGlobalGroup` MUST be false if not set.
* **FR-VG-034**: Preference MUST only affect `GET /users/me/groups` endpoint.

### Key Entities

* **Group**: Existing entity
  * All fields from model (id, name, description, visibility, joinPolicy, isGlobal, createdAt, updatedAt)

* **User**: Extended with preferences
  * `preferences` (json, nullable) - User preferences including `hideGlobalGroup`

* **GroupMember**: Existing entity for membership tracking
  * Used to determine user's role and membership status

### Permission Matrix

| Endpoint | Regular User | Member | Lead | Admin |
|----------|--------------|--------|------|-------|
| GET /groups | VISIBLE groups + own NOT_VISIBLE | VISIBLE + own NOT_VISIBLE | VISIBLE + own NOT_VISIBLE | ALL groups |
| GET /groups/{id} (VISIBLE) | ✅ | ✅ | ✅ | ✅ |
| GET /groups/{id} (NOT_VISIBLE) | ❌ 404 | ✅ | ✅ | ✅ |
| GET /users/me/groups | Own groups | Own groups | Own groups | Own groups |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-VG-001**: Users can list all visible groups via `GET /api/groups` with HTTP 200.
* **SC-VG-002**: Users can search and filter groups with proper results.
* **SC-VG-003**: Users can view details of visible groups and their own groups.
* **SC-VG-004**: NOT_VISIBLE groups return 404 for non-members.
* **SC-VG-005**: My Groups dashboard shows user's groups with correct role indication.
* **SC-VG-006**: Global group is hidden from dashboard when preference is set.
* **SC-VG-007**: Admin sees all groups in list but only their memberships in dashboard.
* **SC-VG-008**: Statistics are accurate and up-to-date.
* **SC-VG-009**: Pagination works correctly with configurable page and limit.
* **SC-VG-010**: Sorting and filtering work as expected.

---

## Optional Notes

* **Performance**: Consider caching group statistics for large groups
* **Real-time updates**: Consider WebSocket/SSE for live member count updates
* **Search optimization**: Consider full-text search index for better performance
* **User preferences**: The `preferences` JSON field can be extended with other settings (theme, language, notifications)
* **Related specs**:
  * Create group: For creating new groups
  * Join group: For joining groups from the list
  * Manage group members: For viewing member details
  * Update group: For modifying group settings

