# Contest Management - Business Logic & Design

**Created**: 2026-01-03

This document centralizes the complete business logic of the Contest Management system and design considerations being applied across all related specs.

---

## 🔹 General Concept

* **`Contest`** is a programming competition that belongs to a Group.
* Contests have a defined **start time** and **end time**.
* Contests contain **problems** that participants must solve.
* **Standings** track participant rankings during and after the contest.
* Contests can be **global** (in the global group) or **group-specific**.

---

## 🔹 Contest States

Contest status is **computed** based on current time, not stored:

| Status | Condition | Description |
|--------|-----------|-------------|
| `SCHEDULED` | currentTime < startTime | Contest hasn't started yet |
| `ACTIVE` | startTime <= currentTime <= endTime | Contest is running |
| `FINISHED` | currentTime > endTime | Contest has ended |

### State Transitions

```
SCHEDULED ──(startTime reached)──> ACTIVE ──(endTime reached)──> FINISHED
```

---

## 🔹 Roles and Permissions

### Contest Creation

| Role | Regular Group | Global Group |
|------|--------------|--------------|
| **Admin** | ✅ Any group | ✅ (as Lead of global group) |
| **Coach** | ✅ Only if Lead | ✅ Only if Lead of global group |
| **Contestant** | ❌ | ❌ |

### Contest Management (Update, Delete, Manage Problems)

* **Contest Owner**: Can lock/unlock their contest
* **Group Lead**: Can manage contests in their group (all Leads can update and delete, including in global group)
* **Admin**: Can manage any contest (can lock/unlock and delete any contest, has implicit Lead permissions)
* **Locked Contests**: Cannot be modified or deleted except by owner or admin (who can lock/unlock)

---

## 🔹 Contest Configuration

### Basic Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Contest name (max 200 chars, mutable) |
| `description` | string | No | Contest description (max 5000 chars, mutable) |
| `startTime` | timestamp | Yes | When the contest starts (UTC, mutable, must be in future) |
| `endTime` | timestamp | Yes | When the contest ends (UTC, mutable, must be in future, cannot modify if postcompetition active) |
| `penalty` | integer | No | Penalty in minutes per wrong submission (default: 20, range: 0-1440, mutable) |
| `enablePostContest` | boolean | No | Enable post-competition phase (default: false, mutable) |
| `locked` | boolean | No | Lock status (default: false, mutable by owner/admin only) |
| `group_id` | UUID | Yes | Group identifier (immutable) |
| `owner_id` | UUID | Yes | Contest creator identifier (immutable) |
| `createdAt` | timestamp | Yes | Creation timestamp (immutable) |
| `updatedAt` | timestamp | No | Last update timestamp (updated on modification) |

### Scoring

* **Penalty-based scoring** (ICPC style):
  * Each wrong submission adds `penalty` minutes to solve time
  * Ranking by: problems solved (desc), then total time (asc)
* Default penalty: 20 minutes
* Penalty range: 0-1440 minutes (0 to 24 hours)

---

## 🔹 Problems in Contests

### Adding/Removing/Reordering Problems

* Problems can be added at contest creation or later via updates
* Problems can be removed from contests
* When a problem is removed, submissions to that problem have `contest_id` set to `null` (no longer associated with the contest)
* Problems can be reordered within contests
* Problems must be in `PUBLISHED` status
* **Accessibility rules**:
  * `PUBLIC` problems: any authorized user can add
  * `PRIVATE` problems: only problem modifiers can add
* **Standing recalculation**:
  * Adding problems does NOT trigger Standing recalculation (no submissions exist yet)
  * Removing problems during ACTIVE contests triggers Standing recalculation

### Problem Visibility During Contest

* **Before contest starts**: Problems are hidden from participants
* **During contest**: Problems are visible to registered participants
* **After contest**: Problems remain visible

### Problem Order

* Problems are assigned sequential order (1, 2, 3, ...)
* Order determines display sequence in the contest

---

## 🔹 Participants

### Participation Modes

Contests support different participation configurations:

