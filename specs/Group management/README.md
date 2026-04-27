# Group Management - Business Logic & Design

**Created**: 2025-12-28

This document centralizes the complete business logic of the Group Management system and design considerations being applied across all related specs.

---

## 🔹 General Concept

* **`Group`** is a root entity.
* **No hierarchy** between groups (flat structure).
* There is a **global group** that contains all users automatically.
* Groups can be **visible or not visible**.
* Groups can define **how users join**:
  * By **invitation** (only leads invite) - **required for non-visible groups**
  * By **request** (users request, leads approve) - **only for visible groups**
  * **Open** (no approval, automatic entry) - **only for visible groups**
* **User identification**: The system uses **nicknames** (unique) to identify users in group operations (creation, invitations, member management).

---

## 🔹 Roles Within the Group

### Available Roles
* Only **two roles** exist:
  * **Lead** (can only be coaches or admin at system level)
  * **Member** (any user)

### Permissions and Restrictions
* **Admin has implicit global permissions** on all groups without explicit membership.
* A **coach** can be lead of one or several groups.
* A **user** can belong to multiple groups.
* Only **coaches** and **admins** can be assigned as **Lead** in a group.
* **Admin** can manage any group without needing to be added as a member.

---

## 🔹 Access and Visibility

| Group State       | Non-member User                 | Allowed Join Policies           |
| ----------------- | ------------------------------- | ------------------------------- |
| **Visible**       | Can see everything in read mode | `INVITE`, `REQUEST`, `OPEN`     |
| **Not Visible**   | Cannot see anything             | Only `INVITE`                   |

### Interaction Rules
* To **interact** (submissions, management, etc.) you must be a **member**.
* To **administer**, you must be a **group lead**.
* Non-members of visible groups can browse content but not participate.
* **Important restriction**: Non-visible groups can only use `INVITE` policy (no point in `OPEN` or `REQUEST` if no one can see the group).

---

## 🔹 Joining the Group

### Join Methods
* **Invitations** and **requests** can only be managed by leads.
* **Open entry** → user automatically enters as member.
* Join event is recorded (e.g. `joined_at`, join method).
* A lead can **change the join mode** at any time.

### Specific Flows
1. **By Invitation (`INVITE`)**:
   - Lead creates invitation generating JWT token with 3-day TTL
   - Can invite by **nickname**, email or user ID (resolved to user_id at creation)
   - System sends email with URL containing the JWT token
   - User accepts by clicking the URL in the email
   - If the same user is re-invited, the previous record is deleted
   - URL format: `https://training-center.com/groups/{groupId}/accept?token={jwt_token}`

2. **By Request (`REQUEST`)**:
   - User creates join request
   - Lead approves/rejects the request
   - **Only works on visible groups** (policy restriction)

3. **Open Entry (`OPEN`)**:
   - User joins directly without approval
   - **Only works on visible groups** (policy restriction)
   - Immediate entry as member

---

## 🔹 Group Content

### Associated Entities
* **Contests** and **materials** **belong to the group**.
* **Problems** are **global and reusable** entities.
* The **global group** can have public materials and contests.

### Content Deletion
* If a **group is deleted**:
  * **Contests** are deleted (hard delete)
  * **Materials** are deleted (hard delete) - including all associated URLs and tags
  * **Standings** are deleted (hard delete) - NoSQL collections `contest_{contestId}_standings` and `contest_{contestId}_standings_final` are deleted for each contest
  * **Submissions are preserved** with `contest_id = NULL` (orphaned but remain in user history)
  * **Problems continue to exist** (they are global)
  * **Memberships, invitations, and join requests** are deleted

---

## 🔹 Deletion and Persistence

### Group Deletion
* **Group deletion**: **hard delete**.
* **Exception**: The global group **cannot be deleted**.

### User Management
* **Deleted users** from the system → are **anonymized** (according to user-management spec).
* **Historical references are preserved** (submissions, membership history).
* Anonymous data maintains referential integrity.

### Membership Rules
* **Global group**: Users **cannot leave** the global group.
* **Other groups**: Users can leave voluntarily.
* **Last lead**: The last lead of a group cannot be removed.
* **Admin in global group**: Admin cannot be removed as lead of the global group (even by other leads).

