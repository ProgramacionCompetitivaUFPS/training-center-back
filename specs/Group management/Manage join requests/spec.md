# Feature Specification: Manage Join Requests

**Created**: 2025-12-28

> **Related Spec**: This spec covers the Lead/Admin perspective for managing join requests. For the user perspective (creating requests, checking status), see [Join Group](../Join%20group/spec.md).

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Lead views pending join requests (Priority: P1)

As a Lead, I want to view pending join requests for my group so that I can review who wants to join and make informed approval decisions.

**Why this priority**: Leads need visibility into who is requesting access to manage group growth effectively. This is essential for the REQUEST join policy to function.

**Independent Test**: Authenticated Lead GET `/api/groups/{g}/requests`. Verify paginated list of pending requests is returned with requester details.

**Acceptance Scenarios**:

1. **Scenario**: Lead lists pending requests

   * **Given** requesting user is lead of group `g`
   * **And** group has several pending join requests
   * **When** lead requests the list of join requests
   * **Then** system returns paginated list of requests with `status = PENDING`
   * **And** response includes requester user data (nickname, name, email)
   * **And** response includes request message and created_at

2. **Scenario**: Lead lists requests with pagination

   * **Given** group has many pending requests
   * **When** lead requests list with page and size parameters
   * **Then** system returns paginated results
   * **And** response includes total count and pagination metadata

3. **Scenario**: Lead filters requests by status

   * **Given** group has requests in different states (PENDING, APPROVED, REJECTED)
   * **When** lead requests list with `status` filter
   * **Then** system returns only requests matching the filter

4. **Scenario**: Non-lead attempts to view requests

   * **Given** requesting user is not lead of the group
   * **When** they try to list join requests
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

5. **Scenario**: Admin views requests for any group

   * **Given** requesting user is Admin (has implicit permissions on all groups)
   * **When** Admin requests join requests list for any group
   * **Then** system returns pending requests successfully

6. **Scenario**: Group has no pending requests

   * **Given** group has no pending join requests
   * **When** lead requests the list
   * **Then** system returns empty list with 200 OK

---

### User Story 2 - Lead approves join request (Priority: P1)

As a Lead, I want to approve a join request so that the requesting user becomes a member of my group.

**Why this priority**: Approval is the core action that completes the REQUEST join flow. Without this, users cannot join groups with REQUEST policy.

**Independent Test**: Authenticated Lead PATCH `/api/groups/{g}/requests/{requestId}` with `status = APPROVED`. Verify membership created, request updated, and audit logged.

**Acceptance Scenarios**:

1. **Scenario**: Lead approves pending request

   * **Given** requesting user is lead of group `g`
   * **And** join request exists with `status = PENDING`
   * **When** lead approves the request
   * **Then** request status is updated to `APPROVED`
   * **And** membership is created with `role = MEMBER`
   * **And** `joined_at` timestamp is recorded
   * **And** audit log entry is created

2. **Scenario**: Lead approves already processed request

   * **Given** request has `status = APPROVED` or `status = REJECTED`
   * **When** lead tries to approve it
   * **Then** system rejects with 400 (`REQUEST_ALREADY_PROCESSED`)

3. **Scenario**: Lead approves request for user who is now already member

   * **Given** request exists with `status = PENDING`
   * **And** user was added via another method (direct add, invitation)
   * **When** lead tries to approve
   * **Then** system rejects with 409 (`ALREADY_MEMBER`)
   * **And** request status is updated to `APPROVED` (cleanup)

4. **Scenario**: Non-lead attempts to approve request

   * **Given** requesting user is not lead of the group
   * **When** they try to approve a request
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 3 - Lead rejects join request (Priority: P1)

As a Lead, I want to reject a join request so that I can control who joins my group and decline inappropriate requests.

**Why this priority**: Rejection is essential for leads to maintain control over group membership and decline requests that don't fit group criteria.

**Independent Test**: Authenticated Lead PATCH `/api/groups/{g}/requests/{requestId}` with `status = REJECTED`. Verify request updated, no membership created, and audit logged.

**Acceptance Scenarios**:

1. **Scenario**: Lead rejects pending request

   * **Given** requesting user is lead of group `g`
   * **And** join request exists with `status = PENDING`
   * **When** lead rejects the request
   * **Then** request status is updated to `REJECTED`
   * **And** no membership is created
   * **And** audit log entry is created

2. **Scenario**: Lead rejects already processed request

   * **Given** request has `status = APPROVED` or `status = REJECTED`
   * **When** lead tries to reject it
   * **Then** system rejects with 400 (`REQUEST_ALREADY_PROCESSED`)

3. **Scenario**: Non-lead attempts to reject request

   * **Given** requesting user is not lead of the group
   * **When** they try to reject a request
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### Edge Cases

* Lead approves request but user was deleted/deactivated → Handle gracefully, mark request as `APPROVED` but skip membership
* Concurrent approval by multiple leads → Use DB transaction, first wins
* Request approved for user who simultaneously joined via OPEN policy change → Return `ALREADY_MEMBER`
* Lead bulk approves/rejects multiple requests → Consider batch endpoint for efficiency
* Group deleted while requests are pending → Requests orphaned, handle gracefully
* Lead demoted while processing request → Check permissions at execution time

## Requirements *(mandatory)*

### Functional Requirements

