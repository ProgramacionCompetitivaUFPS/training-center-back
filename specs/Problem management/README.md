# Problem Management - Business Logic & Design

**Created**: 2025-12-20

This document centralizes the complete business logic of the Problem Management system and design considerations being applied across all related specs.

---

## 🔹 General Concept

* **`Problem`** is a global, reusable entity in the system.
* Problems are not tied to specific groups or contests - they exist independently.
* Problems can be added to multiple contests across different groups.
* Problems have a **lifecycle** (DRAFT → PUBLISHED) and **accessibility** (PRIVATE/PUBLIC).
* Problems contain **judging components** (test cases, checker, validator) used to evaluate submissions.

---

## 🔹 Problem States

### Status (Publication State)

| Status | Description | Can Modify | Available for Contests |
|--------|-------------|------------|----------------------|
| `DRAFT` | Problem is being built. Can have partial data. | ✅ Yes | ❌ No |
| `PUBLISHED` | Problem is complete and validated. | ❌ No (must unpublish first) | ✅ Yes |

### State Transitions

```
DRAFT ──(publish)──> PUBLISHED ──(unpublish)──> DRAFT
```

### Publication Requirements

To publish a problem, all of the following are required:
- `title` (always present)
- `statement` (LaTeX format)
- `timeLimit` (milliseconds)
- `memoryLimit` (MiB)
- `testCases` (ZIP file, ICPC format)
- At least one `solution` file

Optional for publication:
- `tags` (always optional)
- `checker` (default: exact match)
- `validator` (input validation)

---

## 🔹 Accessibility (Who Can Add to Contests)

| Accessibility | Who Can Add to Contests | Default |
|---------------|------------------------|---------|
| `PRIVATE` | Only modifiers (author + assigned modifiers) | ✅ Yes |
| `PUBLIC` | Any contest creator | ❌ No |

### Accessibility Change Behavior

When changing from `PUBLIC` to `PRIVATE`:
- Existing contest associations are **NOT** affected
- Existing submissions continue to execute normally
- Registered contest participants can still submit within those contests
- Only **new** contest additions are restricted to modifiers
- Users lose public access to the problem outside of contests

---

## 🔹 Roles and Permissions

### Problem Creation

| Role | Can Create Problems |
|------|---------------------|
| **Admin** | ✅ Yes |
| **Coach** | ✅ Yes |
| **Contestant** | ❌ No |

### Problem Management (Update, Files, Publish)

| Action | Author | Admin | Modifier | Contestant |
|--------|--------|-------|----------|------------|
| Update metadata | ✅ | ✅ | ✅ | ❌ |
| Update accessibility | ✅ | ✅ | ✅ | ❌ |
| Upload files | ✅ | ✅ | ✅ | ❌ |
| Delete files | ✅ | ✅ | ✅ | ❌ |
| Publish/Unpublish | ✅ | ✅ | ✅ | ❌ |
| Add modifier | ✅ | ✅ | ❌ | ❌ |
| Remove modifier | ✅ | ✅ | ❌ | ❌ |
| Delete problem | ✅ | ✅ | ❌ | ❌ |

### Modifiers

* **Author**: The user who created the problem. Automatically a modifier.
* **Modifiers**: Users assigned by the author or Admin to help edit the problem.
* Modifiers can edit metadata, upload/delete files, and publish/unpublish.
* Modifiers **cannot** add other modifiers or delete the problem.

---

## 🔹 Problem Attributes

### Basic Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `slug` | string | Yes | Unique identifier, user-provided, immutable, 3-70 chars |
| `title` | string | Yes | Problem title (normalized NFKC, max 200 chars) |
| `statement` | string | For publish | Problem statement in LaTeX format |
| `timeLimit` | integer | For publish | Default time limit in milliseconds |
| `memoryLimit` | integer | For publish | Default memory limit in MiB |
| `languageOverrides` | array | No | Language-specific limit overrides |
| `tags` | string[] | No | Tags from predefined list (always optional) |
| `status` | enum | Yes | `DRAFT` or `PUBLISHED` |
| `accessibility` | enum | Yes | `PUBLIC` or `PRIVATE` (default: PRIVATE) |
| `authorId` | UUID | Yes | Problem creator (immutable) |
| `modifierIds` | UUID[] | No | Assigned editors |
| `createdAt` | timestamp | Yes | Creation timestamp (immutable) |
| `updatedAt` | timestamp | Yes | Last modification timestamp |
| `problemJudgingUpdatedAt` | timestamp | No | Updated when judging components change |

### Language-Specific Limits

Problems can define different time/memory limits for specific languages:

```json
{
  "timeLimit": 2000,
  "memoryLimit": 256,
  "languageOverrides": [
    { "language": "python310", "timeLimit": 4000 },
    { "language": "java17", "memoryLimit": 512 }
  ]
}
```

