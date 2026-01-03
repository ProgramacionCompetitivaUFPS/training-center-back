# Feature Specification: Create Group

**Created**: 2025-12-28

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a new Group (Priority: P1)

As an Admin or a Coach, I want to create a Group so that a collection of users can share contests and materials with controlled visibility and membership policies.

**Why this priority**: Groups are the unit of isolation and sharing in the system (courses, friend groups, public collections). Without the ability to create groups, coaches and admins can't structure content or organize users.

**Independent Test**: Authenticated request to the Group creation endpoint by a Coach or Admin. Verify DB persisted Group record, lead membership assigned to creator, and proper defaults (visibility, join policy) applied. Verify non-coach/non-admin requests are rejected.

**Acceptance Scenarios**:

1. **Scenario**: Successful creation by a Coach

* **Given** a Coach is authenticated
* **When** they submit a valid CreateGroup request with `name`, optional `description`, `visibility = VISIBLE`, `join_policy = REQUEST`
* **Then** the system returns 201 Created with the created Group data (id, slug, created_at)
* **And** the creator is recorded as Lead and member (`role = LEAD`)
* **And** `visibility` and `join_policy` are stored correctly
* **And** the group initially has zero contests/materials and zero members except the creator

2. **Scenario**: Attempt to create by a regular Member (not coach/admin)

* **Given** a regular user (not Coach) is authenticated
* **When** they attempt to create a Group
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

3. **Scenario**: Create a private invite-only group

* **Given** a Coach or Admin is authenticated
* **When** they create a Group with `visibility = VISIBLE` and `join_policy = INVITE`
* **Then** the Group is created and non-members can see group content in read-only mode (per visibility rules)
* **And** the creator is recorded as Lead and member (`role = LEAD`)
* **And** only invited users can become members

4. **Scenario**: Create an open public group (free entry)

* **Given** a Coach is authenticated
* **When** they create a Group with `visibility = VISIBLE` and `join_policy = OPEN`
* **Then** users can join immediately without approval (JoinPolicy=OPEN behavior enforced)
* **And** the creator is recorded as Lead and member (`role = LEAD`)

5. **Scenario**: Create a non-visible group with invalid join policy

* **Given** a Coach is authenticated
* **When** they create a Group with `visibility = NOT_VISIBLE` and `join_policy = OPEN` or `join_policy = REQUEST`
* **Then** the system rejects with 400 Bad Request (`INVALID_POLICY_COMBINATION`)

6. **Scenario**: Create a valid non-visible invite-only group

* **Given** a Coach is authenticated
* **When** they create a Group with `visibility = NOT_VISIBLE` and `join_policy = INVITE`
* **Then** the Group is created successfully
* **And** the creator is recorded as Lead and member (`role = LEAD`)
* **And** only invited users can discover and join the group

6. **Scenario**: System creates the global default group (bootstrap)

* **Given** system initialization or migration runs
* **When** the system bootstraps data
* **Then** a special Group (the "global" group) exists with `is_default = true`
* **And** all existing and future users are members of the global group automatically
* **And** this group cannot be deleted and users cannot leave it

---

### User Story 2 - Create Group with explicit initial members/leads (Priority: P2)

As a Coach or Admin, I want to optionally add initial members and leads at creation time by providing their nicknames so the group can be usable immediately with the right staff.

**Why this priority**: Often a coach creates a group and pre-populates it with other coaches or a TA; enabling initial members at creation saves follow-up steps.

**Independent Test**: Create request that contains `initial_member_nicknames` and `initial_lead_nicknames`. Verify that only allowed users are added; verify added leads are coaches or admins; verify non-coach nicknames in `initial_lead_nicknames` are rejected; verify nicknames not found are rejected; verify duplicates within same list are silently deduplicated; verify creator's nickname is automatically added as LEAD.

**Acceptance Scenarios**:

1. **Scenario**: Add valid initial leads by nicknames

* **Given** creator is a Coach with nickname `coach_creator`
* **And** includes `initial_lead_nicknames` with system Coach nicknames: `["coach_john", "coach_mary"]`
* **When** group is created
* **Then** those users are added to the group as `LEAD` and as members
* **And** the creator is also added as `LEAD` automatically
* **And** join events (`joined_at`) are recorded for all members

2. **Scenario**: Add valid initial members by nicknames

* **Given** creator is a Coach
* **And** includes `initial_member_nicknames` with valid user nicknames: `["student_alice", "student_bob"]`
* **When** group is created
* **Then** those users are added to the group as `MEMBER`
* **And** the creator is added as `LEAD` automatically

3. **Scenario**: Attempt to add a non-coach as lead by nickname

