# Feature Specification: Create Group

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a new Group (Priority: P1)

As a System Admin or a Coach, I want to create a Group so that a bounded collection of users can share contests and materials with controlled visibility and membership policies.

**Why this priority**: Groups are the unit of isolation and sharing in the system (courses, friend groups, public collections). Without the ability to create groups, coaches and admins can't structure content or organize users.

**Independent Test**: Authenticated request to the Group creation endpoint by a Coach or System Admin. Verify DB persisted Group record, GroupAdmin membership assigned to creator, and proper defaults (visibility, join policy) applied. Verify non-coach/non-admin requests are rejected.

**Acceptance Scenarios**:

1. **Scenario**: Successful creation by a Coach

* **Given** a Coach is authenticated
* **When** they submit a valid CreateGroup request with `name`, optional `description`, `visibility = VISIBLE`, `join_policy = REQUEST`
* **Then** the system returns 201 Created with the created Group data (id, slug, created_at)
* **And** the creator is recorded as Group admin and member (`role = ADMIN`)
* **And** `visibility` and `join_policy` are stored correctly
* **And** the group initially has zero contests/materials and zero members except the creator

2. **Scenario**: Attempt to create by a regular Member (not coach/system admin)

* **Given** a regular user (not Coach) is authenticated
* **When** they attempt to create a Group
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

3. **Scenario**: Create a private invite-only group

* **Given** a Coach or System Admin is authenticated
* **When** they create a Group with `visibility = VISIBLE` and `join_policy = INVITE`
* **Then** the Group is created and non-members can see group content in read-only mode (per visibility rules)
* **And** only invited users can become members

4. **Scenario**: Create an open public group (free entry)

* **Given** a Coach is authenticated
* **When** they create a Group with `visibility = VISIBLE` and `join_policy = OPEN`
* **Then** users can join immediately without approval (JoinPolicy=OPEN behavior enforced)

5. **Scenario**: Create a non-visible group with invalid join policy

* **Given** a Coach is authenticated
* **When** they create a Group with `visibility = NOT_VISIBLE` and `join_policy = OPEN` or `join_policy = REQUEST`
* **Then** the system rejects with 400 Bad Request (`INVALID_POLICY_COMBINATION`)

6. **Scenario**: Create a valid non-visible invite-only group

* **Given** a Coach is authenticated
* **When** they create a Group with `visibility = NOT_VISIBLE` and `join_policy = INVITE`
* **Then** the Group is created successfully
* **And** only invited users can discover and join the group

6. **Scenario**: System creates the global default group (bootstrap)

* **Given** system initialization or migration runs
* **When** the system bootstraps data
* **Then** a special Group (the "global" group) exists with `is_default = true`
* **And** all existing and future users are members of the global group automatically
* **And** the system admin is an admin of this group
* **And** this group cannot be deleted and users cannot leave it

---

### User Story 2 - Create Group with explicit initial members/admins (Priority: P2)

As a Coach or System Admin, I want to optionally add initial members and admins at creation time so the group can be usable immediately with the right staff.

**Why this priority**: Often a coach creates a group and pre-populates it with other coaches or a TA; enabling initial members at creation saves follow-up steps.

**Independent Test**: Create request that contains `initial_members` and `initial_admin_ids`. Verify that only allowed users are added; verify added admins are coaches or system admin; verify non-coach `initial_admin_ids` are rejected.

**Acceptance Scenarios**:

1. **Scenario**: Add valid initial admins

* **Given** creator is a Coach and includes `initial_admin_ids` that are system Coaches
* **When** group is created
* **Then** those users are added to the group as `ADMIN` and as members
* **And** join events (`joined_at`) are recorded

2. **Scenario**: Attempt to add a non-coach as admin

* **Given** creator includes a regular user id in `initial_admin_ids`
* **When** group creation is processed
* **Then** the system rejects the request with 400 Bad Request (`INVALID_ADMIN_ASSIGNMENT`) and no group is created

---

### User Story 3 - Slug/Name uniqueness and reserved names (Priority: P2)

As a System, I want group names/slugs to be unique (and some reserved names forbidden) so URL/lookup collisions and reserved groups (global) cannot be created accidentally.

**Why this priority**: Naming collisions cause confusion and break direct-linking.

**Independent Test**: Attempt to create duplicate name/slug; attempt to create group with reserved name `global` etc.

**Acceptance Scenarios**:

1. **Scenario**: Create with duplicate slug

* **Given** a Group with slug `algorithms-2025` exists
* **When** another coach tries to create a group with name that maps to same slug
* **Then** the system rejects with 409 Conflict (`SLUG_ALREADY_EXISTS`)

2. **Scenario**: Create group with reserved name

* **Given** `global` is a reserved identifier used by the system default group
* **When** a user tries to create a group with name `Global` (case-insensitive)
* **Then** the system rejects with 400 Bad Request (`RESERVED_NAME`)