Limits are validated against the Virtual Object maximums.

---

## 🔹 Slug (User-Provided Identifier)

The slug is provided by the user when creating a problem and serves as the unique identifier.

### Validation Rules

| Rule | Description |
|------|-------------|
| Length | 3-70 characters |
| Characters | Only lowercase letters (a-z), numbers (0-9), and hyphens (-) |
| Format | Cannot start or end with hyphen |
| Format | Cannot contain consecutive hyphens (--) |
| Uniqueness | Must be unique across all problems |

### Examples

**Valid slugs**:
- `sum-two-numbers` ✅
- `prob-001` ✅
- `dp-knapsack` ✅
- `binary-search-tree` ✅

**Invalid slugs**:
- `ab` ❌ (too short)
- `Sum-Two` ❌ (uppercase)
- `-sum-` ❌ (starts/ends with hyphen)
- `sum--two` ❌ (consecutive hyphens)

> **Important**: Slugs are **immutable** - they cannot be changed after creation. If a slug already exists, the creation is rejected with `SLUG_ALREADY_EXISTS`.

---

## 🔹 Problem Files

### File Types

| File Type | Extensions | Max Size | Required | Updates problemJudgingUpdatedAt |
|-----------|------------|----------|----------|--------------------------------|
| `testCases` | .zip | 200 MB | For publish | ✅ Yes |
| `solution` | .py, .cpp, .java | 10 MB | For publish (at least 1) | ❌ No |
| `checker` | .py, .cpp | 10 MB | No (default: exact match) | ✅ Yes |
| `validator` | .py, .cpp | 10 MB | No | ✅ Yes |

### Test Cases ZIP Structure (ICPC Format)

```
testcases.zip/
├── data/
│   ├── sample/           # Example cases (shown to users)
│   │   ├── 1.in
│   │   ├── 1.ans
│   │   └── ...
│   └── secret/           # Hidden cases (for judging)
│       ├── 01.in
│       ├── 01.ans
│       └── ...
```

### Judging Components

Components that affect how submissions are evaluated:

- **Test Cases**: Input/output files for evaluation
- **Checker**: Custom program that validates submission output (optional)
- **Validator**: Program that validates test inputs conform to constraints (optional)

When ANY judging component is uploaded/replaced, `problemJudgingUpdatedAt` is updated.

---

## 🔹 Rejudging

### When Rejudging is Needed

Submissions need rejudging when: `submittedAt < problemJudgingUpdatedAt`

### problemJudgingUpdatedAt

This timestamp is updated when any judging component is uploaded:
- Test cases (`fileType=testCases`)
- Checker (`fileType=checker`)
- Validator (`fileType=validator`)

> **Note**: Uploading solution files does NOT trigger `problemJudgingUpdatedAt` update.

### Rejudge Permissions

| Action | Contest Owner | Lead | Contestant (own) | Admin |
|--------|---------------|------|------------------|-------|
| Rejudge all in active contest | ✅ | ✅ | ❌ | ✅ |
| Rejudge own submission (practice/finished) | ✅ | ✅ | ✅ | ✅ |
| Rejudge own submission (postcompetition) | ✅ | ✅ | ✅ | ✅ |
| Rejudge own submission (in active contest) | ❌ | ❌ | ❌ | ✅ |
| Rejudge all for problem (global) | ❌ | ❌ | ❌ | ✅ |

### Standing Updates on Rejudge

- **Active contest (submittedAt ≤ endTime)**: Standing is updated
- **Finished contest**: Standing is frozen, NOT updated
- **Postcompetition (submittedAt > endTime)**: Standing is NOT updated

---

## 🔹 Publication Validation

The publish endpoint triggers comprehensive validation:

```
1. Check required fields
   ├─ title ✓ (always present)
   ├─ statement
   ├─ timeLimit
   ├─ memoryLimit
   ├─ testCases file
   └─ solution file(s)

2. Validate test cases structure (ICPC format)

3. Compile checker (if provided)

4. Compile validator (if provided)

5. Run validator against all inputs (if validator exists)

6. Compile solution(s)

7. Execute solution(s) against all test cases
   ├─ Check output correctness
   ├─ Enforce time limit
   └─ Enforce memory limit

8. If all pass → status = PUBLISHED
   If any fail → return detailed error logs
```

---

## 🔹 Virtual Object (Limits Configuration)

Problems validate limits against platform-wide maximums defined in the Virtual Object:

```json
{
  "maxTimeLimitGlobal": 300000,
  "maxMemoryLimitGlobal": 2048,
  "languageOverrides": [
    { "language": "cpp20", "maxTimeLimit": 300000, "maxMemoryLimit": 2048 },
    { "language": "java17", "maxTimeLimit": 400000, "maxMemoryLimit": 2048 },
    { "language": "python310", "maxTimeLimit": 600000, "maxMemoryLimit": 2048 }
  ]
}
```