| Mode | Description |
|------|-------------|
| `INDIVIDUAL` | Only individual users can register (default) |
| `TEAM` | Only teams can register |
| `MIXED` | Both individuals and teams can register |

### Team Configuration (for TEAM/MIXED modes)

| Attribute | Type | Description |
|-----------|------|-------------|
| `teamSizeMin` | integer | Minimum selectedMembers required (nullable) |
| `teamSizeMax` | integer | Maximum selectedMembers allowed (nullable) |
| `showTeamMembers` | boolean | Show member names in standings (default: false) |

### Individual Registration

* Participants must register before the contest starts (status SCHEDULED)
* Only Members can register (Leads and Admins cannot register)
* Registration is idempotent (registering twice returns success)
* Participants can unregister before contest starts
* Cannot unregister once contest has started (ACTIVE or FINISHED)
* If user is removed from group before contest starts: automatic unregistration
* If user is removed from group after contest starts: remains registered but cannot access
* No limit on number of participants

### Team Registration

* See [Team management specs](../Team%20management/README.md) for full details
* Teams register via separate endpoint with selectedMembers
* All selectedMembers must be group members (if contest belongs to a group)
* A user can only participate ONCE per contest (individually OR as part of one team)

### Submissions

* Participants can submit solutions during `ACTIVE` status
* If `enablePostContest = true`, registered users can also submit after `endTime` (postcompetition phase)
* Submissions are linked to both the contest and the problem
* For team submissions, the system automatically resolves team attribution based on selectedMembers lookup
* Each submission is judged against the problem's test cases
* Submissions during postcompetition (`submittedAt > endTime`) do NOT affect standings

---

## 🔹 Postcompetition Phase

### Concept

* **Postcompetition** is an optional phase that allows registered users to continue submitting after the contest ends
* Submissions during postcompetition are judged normally but do NOT affect standings
* Standing is frozen at `endTime` and remains visible but immutable during postcompetition

### Activation

* Postcompetition is controlled by the `enablePostContest` field (default: false)
* Can be enabled before contest ends: Will start automatically after `endTime`
* Can be enabled after contest ends: Starts immediately when enabled
* Once enabled, postcompetition continues indefinitely (until disabled)

### Behavior

* **Submissions**: Registered users can submit solutions during postcompetition
* **Standing**: Frozen at `endTime`, visible but immutable
* **Identification**: Submissions during postcompetition are identified by `submittedAt > endTime` and having `contest_id`
* **Judging**: Submissions are judged normally against problem test cases
* **Standing Impact**: Submissions do NOT affect standings (standing remains frozen)

### Restrictions

* **Cannot disable**: Postcompetition cannot be disabled (`enablePostContest` from true to false) if submissions exist after `endTime`
* **Cannot modify endTime**: When postcompetition is active (submissions exist after `endTime`), `endTime` cannot be extended or reduced
* **Can reduce endTime**: Only if no postcompetition submissions exist, and only to time >= last submission before original `endTime`

---

## 🔹 Standings

### Data Architecture

* **NoSQL Document Storage**: Registration and Standing data are stored together in NoSQL documents
* **One Collection Per Contest**: Each contest has its own collection `contest_{contestId}_standings` for optimal performance
* **Document Structure**: Each document contains registration info and standing data for one participant
* **Final Snapshot**: When contest ends, a snapshot collection `contest_{contestId}_standings_final` is created for historical records

### Calculation

* **ICPC-style ranking**:
  1. Primary: Number of problems solved (descending)
  2. Secondary: Total time including penalties (ascending)
* Time is calculated from contest start to accepted submission
* Penalties added for each wrong submission before acceptance
* **Penalty calculation**: Only added when problem is first accepted = (attempts - 1) * contestPenalty

### Updates

* Standings are updated in real-time during `ACTIVE` contests using atomic operations
* Standings are **frozen** at `endTime` (when contest ends)
* Standings remain visible but immutable during postcompetition phase
* Standings are recalculated when:
  * Problems are removed during ACTIVE contests (submissions to removed problems have `contest_id` set to `null`)
  * Penalty is changed during ACTIVE contests