---

### Edge Cases

* What happens when the creator provides an invalid `initial_admin_ids` list? → Entire request must fail (atomic create) with descriptive error.
* How does system handle creation when a system-level member limit per group is reached? → If a system limit (FR-XXX) exists, reject with 400 (`GROUP_MEMBER_LIMIT`).
* Creating a group with `join_policy = OPEN` but `visibility = NOT_VISIBLE` → This combination is invalid; system must reject (`INVALID_POLICY_COMBINATION`) because non-visible groups cannot allow open joining.
* Creating a group with `join_policy = REQUEST` but `visibility = NOT_VISIBLE` → This combination is invalid; system must reject (`INVALID_POLICY_COMBINATION`) because users cannot request to join groups they cannot see.
* What happens when System Admin attempts to set `is_default = true` in the create endpoint? → Creation endpoints must reject attempts to set `is_default`; only bootstrap/migration may set it (`FORBIDDEN_FIELD`).
* Name/slug normalization: whitespace/diacritics/uppercase must be normalized before uniqueness checks; collisions after normalization must be treated as conflicts.
* Attempts to create group without leaving at least one admin (e.g., providing `initial_admin_ids` empty and creator excluded) → reject (GROUP_MUST_HAVE_ADMIN).

## Requirements *(mandatory)*

### Functional Requirements

* **FR-001**: System MUST allow System Admins and Coaches to create Groups.
* **FR-002**: System MUST reject group creation attempts by non-Coach, non-Admin users with 403 (`INSUFFICIENT_PERMISSIONS`).
* **FR-003**: When a group is created, the creator MUST be added as a Group member and assigned role `ADMIN` for that group.
* **FR-004**: Groups MUST have attributes: `id`, `name`, `slug`, `description`, `visibility` (`VISIBLE` | `NOT_VISIBLE`), `join_policy` (`INVITE` | `REQUEST` | `OPEN`), `created_by`, `created_at`, `is_default` (boolean), `members_count`.
* **FR-005**: System MUST record membership entries (`GroupMember`) for the creator with `role = ADMIN` and `joined_at` timestamp.
* **FR-006**: System MUST enforce that only System Admin and Coaches (system-level) can be assigned the ADMIN role inside a group.
* **FR-007**: System MUST persist groups with a unique `slug`. Attempts to create with an existing slug MUST fail (409 `SLUG_ALREADY_EXISTS`).
* **FR-008**: System MUST allow optional `initial_members` and `initial_admin_ids` in request payload. When provided, initial_admins MUST be validated as Coaches or System Admin.
* **FR-009**: System MUST store `visibility` and `join_policy` and enforce these rules at runtime (discovery, join flows).
* **FR-010**: System MUST record `joined_at` for every initial added member and for the creator when group is created.
* **FR-011**: System MUST forbid creation of `is_default = true` via regular CreateGroup endpoint. The bootstrap/migration code is responsible for default/global group creation.
* **FR-012**: System MUST prevent creation of groups with zero members. (Creator is always added; attempts to explicitly clear members on create are invalid.)
* **FR-013**: System MUST validate combinations of `visibility` and `join_policy`. Groups with `visibility = NOT_VISIBLE` MUST only allow `join_policy = INVITE`.
* **FR-014**: System MUST record `created_by` with the creator user id.
* **FR-015**: System MUST return meaningful error codes/messages for invalid inputs (e.g., `RESERVED_NAME`, `SLUG_ALREADY_EXISTS`, `INVALID_ADMIN_ASSIGNMENT`, `INVALID_POLICY_COMBINATION`, `FORBIDDEN_FIELD`).

* **FR-016**: System MUST capture critical group creation events in audit logs (who created, timestamp, group configuration).
* **FR-017**: System MUST ensure only System Admin and Coaches can be assigned Group Admin role.
* **FR-018**: System MUST create a default/global Group during bootstrap that contains all users and cannot be deleted.
* **FR-019**: System MUST ensure groups have at least one admin member at all times.
* **FR-020**: System MUST make all membership changes auditable with timestamps and role information.

### Key Entities *(include if feature involves data)*

* **Group**

  * **Description**: A logical container for members, contests, and materials.
  * **Core attributes**:

    * `id` (UUID)
    * `name` (string)
    * `slug` (string, unique, URL-safe)
    * `description` (text, optional)
    * `visibility` (enum: `VISIBLE`, `NOT_VISIBLE`)
    * `join_policy` (enum: `INVITE`, `REQUEST`, `OPEN`)
    * `is_default` (bool) — only set by system bootstrap
    * `created_by` (user id)
    * `created_at` (timestamp)
    * `members_count` (integer, derived)
  * **Relationships**:

    * has many `GroupMember`
    * has many `Contests`
    * has many `Materials`