* **FR-MJR-001**: System MUST allow Leads to view pending join requests for their groups.
* **FR-MJR-002**: System MUST allow Admins to view join requests for any group (implicit permissions).
* **FR-MJR-003**: System MUST support pagination for join request listings.
* **FR-MJR-004**: System MUST support filtering join requests by status (PENDING, APPROVED, REJECTED).
* **FR-MJR-005**: System MUST allow Leads to approve pending join requests.
* **FR-MJR-006**: System MUST allow Leads to reject pending join requests.
* **FR-MJR-007**: System MUST create GroupMember record when request is approved.
* **FR-MJR-008**: System MUST prevent processing already handled requests (return `REQUEST_ALREADY_PROCESSED`).
* **FR-MJR-009**: System MUST handle race condition where user is already member when approving (return `ALREADY_MEMBER`).
* **FR-MJR-010**: System MUST record audit logs for request approval and rejection.
* **FR-MJR-011**: System MUST use database transactions for approval to ensure consistency.
* **FR-MJR-012**: System MUST restrict request management operations to Leads and Admins.

### Key Entities *(include if feature involves data)*

* **JoinRequest**
  * **Description**: User-created request to join a group with REQUEST policy.
  * **Core attributes**:
    * `id` (UUID)
    * `group_id` (UUID)
    * `requester_user_id` (UUID)
    * `message` (text, optional) - User's message explaining why they want to join
    * `status` (enum: `PENDING`, `APPROVED`, `REJECTED`)
    * `created_at` (timestamp)
  * **Constraints**:
    * Unique constraint on (`group_id`, `requester_user_id`) for pending requests

* **GroupMember** (creation aspect)
  * **Description**: Membership record created when request is approved.
  * **Core attributes**:
    * `group_id` (UUID)
    * `user_id` (UUID)
    * `role` (enum: `MEMBER` for approved requests)
    * `joined_at` (timestamp)

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-MJR-001**: Leads can view paginated list of pending join requests with requester details.
* **SC-MJR-002**: Leads can approve pending requests and membership is created automatically.
* **SC-MJR-003**: Leads can reject pending requests.
* **SC-MJR-004**: Non-leads attempting to view/process requests receive 403 Forbidden.
* **SC-MJR-005**: Already processed requests cannot be re-processed (400 `REQUEST_ALREADY_PROCESSED`).
* **SC-MJR-006**: Admins can view and process requests for any group.
* **SC-MJR-007**: All approval/rejection operations are recorded in audit logs.

## API Contract

### GET /api/groups/{groupId}/requests

List join requests for the group (Lead/Admin only).

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead/Admin authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |

**Query Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| status | string | No | Filter by status: `PENDING`, `APPROVED`, `REJECTED` (default: `PENDING`) |
| page | integer | No | Page number for pagination (default: 1) |
| size | integer | No | Number of items per page (default: 20, max: 100) |

**Responses**:

#### 200 OK
Join requests retrieved successfully.

```json
{
  "requests": [
    {
      "id": "string",
      "group_id": "string",
      "requester": {
        "user_id": "string",
        "nickname": "string",
        "name": "string",
        "email": "string"
      },
      "message": "string",
      "status": "PENDING",
      "created_at": "string"
    }
  ],
  "pagination": {
    "page": 1,
    "size": 20,
    "total_items": 45,
    "total_pages": 3
  }
}
```

#### 403 Forbidden
User does not have permission to view requests.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can view join requests"
}
```

---

### PATCH /api/groups/{groupId}/requests/{requestId}

Approve or reject a join request (Lead/Admin only).

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Lead/Admin authentication |

**Path Parameters**:
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string | Yes | ID of the group |
| requestId | string | Yes | ID of the join request |

**Request Body**:
```json
{
  "status": "string"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| status | string | Yes | New status: `APPROVED` or `REJECTED` |

**Responses**:

#### 200 OK
Request processed successfully.

```json
{
  "id": "string",
  "group_id": "string",
  "requester": {
    "user_id": "string",
    "nickname": "string"
  },
  "status": "APPROVED"
}
```

#### 400 Bad Request
Request cannot be processed.

```json
{
  "error": "REQUEST_ALREADY_PROCESSED",
  "message": "This request has already been processed"
}
```

```json
{
  "error": "INVALID_STATUS",
  "message": "Status must be APPROVED or REJECTED"
}
```

#### 403 Forbidden
User does not have permission to process requests.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only leads can process join requests"
}
```

#### 404 Not Found
Request not found.

```json
{
  "error": "REQUEST_NOT_FOUND",
  "message": "Join request not found"
}
```

#### 409 Conflict
User is already a member (race condition).

```json
{
  "error": "ALREADY_MEMBER",
  "message": "User is already a member of this group"
}
```

---

## Notes / Implementation hints

**Permission Checks**
* Verify requesting user is Lead of the group OR has Admin role (system-level)
* Admin has implicit permissions on all groups without requiring membership

**Request Processing**
* Use database transactions when approving requests to ensure membership creation and status update are atomic
* Consider soft-deleting join requests rather than hard deletion for audit purposes
* When approving, check if user is already member (race condition) before creating membership
* Default listing to `status = PENDING` to show actionable requests first

**Pagination & Filtering**
* Implement pagination to handle groups with many requests
* Support filtering by status to view historical requests

**Performance Considerations**
* Index on (`group_id`, `status`) for efficient request listing
* Consider caching request counts for groups with high volume

**Future Considerations**
* Consider notification system to alert users when their request is approved/rejected
* Consider batch endpoint for bulk approve/reject operations for efficiency

---