See Platform README for full configuration details.

---

## 🔹 Tags

* Tags are loaded from an external configuration file at application startup.
* Tags are **always optional** - not required for creation or publication.
* Invalid tags are rejected with descriptive error messages.

---

## 🔹 Problem Import (ICPC Format)

Problems can be imported from ICPC-format ZIP packages:

```
problem-package.zip/
├── problem.yaml              # Problem metadata
├── problem_statement/
│   └── problem.en.tex        # LaTeX statement
├── data/
│   ├── sample/               # Sample test cases
│   └── secret/               # Secret test cases
├── submissions/              # Optional solutions
│   └── accepted/
├── output_validators/        # Optional custom checker
└── input_validators/         # Optional input validator
```

Imported problems are created with status `DRAFT` and accessibility `PRIVATE`.

---

## 🔹 Content Deletion

### When a Problem is Deleted

| Entity | Action |
|--------|--------|
| Problem | Hard delete |
| Problem files | Hard delete (test cases, solutions, checker, validator) |
| Contest_Problem | Hard delete (associations to contests removed) |
| Submission | Preserved with `problem_id` intact (for user history) |

> **Note**: Submissions are preserved to maintain user history and statistics.

---

## 🔹 Technical Considerations

### Concurrency

* Optimistic locking for problem updates
* Handle concurrent file uploads gracefully
* Slug generation handles concurrent creation with same title

### Performance

* Index on `slug` for problem queries
* Index on `authorId` for author's problems
* Index on `status` and `accessibility` for filtering
* File uploads go directly to storage service

### Validation

* All text normalized using Unicode NFKC
* Limits validated against Virtual Object at creation and update
* Test cases ZIP structure validated on upload
* Full validation on publish

---

## 🔹 Related Specs

### Implemented Specs

1. **[Create problem](Create%20problem/spec.md)** - Problem creation with minimal or full data, ZIP import
2. **[Update problem](Update%20problem/spec.md)** - Metadata updates, file uploads, modifier management
3. **[Change problem visibility](Change%20problem%20visibility/spec.md)** - Publish/unpublish and accessibility changes
4. **[Rejudge submissions](Rejudge%20submissions/spec.md)** - Rejudging when judging components change

### Future Specs (Planned)

* **Delete problem** - Remove problems (handle submissions, contest associations)
* **View problem** - Problem details for contestants and modifiers
* **Search problems** - Filter by tags, accessibility, author

### Implementation Dependencies

```
Create Problem (base)
    ↓
Update Problem (P1) ✅ ← (metadata, files, modifiers)
    ↓
Change Problem Visibility (P1) ✅ ← (publish/unpublish, accessibility)
    ↓
Rejudge Submissions (P1) ✅ ← (when judging components change)
    ↓
Delete Problem (P2) ← (handle associations)
    ↓
View Problem (P2) ← (public information)
```

---

## 🔹 Key Design Decisions

### Why global problems instead of group-owned?

* **Reusability**: Problems can be used in multiple contests across groups
* **Simplicity**: No complex ownership hierarchy
* **Sharing**: Coaches can share problems easily via accessibility settings

### Why DRAFT/PUBLISHED status?

* **Quality control**: Ensures problems are complete before being used
* **Validation**: Full validation pipeline guarantees problem integrity
* **Safety**: Prevents incomplete problems from being used in contests

### Why separate status and accessibility?

* **Independence**: These are orthogonal concerns
* **Flexibility**: A problem can be PUBLISHED + PRIVATE (modifiers only)
* **Use cases**: Some problems should be validated but restricted

### Why user-provided immutable slugs?

* **User control**: Users choose meaningful, memorable identifiers
* **Unique identifiers**: URL-friendly, human-readable
* **Stability**: URLs don't break when titles change
* **Validation**: System validates format (3-70 chars, lowercase, alphanumeric with hyphens)

### Why single problemJudgingUpdatedAt instead of separate timestamps?

* **Simplicity**: Single check for rejudge necessity
* **Consistency**: All submissions rejudged when any component changes
* **Safety**: No risk of missing rejudges due to partial checks

### Why preserve submissions when problems are deleted?

* **User history**: Users can see their submission history
* **Analytics**: Submission data valuable for statistics
* **Fairness**: Users' work should not be deleted

### Why accessibility change is non-destructive?

* **Fairness**: Contest participants shouldn't lose access mid-contest
* **Stability**: Existing contest associations remain valid
* **Control**: Only restricts future additions

---

*This document should be updated when new design decisions are made or additional specs are implemented.*

