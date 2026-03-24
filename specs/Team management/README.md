# Team Management - Business Logic & Design

**Created**: 2026-02-07

This document centralizes the complete business logic of the Team Management system and design considerations being applied across all related specs.

---

## 🔹 General Concept

* **`Team`** is a global, reusable entity independent of contests.
* Teams can have **unlimited members**, but only a **subset participates** in each contest.
* Teams **register to contests** similarly to how individual users register.
* A user can belong to **multiple teams** but can only compete **once per contest** (individually OR as part of one team).

---

## 🔹 Team Structure

### Core Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| id | UUID | Unique identifier |
| name | string | Team name (1-100 chars) |
| createdBy | User | User who created the team (no special privileges) |
| createdAt | timestamp | Creation time |

### No Hierarchy

* **No captain or leader** role within the team.
* Any member can:
  * Invite new members
  * Register the team to a contest
  * Select which members participate
  * Submit solutions (when part of selectedMembers)

---

## 🔹 Team Membership

### Member Lifecycle

```
User A creates Team X
       │
       ▼
User A invites User B ──► TeamInvitation created
       │                         │
       │                         ▼
       │                  User B accepts ──► TeamMember created
       │                         │
       │                         ▼
       │                  User B can leave at any time
       │
       ▼
User A is automatically a member (no invitation needed)
```

### Invitation Rules

* Any member can invite users to the team.
* Invitations require **user acceptance** before membership is granted.
* A user can be invited to multiple teams.
* A user can accept invitations to multiple teams.
* Duplicate invitations to the same user are rejected.

### Leaving a Team

* Any member can leave at any time.
* **Exception**: Cannot leave if currently selected for an **ACTIVE contest**.
* If user leaves while selected for a **SCHEDULED contest** → automatically removed from selectedMembers.

---

## 🔹 Contest Registration

### Registration Modes

Contests define their participation mode:

| Mode | Description |
|------|-------------|
| `INDIVIDUAL` | Only individual users can register |
| `TEAM` | Only teams can register |
| `MIXED` | Both individuals and teams can register |

### Team Size Configuration

For `TEAM` and `MIXED` modes:

```json
{
  "participationMode": "TEAM",
  "teamSize": {
    "min": 2,    // Minimum members required
    "max": 3     // Maximum members allowed (ICPC style)
  }
}
```

### Registration Process

```
Team X (members: [A, B, C, D, E]) registers to Contest Z
       │
       ▼
Select participating members: [A, B, C]  (max 3 for this contest)
       │
       ▼
Validations:
  ✓ A, B, C are all members of Team X
  ✓ A, B, C are not registered individually
  ✓ A, B, C are not in another team registered to Contest Z
  ✓ If Contest Z belongs to a Group → A, B, C are members of that Group
  ✓ selectedMembers count is within [teamSize.min, teamSize.max]
       │
       ▼
ContestTeamParticipant created
```

### Modifying Participation

| Contest Status | Can modify selectedMembers? |
|----------------|----------------------------|
| `SCHEDULED` | ✅ Yes |
| `ACTIVE` | ❌ No (locked at start) |
| `FINISHED` | ❌ No |

---

## 🔹 Submissions

### Server-Side Participant Resolution

When a user submits during a contest, the system **automatically resolves** their participation type:

```
User A submits to Contest Z
       │
       ▼
┌────────────────────────────────────────┐
│ Step 1: Is User A registered     │
│ individually? (fast O(1) lookup)  │
└───────────────────┴────────────────────┘
       │
       ├── YES → standingId = userId ✓
       │
       └── NO → Continue to Step 2
              │
              ▼
       ┌────────────────────────────────────────┐
       │ Step 2: Is User A in a team's    │
       │ selectedMembers? (slower lookup) │
       └───────────────────┴────────────────────┘
              │
              ├── YES → standingId = teamId ✓
              │
              └── NO → Reject (NOT_REGISTERED)
```

> **Why this order?** Individual lookup is O(1) with a primary key index. Team lookup requires scanning `selectedMembers` arrays, which is more expensive.

### Submission Endpoint

The endpoint remains the same for all users:

```
POST /api/groups/{groupId}/contests/{contestId}/problems/{problemSlug}/submissions
```

No team ID in the request - the system determines standing attribution automatically.

**Resolution order (performance optimized):**
1. Check individual registration (fast, O(1) primary key lookup)
2. Check team membership (slower, array contains lookup)

### Submission Storage

Submissions are **always linked to the user** who submitted (`submittedBy`):

```json
{
  "id": "submission-uuid",
  "problemId": "problem-456",
  "contestId": "contest-123",
  "submittedBy": "user-A",              // FK to User - who physically submitted
  "standingId": "team-X or user-A",     // Who gets credit in standings
  "language": "cpp20"
}
```

> **Important**: The submission is linked to the **user** via `submittedBy`. The `standingId` is only used to determine which standing document to update.

### Visibility Rules