---

## 🔹 Admin - Implicit Global Permissions

### Fundamental Rule
* **Admin (system role) has implicit permissions** on ALL groups without requiring explicit membership.
* Admin can perform any group management operation without being added as member or lead.
* No registration in `GroupMember` table required - permissions are at system role level.

### Permission Verification
**Authorization logic** for group operations:
```
User authorized IF:
  - User has ADMIN role (system) 
  OR
  - User is LEAD of the group (explicit membership)
```

### Implementation
* **Verification at each endpoint**: Before any management operation, verify system role.
* **No membership registration**: Admin does NOT appear in `GroupMember` of any group.
* **Full permissions**: Admin can create, modify, delete groups and manage members.
* **No restrictions**: Admin can operate on visible and non-visible groups equally.

### Allowed Operations
Admin can perform:
* ✅ Create groups
* ✅ Modify configuration of any group (name, visibility, policies) - **except global group**
* ✅ Add/remove members from any group - **except global group membership**
* ✅ Change member roles (Member ↔ Lead) - **except cannot remove self as lead from global group**
* ✅ View pending invitations of any group
* ✅ Create invitations for any group
* ✅ Delete groups (except global group)
* ✅ Manage group content (contests, materials)
* ✅ Add other leads to global group (Coaches or Admins only)

### UI Considerations
* **Member listings**: Admin does NOT appear in member/lead lists (has no membership).
* **Permission indicator**: UI can show "Access as Admin" message when Admin manages a group.
* **No self-assignment**: Admin does NOT need to add themselves as lead to manage groups.

### Purpose
* **Global supervision**: Admin has complete visibility and control of the system.
* **Emergency management**: Admin can intervene in any group immediately.
* **Separation of concerns**: System role ≠ Group role.
* **Scalability**: Does not pollute membership tables with unnecessary records.

---

## 🔹 Global Group (Default Group)

### Special Characteristics
* **Automatically created** during system bootstrap.
* **All users** are members automatically.
* **Cannot be deleted** or have its membership modified manually.
* **Admin** is lead of the global group (explicit membership in this special case).
* **Admin cannot be removed as lead** of the global group.
* **Admin can add other leads** to the global group (must be Coaches or Admins).
* **Other leads of the global group** have full permissions to manage the group and create contests, but cannot remove Admin as lead.
* Marked with `is_default = true`.
* **Users can "hide" the global group** from their personal view (soft hide) without affecting actual membership.

### Purpose
* Container for public contests and materials.
* Common reference point for all users.
* Facilitates management of global system content.
* **Can be hidden from the UI** for users who prefer to see only their specific groups.

---

## 🔹 Technical Considerations

### User Identification
* **Nicknames as identifiers**: All group operations use nicknames instead of UUIDs for better user experience.
* **Unique nicknames**: The `nickname` field in the User entity is unique at system level.
* **Nickname validation**: System validates nickname existence before critical operations.
* **Automatic resolution**: Nicknames are resolved to user_id internally to maintain referential integrity.

### Security
* **Invitation tokens**: JWT with signed payload containing user_id, group_id and expiry.
* **Fixed 3-day TTL**: All invitations automatically expire after 3 days.
* **JWT validation**: Signature, expiry and payload verification before accepting invitation.
* **Single-use tokens**: Record is deleted when accepted or when re-inviting.
* **Invitations by nickname or email**: Resolved to user_id at creation time.
* **Email sending**: URL with JWT token sent to user's registered email.

### Concurrency
* **Transactions** for critical membership operations.
* **DB constraints** to prevent inconsistent states.
* **Atomic validation** for rules like "last lead".

### Selective Auditing
* **Only critical changes** are logged to reduce overhead:
  * Group creation/deletion
  * Role changes (Member ↔ Lead)
  * Member addition/removal by lead
  * Group policy changes (visibility, join policy)
* **Not audited** minor operations such as:
  * Invitation acceptance (already recorded in `joined_at`)
  * Member or invitation listings
  * Read-only operations

### Scalability
* **No limits** on number of members per group.
* **Appropriate indexes** for membership queries.
* **Pagination** in member and invitation listings.