* **GroupMember**

  * **Description**: Join record linking a user to a group.
  * **Core attributes**:

    * `group_id` (UUID)
    * `user_id` (UUID)
    * `role` (enum: `ADMIN`, `MEMBER`)
    * `joined_at` (timestamp)

* **JoinPolicy** (concept)

  * `INVITE`: only admins can invite users; membership by invitation only.
  * `REQUEST`: users request to join and admins approve/deny.
  * `OPEN`: any user may join immediately (unless group is not visible; see BR & FR).

* **Visibility** (concept)

  * `VISIBLE`: non-members can discover and read group content *in read-only mode*.
  * `NOT_VISIBLE`: non-members cannot see or discover the group; membership only via invite.

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-001**: 100% of CreateGroup requests from valid System Admins/Coaches result in a persistent Group record with creator assigned as ADMIN (tested via automated integration tests).
* **SC-002**: Attempted group creation by non-authorized users is rejected with 403 in all tested cases.
* **SC-003**: The system enforces slug uniqueness: attempts to create duplicate slugs fail with 409 (`SLUG_ALREADY_EXISTS`).
* **SC-004**: The system enforces ADMIN assignment rules: only Coaches/System Admin can be assigned as group ADMIN; invalid assignments rejected with 400.
* **SC-005**: The bootstrap process guarantees existence of the global default group upon system start (test: after migration/bootstrap, the `global` group exists and contains all users; cannot be deleted).
* **SC-006**: Visibility + join_policy rules are enforced: `NOT_VISIBLE` + `OPEN` combination is rejected at create time; `VISIBLE` groups allow non-member read-only discovery.
* **SC-007**: For created groups, `joined_at` timestamps are present for the creator and any provided `initial_members` (verified by unit tests).
* **SC-008**: Deleting a non-global group triggers hard delete and removes associated contests and materials (integration test coverage).

## Example API (informational, optional)

**Request** (POST `/api/groups`)

```json
{
  "name": "Algorithms 101 - 2026",
  "description": "Group for the Algorithms course",
  "visibility": "VISIBLE",
  "join_policy": "REQUEST",
  "initial_admin_ids": ["user-uuid-coach-1"],
  "initial_members": ["user-uuid-student-1", "user-uuid-student-2"]
}
```

**Success Response** (201 Created)

```json
{
  "id": "group-uuid-123",
  "name": "Algorithms 101 - 2026",
  "slug": "algorithms-101-2026",
  "description": "Group for the Algorithms course",
  "visibility": "VISIBLE",
  "join_policy": "REQUEST",
  "created_by": "user-uuid-coach-1",
  "created_at": "2025-12-28T12:00:00Z",
  "members_count": 3
}
```

**Error Responses**

* `403 Forbidden` — when an unauthorized user attempts to create a group. (`INSUFFICIENT_PERMISSIONS`)
* `400 Bad Request` — invalid `initial_admin_ids` (non-coach) (`INVALID_ADMIN_ASSIGNMENT`), inconsistent visibility/join_policy (`INCONSISTENT_JOIN_VISIBILITY`), reserved name (`RESERVED_NAME`), forbidden field attempt (`FORBIDDEN_FIELD`).
* `409 Conflict` — slug already exists. (`SLUG_ALREADY_EXISTS`)

## Notes / Implementation hints

* Slug generation must be deterministic and normalized (lowercase, remove punctuation, collapse spaces, strip diacritics).
* Enforce uniqueness at DB level (unique index on `slug`) and in application logic to return user-friendly errors.
* `is_default` must not be writable by regular endpoints — only migration/bootstrap scripts should set it.
* When `initial_members` or `initial_admin_ids` are provided, treat group creation as atomic (either all members added and group created, or roll back).
* Audit log entry should include `created_by`, `payload summary`, and `request origin`.
* Ensure integration tests simulate the "global group" semantics: after new user creation, the user is auto-added to the global group.
* Do not allow `visibility = NOT_VISIBLE` with `join_policy = OPEN` or `join_policy = REQUEST`; validate and reject with `INVALID_POLICY_COMBINATION`.
* Ensure membership events record `joined_at` and creator is always added as `ADMIN`.
* Hard delete: when a group is deleted (non-global), delete associated contests and materials; keep referenced problem entities intact.
* Account anonymization (user deletion) is governed by `selfdeactivate account` spec — deleted/anonimized users must remain referenced in historical records (submissions, group membership history) with anonymized display fields.
* The create endpoint must reject attempts to set `is_default`, `members_count` or other computed/guarded fields in request payload (`FORBIDDEN_FIELD`).
* Problems are global entities and may be referenced by contests inside any group; creating a group does not duplicate problem entities.
* Deleting a group results in hard deletion of that group's contests and materials; problem entities referenced by those contests remain in the system.
* Data model should scale to groups with large numbers of members (no hard-coded per-group limits unless system policy later introduces one).

---