* When a problem is added to a contest, standings are NOT recalculated (no submissions exist yet)
* Submissions during postcompetition do NOT affect standings (standing is frozen at endTime)
* If a contest is deleted, the standings collection is deleted (final snapshot also deleted if exists, submissions preserved)

### Freeze (Optional)

* **`freezeMinutes`** (integer, nullable, default: 60) - Minutes before `endTime` to freeze standings
* When `freezeMinutes` is set, standings freeze at `endTime - freezeMinutes`
* During freeze:
  * Regular users see submissions after freeze time as "pending" (?)
  * Leads and Admin can view real-time standings with `?realtime=true`
* When `freezeMinutes = null`, no freeze is applied
* After contest ends, full standings are revealed (freeze lifted)

### Real-Time Updates (SSE)

* **Server-Sent Events** endpoint: `GET /contests/{id}/standings/stream`
* Sends incremental updates when standings change
* Event types: `snapshot`, `update`, `freeze`, `contest_ended`, `ping`
* Only available for ACTIVE contests
* Respects freeze rules (unless Leads/Admin with `?realtime=true`)

---

## 🔹 Relationship with Groups

### Regular Groups

* Contest belongs to exactly one group
* Only Leads of the group can create contests
* Group members are potential participants

### Global Group

* Special group accessible to all users
* Only Leads of the global group can create contests (Admin and any Coaches added as leads)
* Admin is automatically lead of the global group and can add other leads
* All platform users can participate

---

## 🔹 Content Deletion

### When a Contest is Deleted

| Entity | Action |
|--------|--------|
| Contest | Hard delete |
| Contest_Problem | Hard delete (cascade) |
| Standing Collection | Hard delete (`contest_{contestId}_standings`) |
| Final Snapshot | Deleted (`contest_{contestId}_standings_final` deleted if exists) |
| Submission | Preserved with `contest_id = NULL` (orphaned) |
| Problem | NOT deleted (global entities) |

> **Note**: Both the NoSQL collection `contest_{contestId}_standings` and the final snapshot `contest_{contestId}_standings_final` are deleted when the contest is deleted (if they exist).

### When the Parent Group is Deleted

* All contests in the group are deleted (cascade)
* Same deletion rules apply as above

---

## 🔹 Technical Considerations

### Time Handling

* All times stored in **UTC**
* Client responsible for timezone conversion
* Status computed on each request based on server time
* `startTime` and `endTime` must always be in the future (cannot be set to past)
* `endTime` must be after `startTime`
* Times can be modified even after contest has started (but must remain in future)
* `endTime` cannot be modified when postcompetition is active (submissions exist after endTime)
* `endTime` can only be reduced to time >= last submission before original endTime

### Concurrency

* Optimistic locking for contest updates
* Atomic operations for standings updates
* Handle race conditions in registration

### Performance

* Index on `group_id` for contest queries
* Index on `contest_id` for problem and submission queries
* Pagination for standings with many participants

---

## 🔹 Related Specs

### Implemented Specs

1. **[Create contest](Create%20contest/spec.md)** - Contest creation with initial problems
2. **[Update contest](Update%20contest/spec.md)** - Modify contest details, problems, times, penalty, and lock/unlock functionality
3. **[Delete contest](Delete%20contest/spec.md)** - Remove contest and handle associated data (Contest_Problem, Standing, Register deleted; submissions orphaned)
4. **[Register to contest](Register%20to%20contest/spec.md)** - Participant registration and unregistration (Members only, before contest starts)
5. **[View contest](View%20contest/spec.md)** - Contest details, problem list, and contest listing with filters
6. **[View contest standings](View%20contest%20standings/spec.md)** - ICPC-style rankings with freeze support and SSE real-time updates

### Future Specs (Planned)

* **View postcompetition progress** (`GET /contests/{id}/postcompetition`)
  - Shows per-user progress during postcompetition
  - Which problems each user is attempting/has solved after contest ended
  - Does NOT affect official standings