### User Preferences
* **Storage**: User preferences (like `hideGlobalGroup`) are stored in a JSON field in the `User` table.
* **Field name**: `preferences` (json type)
* **Example structure**: `{"hideGlobalGroup": false, "theme": "dark", "language": "en"}`
* **Flexibility**: JSON format allows adding new preferences without schema changes.
* **Default values**: System provides sensible defaults when preference is not set.
* **Scope**: Preferences are user-specific and affect only their personal view/experience.

> **Note**: The `hideGlobalGroup` preference only affects the `GET /users/me/groups` endpoint (My Groups dashboard). The global group still appears in `GET /groups` (list all groups) and can be accessed via `GET /groups/{id}` normally.

---

## 🔹 Related Specs

### Implemented Specs
1. **[Create group](Create%20group/spec.md)** - Group creation with initial configuration
2. **[Update group](Update%20group/spec.md)** - Update group metadata and policies (with automatic handling of pending requests)
3. **[Delete group](Delete%20group/spec.md)** - Safe deletion with confirmation and associated content handling
4. **[Join group](Join%20group/spec.md)** - User perspective: direct join (OPEN), request-to-join (REQUEST), invitation acceptance (INVITE), view/cancel own request
5. **[Manage join requests](Manage%20join%20requests/spec.md)** - Lead perspective: view, approve, and reject join requests
6. **[Invite to group](Invite%20to%20group/spec.md)** - Invitation system with JWT tokens
7. **[Manage group members](Manage%20group%20members/spec.md)** - Administrative membership management
8. **[View groups](View%20groups/spec.md)** - List groups, search, filter, and "My Groups" dashboard with user preferences

### Implementation Dependencies
```
Create Group (base)
    ↓
Update Group (P1) ← (modify metadata and policies)
    ↓
Delete Group (P1) ← (requires confirmation, handles associated content)
    ↓
Join Group (P1) ← Invite to Group (P2)
    ↓                    ↓
Manage Join Requests (P1) ← (completes REQUEST flow)
    ↓
Manage Group Members (P2-P3)
    ↓
View Groups (P2) ← (discovery, search, dashboard)
```

### Future Specs (Considered)
* **Group Analytics** - Participation and activity metrics
* **Bulk Operations** - Mass management of members and invitations
* **Membership Notifications** - Email/in-app alerts for membership events (removed, role changed, invited). Adding this feature is also the trigger to migrate audit writes to a synchronous in-process event bus — see *Key Design Decisions* for the planned migration path.

---

## 🔹 Final Confirmations

### Nomenclature
* **Final name**: **Group** (not "Team", "Organization", etc.)
* **Consistency** with the rest of the system in naming and structure.

### Project Alignment
* Group specs follow **exactly the same structure** as other specs.
* **No new sections invented** nor consistency broken.
* **Error codes** follow established conventions.

### 📋 Creation Spec Scope
The **Create Group** spec includes:
* ✅ Group creation with metadata
* ✅ Automatic assignment of creator as lead
* ✅ Policy configuration (visibility, join)
* ✅ Initial relationships: add members and leads by **nickname** optionally
* ✅ Business validations and restrictions:
  * Nickname existence validation
  * Role validation for leads (only Coaches/Admin)
  * Silent deduplication of nicknames in same list
  * Creator handling in lists (auto-lead, rejection if in members)
  * Detection of nicknames in both lists
* ✅ Detailed errors with complete problem information

---

## 🔹 Usability and Maintainability Improvements

### **"My Groups" Dashboard**
* **Centralized view** of all user's groups
* **Filters**: By role (Admin/Member), visibility, recent activity
* **Search**: By group name, description, tags
* **Visibility management**: Option to hide/show global group
* **Quick actions**: Leave group, view pending invitations

### **Improved Invitations**
* **By nickname**: Invite known users without needing to know their UUID (resolved to user_id at creation)
* **By email**: For registered users inviting by their email (resolved to user_id at creation)
* **JWT tokens**: Signed, with 3-day TTL, contain user_id and group_id in payload
* **Automatic sending**: Email with acceptance URL sent automatically
* **Re-invitation**: Deletes previous invitation and generates new token with new expiration
* **Preview**: Show group information before accepting
* **Nickname validation**: System verifies existence before creating invitation