* **Given** creator includes a contestant nickname in `initial_lead_nicknames`: `["contestant_john"]`
* **When** group creation is processed
* **Then** the system rejects the request with 400 Bad Request (`INVALID_LEAD_ASSIGNMENT`) and no group is created

4. **Scenario**: Attempt to add non-existent nicknames

* **Given** creator includes nicknames that don't exist: `["nonexistent1", "nonexistent2"]`
* **When** group creation is processed
* **Then** the system rejects with 400 (`INVALID_INITIAL_MEMBERS`) with details showing which nicknames were not found

5. **Scenario**: Nickname appears in both lists

* **Given** creator includes same nickname in both lists: `initial_lead_nicknames: ["coach_john"]` and `initial_member_nicknames: ["coach_john"]`
* **When** group creation is processed
* **Then** the system rejects with 400 (`DUPLICATE_NICKNAME_IN_LISTS`)

6. **Scenario**: Creator includes own nickname in lead list

* **Given** creator with nickname `coach_creator` includes it in `initial_lead_nicknames: ["coach_creator", "coach_john"]`
* **When** group is created
* **Then** the duplicate is silently ignored (creator is LEAD by default)
* **And** group is created successfully with creator and coach_john as leads

7. **Scenario**: Creator includes own nickname in member list

* **Given** creator with nickname `coach_creator` includes it in `initial_member_nicknames: ["coach_creator", "student_alice"]`
* **When** group creation is processed
* **Then** the system rejects with 400 (`DUPLICATE_NICKNAME_IN_LISTS`) because creator is automatically LEAD

8. **Scenario**: Duplicate nicknames within same list

* **Given** creator includes duplicates in same list: `initial_lead_nicknames: ["coach_john", "coach_john", "coach_mary"]`
* **When** group is created
* **Then** duplicates are silently deduplicated
* **And** group is created successfully with coach_john and coach_mary as leads (each added once)

9. **Scenario**: Multiple validation errors

* **Given** creator includes: `initial_lead_nicknames: ["nonexistent1", "contestant_bob"]` and `initial_member_nicknames: ["nonexistent2"]`
* **When** group creation is processed
* **Then** the system rejects with 400 (`INVALID_INITIAL_MEMBERS`) with all errors showing which nicknames were not found and which don't have appropriate roles for leads

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

* What happens when the creator provides invalid nicknames in `initial_lead_nicknames` or `initial_member_nicknames`? → Entire request must fail (atomic create) with descriptive error showing all validation issues.
* Creating a group with `join_policy = OPEN` but `visibility = NOT_VISIBLE` → This combination is invalid; system must reject (`INVALID_POLICY_COMBINATION`) because non-visible groups cannot allow open joining.
* Creating a group with `join_policy = REQUEST` but `visibility = NOT_VISIBLE` → This combination is invalid; system must reject (`INVALID_POLICY_COMBINATION`) because users cannot request to join groups they cannot see.
* What happens when Admin attempts to set `is_default = true` in the create endpoint? → Creation endpoints must reject attempts to set `is_default`; only bootstrap/migration may set it (`FORBIDDEN_FIELD`).
* Name/slug normalization: whitespace/diacritics/uppercase must be normalized before uniqueness checks; collisions after normalization must be treated as conflicts.
* Creator is always added as LEAD automatically; system ensures at least one lead exists (the creator).
* Duplicate nicknames in the same list → silently deduplicated before processing.
* Creator's nickname in `initial_lead_nicknames` → silently ignored (already LEAD).
* Creator's nickname in `initial_member_nicknames` → reject with `DUPLICATE_NICKNAME_IN_LISTS`.
* Same nickname in both lists → reject with `DUPLICATE_NICKNAME_IN_LISTS`.
* Non-existent nicknames → reject with `INVALID_INITIAL_MEMBERS` showing which nicknames don't exist.
* Non-coach nicknames in `initial_lead_nicknames` → reject with `INVALID_LEAD_ASSIGNMENT` or include in `INVALID_INITIAL_MEMBERS` details.
* All nickname validations happen atomically before any group creation.

## Requirements *(mandatory)*

### Functional Requirements