* **View contest submissions** (`GET /contests/{id}/submissions`)
  - Filter by `phase`: `competition` | `postcompetition` | `all`
  - Shows submission list with verdict, time, memory

* **Export standings** (`GET /contests/{id}/standings/export`)
  - CSV, PDF, Excel formats
  - For official records and certificates

### Future Frontend Features

* **Standing Freeze Visualizer**
  - Animated "unfreeze" replay after contest ends
  - Shows how standings evolved during freeze period
  - Data already available in `contest_{contestId}_standings_final` with full history

### Implementation Dependencies

```
Create Contest (base)
    ↓
Update Contest (P1) ✅ ← (modify details, problems, times, penalty, lock/unlock)
    ↓
Delete Contest (P1) ✅ ← (remove contest, handle associated data)
    ↓
Register to Contest (P1) ✅ ← (participant registration and unregistration)
    ↓
View Contest (P1) ✅ ← (contest details, problem list, listing)
    ↓
View Contest Standings (P1) ✅ ← (ICPC rankings, freeze, SSE)
    ↓
View Postcompetition Progress (P2) ← (per-user progress after contest)
    ↓
View Contest Submissions (P2) ← (submission list, filters)
```

> **Note**: Update Contest includes functionality for managing problems (add/remove/reorder), so a separate "Manage Contest Problems" spec is not needed.

---

## 🔹 Key Design Decisions

### Why computed status instead of stored?

* **Accuracy**: Status is always correct based on real time
* **Simplicity**: No background jobs to update status
* **Consistency**: No risk of stale status values

### Why penalty-based scoring as default?

* **Industry standard**: ICPC uses this format
* **Simplicity**: Easy to understand and implement
* **Extensibility**: Can add other modes later (IOI, custom)

### Why problem accessibility validation?

* **Control**: Problem creators control where their problems appear
* **Quality**: Prevents unauthorized use of private problems
* **Flexibility**: PUBLIC problems can be freely shared

### Why allow Admin to create contests anywhere?

* **Administrative needs**: Platform management and support
* **Emergency**: Can assist groups when needed
* **Consistency**: Follows same pattern as other admin permissions

### Why allow Admin to lock/unlock contests?

* **Administrative override**: Admins need to fix issues when contest owners are unavailable
* **Support**: Can assist groups when contest owners need help
* **Consistency**: Follows same pattern as other admin override capabilities
* **Protection**: Admins can lock contests to prevent accidental modifications

### Why allow updates during active contests?

* **Flexibility**: Contest organizers need to adapt to changing circumstances
* **Error correction**: Allows fixing mistakes discovered during the contest
* **Problem management**: Enables adding/removing problems as needed
* **Time adjustments**: Allows extending or reducing contest duration

### Why lock contests?

* **Data integrity**: Prevents accidental modifications after contest ends
* **Historical accuracy**: Preserves contest state for historical records
* **Fairness**: Ensures standings remain unchanged after contest completion
* **Control**: Contest owners can lock contests when they're satisfied with the state

### Why orphan submissions instead of deleting them?

* **Data preservation**: Submissions represent user work and should be preserved
* **Analytics**: Orphaned submissions can still be analyzed for problem statistics
* **User history**: Users can see their submission history even if contest is deleted
* **Problem statistics**: Problems maintain their submission statistics across contests

### Why can't locked contests be deleted?

* **Safety**: Prevents accidental deletion of important contests
* **Explicit action**: Requires intentional unlock before deletion
* **Audit trail**: Ensures deletion is a deliberate action, not accidental

### Why postcompetition phase?

* **Learning opportunity**: Allows participants to continue practicing after contest ends
* **Fairness**: Standing is frozen at endTime, ensuring final rankings are preserved
* **Flexibility**: Contest organizers can enable/disable based on their needs
* **Data preservation**: Submissions during postcompetition are preserved for learning purposes
* **No standing impact**: Ensures contest integrity - only submissions during official contest time affect rankings

---

*This document should be updated when new design decisions are made or additional specs are implemented.*