### **Simplified Policies**
* **Reduced valid combinations**:
  * `VISIBLE + INVITE` ✅
  * `VISIBLE + REQUEST` ✅  
  * `VISIBLE + OPEN` ✅
  * `NOT_VISIBLE + INVITE` ✅
  * ~~`NOT_VISIBLE + REQUEST`~~ ❌ (makes no sense)
  * ~~`NOT_VISIBLE + OPEN`~~ ❌ (makes no sense)

### **Intelligent Auditing**
* **Logged**:
  * Group creation/deletion
  * Admin ↔ Member role changes
  * Manual member addition/removal
  * Group policy changes
* **Not logged**:
  * Invitation acceptance (already in `joined_at`)
  * Read-only queries
  * Automatic system operations

---

## 🔹 Key Design Decisions

### Why flat structure?
* **Simplicity**: Avoids complexity of hierarchical permissions
* **Flexibility**: Users can be in multiple independent groups
* **Scalability**: Easier to query and maintain

### Why only 2 roles?
* **Clarity**: Simple distinction between leaders and members
* **Sufficiency**: Covers all identified use cases
* **Extensibility**: Can be expanded in the future if needed

### Why mandatory global group?
* **Public content**: Place for contests/materials accessible to all
* **Common reference**: All users have at least one membership
* **Bootstrapping**: Facilitates system initialization

### Why policy restrictions by visibility?
* **Logic**: Makes no sense for a non-visible group to allow open entry (`OPEN`) or by request (`REQUEST`)
* **Simplicity**: Reduces combinations from 6 to 4 valid cases
* **Usability**: Avoids confusing configurations for group leaders

### Why JWT tokens with 3-day TTL?
* **Enhanced security**: Signed JWT prevents token manipulation
* **Self-contained**: Token includes user_id and group_id without DB query on first validation
* **Fixed TTL**: 3 days is sufficient for user to check email and accept, without prolonged risk
* **Operational simplicity**: No cleanup job required - JWT expires automatically
* **Industry standard**: JWT is widely supported and understood

### Why selective auditing?
* **Performance**: Significantly reduces log volume
* **Maintainability**: Less data to clean and manage
* **Focus**: Concentrates on changes that really matter for compliance and debugging

### Why direct audit writes instead of a domain event bus?

Current implementation writes to `group_membership_audit_log` directly inside the use case, within the same DB transaction as the membership change.

**Why not an event bus right now:**
* No side effects beyond the audit log exist yet — a bus adds abstraction without a second consumer to justify it
* Direct writes are synchronous, testable, and transactionally safe without extra infrastructure

**Path forward — when to migrate to Option B (synchronous in-process event bus):**

When membership notifications are added (e.g. "You were removed from group X", "Your role was changed to Lead"), the use case would have 2+ independent side effects (audit + email). At that point, refactoring to a synchronous event bus makes sense:

```
UseCase → eventBus.Publish(MemberRemoved{...})
              → AuditLogHandler    (saves audit log, same TX)
              → NotificationHandler (sends email, async or sync)
```

This requires no message broker — just in-process function dispatch. The migration is localized to the use case layer and does not affect the domain or infrastructure.

**If guaranteed delivery is ever needed** (e.g. notifications must not be lost even if the process crashes mid-operation), the right approach is the **Transactional Outbox Pattern** using Postgres itself — no external broker required. An `outbox_events` table is written in the same TX; a background worker processes and delivers the events.

### Why Admin has implicit permissions without membership?
* **Clean architecture**: Separates system permissions from group permissions
* **Clean membership table**: `GroupMember` only contains real user memberships
* **Clearer UI**: Users don't see "phantom admin" in their groups
* **Scalability**: Doesn't pollute tables with unnecessary admin records
* **Flexibility**: Facilitates adding more system roles in the future
* **Necessary supervision**: Admin requires immediate access without barriers

### Why option to hide global group?
* **Usability**: Advanced users can focus on their specific groups
* **Flexibility**: Maintains functionality without forcing visibility
* **Adoption**: Facilitates transition for users who prefer clean interfaces

---

*This document should be updated when new design decisions are made or additional specs are implemented.*