* **FR-001**: System MUST allow Admins and Coaches to create Groups.
* **FR-002**: System MUST reject group creation attempts by non-Coach, non-Admin users with 403 (`INSUFFICIENT_PERMISSIONS`).
* **FR-003**: When a group is created, the creator MUST be added as a Group member and assigned role `LEAD` for that group.
* **FR-004**: Groups MUST have attributes: `id`, `name`, `slug`, `description`, `visibility` (`VISIBLE` | `NOT_VISIBLE`), `join_policy` (`INVITE` | `REQUEST` | `OPEN`), `created_by`, `created_at`, `is_default` (boolean), `members_count`.
* **FR-005**: System MUST record membership entries (`GroupMember`) for the creator with `role = LEAD` and `joined_at` timestamp.
* **FR-006**: System MUST enforce that only Admins and Coaches (system-level roles) can be assigned the LEAD role inside a group.
* **FR-007**: System MUST persist groups with a unique `slug`. Attempts to create with an existing slug MUST fail (409 `SLUG_ALREADY_EXISTS`).
* **FR-008**: System MUST allow optional `initial_member_nicknames` and `initial_lead_nicknames` in request payload. When provided, all nicknames MUST be validated to exist and initial_lead nicknames MUST be validated as Coaches or Admin.
* **FR-009**: System MUST store `visibility` and `join_policy` and enforce these rules at runtime (discovery, join flows).
* **FR-010**: System MUST record `joined_at` for every initial added member and for the creator when group is created.
* **FR-011**: System MUST forbid creation of `is_default = true` via regular CreateGroup endpoint. The bootstrap/migration code is responsible for default/global group creation.
* **FR-012**: System MUST prevent creation of groups with zero members. (Creator is always added; attempts to explicitly clear members on create are invalid.)
* **FR-013**: System MUST validate combinations of `visibility` and `join_policy`. Groups with `visibility = NOT_VISIBLE` MUST only allow `join_policy = INVITE`.
* **FR-014**: System MUST record `created_by` with the creator user id.
* **FR-015**: System MUST return meaningful error codes/messages for invalid inputs (e.g., `RESERVED_NAME`, `SLUG_ALREADY_EXISTS`, `INVALID_LEAD_ASSIGNMENT`, `INVALID_POLICY_COMBINATION`, `FORBIDDEN_FIELD`, `DUPLICATE_NICKNAME_IN_LISTS`, `INVALID_INITIAL_MEMBERS`).
* **FR-016**: System MUST capture critical group creation events in audit logs (who created, timestamp, group configuration).
* **FR-017**: System MUST ensure only Admins and Coaches can be assigned Lead role.
* **FR-018**: System MUST create a default/global Group during bootstrap that contains all users and cannot be deleted.
* **FR-019**: System MUST ensure groups have at least one lead member at all times.
* **FR-020**: System MUST make all membership changes auditable with timestamps and role information.
* **FR-021**: System MUST validate all nicknames in `initial_member_nicknames` and `initial_lead_nicknames` exist before creating group.
* **FR-022**: System MUST reject group creation if same nickname appears in both `initial_lead_nicknames` and `initial_member_nicknames` with error `DUPLICATE_NICKNAME_IN_LISTS`.
* **FR-023**: System MUST silently deduplicate nicknames within the same list (either leads or members).
* **FR-024**: System MUST automatically add creator as LEAD; if creator's nickname appears in `initial_lead_nicknames`, it is silently ignored.
* **FR-025**: System MUST reject group creation if creator's nickname appears in `initial_member_nicknames` with error `DUPLICATE_NICKNAME_IN_LISTS`.
* **FR-026**: System MUST perform all nickname validations atomically before persisting any group data.
* **FR-027**: System MUST return detailed error information when multiple validation errors exist, including which nicknames were not found and which don't have appropriate roles.
* **FR-028**: System MUST verify user has Admin role (system-level) OR is a Lead (group-level) when checking permissions for group management operations.

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
    * `role` (enum: `LEAD`, `MEMBER`)
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

* **SC-001**: 100% of CreateGroup requests from valid Admins/Coaches result in a persistent Group record with creator assigned as LEAD (tested via automated integration tests).
* **SC-002**: Attempted group creation by non-authorized users is rejected with 403 in all tested cases.
* **SC-003**: The system enforces slug uniqueness: attempts to create duplicate slugs fail with 409 (`SLUG_ALREADY_EXISTS`).
* **SC-004**: The system enforces LEAD assignment rules: only Coaches/Admins can be assigned as group LEAD; invalid assignments rejected with 400.
* **SC-005**: The bootstrap process guarantees existence of the global default group upon system start (test: after migration/bootstrap, the `global` group exists and contains all users; cannot be deleted).
* **SC-006**: Visibility + join_policy rules are enforced: `NOT_VISIBLE` + `OPEN` combination is rejected at create time; `VISIBLE` groups allow non-member read-only discovery.
* **SC-007**: For created groups, `joined_at` timestamps are present for the creator and any provided initial members by nickname (verified by unit tests).
* **SC-008**: Deleting a non-global group triggers hard delete and removes associated contests and materials (integration test coverage).