* If contest belongs to a **group** → submission visibility follows group rules.
* Team members (not in selectedMembers) can view submissions if group allows.
* Non-group members cannot view submissions.

---

## 🔹 Standings

### Unified Standing Document

Standings use a **generic `id`** that can be either a userId or teamId:

```json
{
  "id": "team-X or user-A",           // Generic: userId OR teamId
  "type": "TEAM" | "INDIVIDUAL",       // Identifies the type
  "displayName": "Team Alpha",         // Team name OR user nickname
  "members": ["alice", "bob"],         // Only for TEAM (if showTeamMembers)
  "registeredAt": "2026-01-10T10:00:00Z",
  "problemsSolved": 3,
  "penalty": 125,
  "problems": {...},
  "lastUpdated": "..."
}
```

### Standing Update Flow

```
Submission judged → lookup standingId → update standing document
       │
       ├── standingId = teamId → Update team's standing
       │
       └── standingId = userId → Update individual's standing
```

Teams and individuals compete in the **same ranking** (for MIXED mode):

```
Rank | Participant     | Solved | Penalty
-----|-----------------|--------|--------
1    | Team Alpha      | 5      | 120
2    | user_john       | 5      | 135
3    | Team Beta       | 4      | 90
4    | user_alice      | 4      | 110
```

### Display Options

Contest creator can configure:

| Option | Description |
|--------|-------------|
| `showTeamMembers: false` | Only show team name |
| `showTeamMembers: true` | Show team name + member names |

Example with `showTeamMembers: true`:
```
Rank | Participant                    | Solved
-----|--------------------------------|-------
1    | Team Alpha (alice, bob, carol) | 5
```

---

## 🔹 Group Integration

### Contests in Groups

When a contest belongs to a group:

1. **Individual registration**: User must be a group member.
2. **Team registration**: ALL selectedMembers must be group members.

### Visibility Rules

* Team members not in the group cannot see contest details.
* Submissions follow group visibility rules.

---

## 🔹 Technical Considerations

### Validation Summary

| Action | Validations |
|--------|-------------|
| Create team | Name uniqueness (optional), valid characters |
| Invite member | User exists, not already member, not already invited |
| Accept invitation | Invitation exists, not expired |
| Register team to contest | See Registration Process above |
| Submit as team | User is in selectedMembers, contest is ACTIVE |
| Leave team | Not in selectedMembers for ACTIVE contest |

### Concurrency

* Use transactions for registration to prevent race conditions.
* Atomic check for "user already registered" across individuals and teams.

### Team Lookup for Submissions

```sql
-- Find if user is in a team for this contest
SELECT teamId FROM ContestTeamParticipant
WHERE contestId = :contestId
  AND :userId = ANY(selectedMembers)
```

This lookup determines whether to attribute the submission to a team.

### Indexes

* `TeamMember(userId, teamId)` - Check user membership
* `ContestTeamParticipant(contestId, teamId)` - Team registration
* `ContestTeamParticipant(contestId, selectedMembers)` - User participation check (GIN index)

---

## 🔹 Related Specs

### Specs to Create

1. **[Create team](Create%20team/spec.md)** - Team creation ✅
2. **[Manage team members](Manage%20team%20members/spec.md)** - Invitations, accept/reject, leave ✅
3. **[Register team to contest](Register%20team%20to%20contest/spec.md)** - Registration with member selection ✅
4. **[View teams](View%20teams/spec.md)** - List user's teams, team details ✅

### Specs to Update

* **Contest management** - Add participationMode, teamSize configuration
* **Submit solution** - Support teamId in submissions ✅
* **Contest standings** - Display teams with optional member names ✅

### Implementation Dependencies

```
Create Team (base) ✅
    ↓
Manage Team Members (P1) ✅ ← (invitations, accept, leave)
    ↓
Register Team to Contest (P1) ✅ ← (requires Contest update)
    ↓
Submit as Team (P1) ✅ ← (requires Submission update)
    ↓
View Teams (P2) ✅
```

---

## 🔹 Key Design Decisions

### Why global teams?
* **Reusability**: Same team competes in multiple contests.
* **History**: Track team performance over time.
* **Identity**: Teams have a persistent identity.

### Why no leadership?
* **Simplicity**: Any member can manage the team.
* **Flexibility**: No bottleneck on a single person.
* **Collaboration**: Equal rights for all members.

### Why selectedMembers?
* **ICPC rules**: Only 3 members compete, but teams can have reserves.
* **Flexibility**: Different members for different contests.
* **Roster management**: Handle unavailability gracefully.

### Why require group membership?
* **Access control**: Ensure all participants can access contest materials.
* **Consistency**: Same rules for individuals and teams.
* **Privacy**: Prevent unauthorized access to group content.

### Why shared standings?
* **Competition fairness**: Teams and individuals compete equally.
* **Simplicity**: One unified ranking.
* **Transparency**: Clear comparison between all participants.

---

*This document should be updated when new design decisions are made or additional specs are implemented.*
