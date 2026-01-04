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
| **Admin** | ✅ Any group | ✅ |
| **Coach** | ✅ Only if Lead | ✅ |
| **Contestant** | ❌ | ❌ |

### Contest Management (Update, Delete, Manage Problems)

* **Contest Owner**: Full control over their contest
* **Group Lead**: Can manage contests in their group
* **Admin**: Can manage any contest

---

## 🔹 Contest Configuration

### Basic Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Contest name (max 200 chars) |
| `description` | string | No | Contest description (max 5000 chars) |
| `startTime` | timestamp | Yes | When the contest starts (UTC) |
| `endTime` | timestamp | Yes | When the contest ends (UTC) |
| `penalty` | integer | No | Penalty in minutes per wrong submission (default: 20) |

### Scoring

* **Penalty-based scoring** (ICPC style):
  * Each wrong submission adds `penalty` minutes to solve time
  * Ranking by: problems solved (desc), then total time (asc)
* Default penalty: 20 minutes
* Penalty range: 0-1440 minutes (0 to 24 hours)

---

## 🔹 Problems in Contests

### Adding Problems

* Problems can be added at contest creation or later
* Problems must be in `PUBLISHED` status
* **Accessibility rules**:
  * `PUBLIC` problems: any authorized user can add
  * `PRIVATE` problems: only problem modifiers can add

### Problem Visibility During Contest

* **Before contest starts**: Problems are hidden from participants
* **During contest**: Problems are visible to registered participants
* **After contest**: Problems remain visible

### Problem Order

* Problems are assigned sequential order (1, 2, 3, ...)
* Order determines display sequence in the contest

---

## 🔹 Participants

### Registration (Future Spec)

* Participants must register before or during the contest
* Group members may have automatic registration (configurable)
* Global contests are open to all users

### Submissions

* Participants can submit solutions only during `ACTIVE` status
* Submissions are linked to both the contest and the problem
* Each submission is judged against the problem's test cases

---

## 🔹 Standings

### Calculation

* **ICPC-style ranking**:
  1. Primary: Number of problems solved (descending)
  2. Secondary: Total time including penalties (ascending)
* Time is calculated from contest start to accepted submission
* Penalties added for each wrong submission before acceptance

### Updates

* Standings are updated in real-time during `ACTIVE` contests
* Standings are **frozen** for `FINISHED` contests
* If a contest is deleted, standings are deleted (submissions preserved)

---

## 🔹 Relationship with Groups

### Regular Groups

* Contest belongs to exactly one group
* Only Leads of the group can create contests
* Group members are potential participants

### Global Group

* Special group accessible to all users
* Any Coach or Admin can create contests
* All platform users can participate

---

## 🔹 Content Deletion

### When a Contest is Deleted

| Entity | Action |
|--------|--------|
| Contest | Hard delete |
| Contest_Problem | Hard delete (problem links) |
| Standings | Hard delete |
| Submissions | Preserved with `contest_id = NULL` |
| Registrations | Hard delete |

### When the Parent Group is Deleted

* All contests in the group are deleted (cascade)
* Same deletion rules apply as above

---

## 🔹 Technical Considerations

### Time Handling

* All times stored in **UTC**
* Client responsible for timezone conversion
* Status computed on each request based on server time

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

### Future Specs (Planned)

* **Update contest** - Modify contest details before start
* **Delete contest** - Remove contest with content handling
* **Manage contest problems** - Add/remove problems
* **Register to contest** - Participant registration
* **View contest** - Contest details and problem list
* **Contest standings** - Rankings and leaderboard
* **Submit solution** - Problem submissions (may be separate module)

### Implementation Dependencies

```
Create Contest (base)
    ↓
Update Contest (P1) ← (modify before start)
    ↓
Manage Contest Problems (P1) ← (add/remove problems)
    ↓
Register to Contest (P1) ← (participant registration)
    ↓
View Contest (P2) ← (public information)
    ↓
Contest Standings (P2) ← (rankings)
    ↓
Delete Contest (P3)
```

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

---

*This document should be updated when new design decisions are made or additional specs are implemented.*