## API Contract

### POST /api/groups

Create a new group with optional initial members and leads.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Coach or Admin authentication |

**Request Body**:
```json
{
  "name": "string",
  "description": "string",
  "visibility": "string",
  "join_policy": "string",
  "initial_lead_nicknames": ["string"],
  "initial_member_nicknames": ["string"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Name of the group |
| description | string | No | Description of the group |
| visibility | string | Yes | Group visibility: `VISIBLE` or `NOT_VISIBLE` |
| join_policy | string | Yes | Join policy: `INVITE`, `REQUEST`, or `OPEN` |
| initial_lead_nicknames | array[string] | No | Nicknames of users to add as leads (must be Coaches or Admin) |
| initial_member_nicknames | array[string] | No | Nicknames of users to add as members |

**Responses**:

#### 201 Created
Group created successfully.

```json
{
  "id": "string",
  "name": "string",
  "slug": "string",
  "description": "string",
  "visibility": "string",
  "join_policy": "string",
  "created_by": "string",
  "created_at": "string",
  "members_count": 0
}
```

#### 400 Bad Request
Validation error in the request.

```json
{
  "error": "INVALID_LEAD_ASSIGNMENT",
  "message": "Non-coach user cannot be assigned as lead"
}
```

```json
{
  "error": "DUPLICATE_NICKNAME_IN_LISTS",
  "message": "Nickname appears in both lead and member lists"
}
```

```json
{
  "error": "INVALID_INITIAL_MEMBERS",
  "message": "Validation errors in initial members",
  "details": {
    "not_found": ["nickname1", "nickname2"],
    "invalid_lead_role": ["contestant_nickname"]
  }
}
```

```json
{
  "error": "INVALID_POLICY_COMBINATION",
  "message": "Visibility and join policy combination is not allowed"
}
```

```json
{
  "error": "RESERVED_NAME",
  "message": "The group name is reserved"
}
```

```json
{
  "error": "FORBIDDEN_FIELD",
  "message": "Field cannot be set through this endpoint"
}
```

#### 403 Forbidden
User does not have permission to create groups.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only Coaches and Admins can create groups"
}
```

#### 409 Conflict
Group slug already exists.

```json
{
  "error": "SLUG_ALREADY_EXISTS",
  "message": "A group with this slug already exists"
}
```

## Notes / Implementation hints

* Slug generation must be deterministic and normalized (lowercase, remove punctuation, collapse spaces, strip diacritics).
* Enforce uniqueness at DB level (unique index on `slug`) and in application logic to return user-friendly errors.
* `is_default` must not be writable by regular endpoints — only migration/bootstrap scripts should set it.
* When `initial_member_nicknames` or `initial_lead_nicknames` are provided, treat group creation as atomic (either all validations pass and all members added, or reject with detailed errors).
* All nickname validations must happen before any database writes: validate existence, validate roles for leads, check for duplicates across lists.
* Duplicates within the same list are silently deduplicated before processing (e.g., `["coach_a", "coach_a"]` becomes `["coach_a"]`).
* Creator is always added as LEAD automatically; if creator's nickname is in `initial_lead_nicknames`, ignore it; if in `initial_member_nicknames`, reject.
* **Admin (system-level role) has implicit permissions** to perform all group management operations without being explicitly added as a member. Permission checks should verify: user is Lead of the group OR user has Admin role.
* Audit log entry should include `created_by`, `payload summary`, and `request origin`.
* Ensure integration tests simulate the "global group" semantics: after new user creation, the user is auto-added to the global group.
* Do not allow `visibility = NOT_VISIBLE` with `join_policy = OPEN` or `join_policy = REQUEST`; validate and reject with `INVALID_POLICY_COMBINATION`.
* Ensure membership events record `joined_at` and creator is always added as `LEAD`.
* Hard delete: when a group is deleted (non-global), delete associated contests and materials; keep referenced problem entities intact.
* Account anonymization (user deletion) is governed by `selfdeactivate account` spec — deleted/anonimized users must remain referenced in historical records (submissions, group membership history) with anonymized display fields.
* The create endpoint must reject attempts to set `is_default`, `members_count` or other computed/guarded fields in request payload (`FORBIDDEN_FIELD`).
* Problems are global entities and may be referenced by contests inside any group; creating a group does not duplicate problem entities.
* Deleting a group results in hard deletion of that group's contests and materials; problem entities referenced by those contests remain in the system.
* Data model should scale to groups with large numbers of members without any limits.
* Nickname validation should use the `nickname` field from User entity (must be unique across system).
* Error responses for invalid nicknames should be comprehensive and list all problems found in a single response.

---
