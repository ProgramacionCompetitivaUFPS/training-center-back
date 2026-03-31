# Feature Specification: Create Problem

**Created**: 2025-12-20

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Create a new problem with minimal data (Priority: P1)

As a Coach or Admin, I want to create a new problem by providing a slug and title so that I can start building the problem incrementally and publish it when ready.

**Why this priority**: Problem creation is the foundation for the platform's core functionality. Allowing incremental creation with minimal initial data provides a better user experience and enables coaches to work on problems over time.

**Independent Test**: This user story can be tested independently by consuming the `POST /problems` endpoint with valid authentication (Coach or Admin role), validating that a problem is created with status `DRAFT`, accessibility `PRIVATE`, and the provided slug.

**Acceptance Scenarios**:

1. **Scenario**: Successful problem creation with slug and title
   - **Given** a Coach or Admin is authenticated
   - **When** they submit a problem creation request with slug and title
   - **Then** the system creates the problem with status `DRAFT` and accessibility `PRIVATE`
   - **And** uses the provided slug as the unique identifier
   - **And** sets the authenticated user as the problem author
   - **And** returns the created problem data

2. **Scenario**: Successful problem creation with full metadata
   - **Given** a Coach or Admin is authenticated
   - **When** they submit a problem creation request with slug, title, statement, timeLimit, memoryLimit, and tags
   - **Then** the system creates the problem with all provided data
   - **And** status is `DRAFT` and accessibility is `PRIVATE`
   - **And** returns the created problem data

3. **Scenario**: Contestant attempts to create problem
   - **Given** a Contestant is authenticated
   - **When** they attempt to create a problem
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

4. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** a problem creation request is submitted
   - **Then** the system rejects with 401 Unauthorized

5. **Scenario**: Missing slug or title
   - **Given** a Coach or Admin is authenticated
   - **When** the request omits the slug or title field
   - **Then** the system rejects with 400 Bad Request indicating the missing required field

6. **Scenario**: Invalid time limit or memory limit
   - **Given** a Coach or Admin is authenticated
   - **And** timeLimit or memoryLimit is zero, negative, or exceeds maximum allowed value from Virtual Object
   - **When** they submit the problem creation request
   - **Then** the system rejects with 400 Bad Request indicating invalid limits

7. **Scenario**: Invalid tags provided
   - **Given** a Coach or Admin is authenticated
   - **And** the request includes tags that are not in the system's predefined tag list
   - **When** they submit the problem creation request
   - **Then** the system rejects with 400 Bad Request indicating invalid tags

12. **Scenario**: Unsupported language in languageOverrides
    - **Given** a Coach or Admin is authenticated
    - **When** they submit a problem with a language in `languageOverrides` that is not supported by the platform (not in Virtual Object `supportedLanguages` list)
    - **Then** the system rejects with 400 Bad Request and field `languageOverrides.language` indicating the unsupported language

13. **Scenario**: Duplicate language in languageOverrides
    - **Given** a Coach or Admin is authenticated
    - **When** they submit a problem with the same language appearing more than once in `languageOverrides`
    - **Then** the system rejects with 400 Bad Request and field `languageOverrides.language` indicating the duplicate language

14. **Scenario**: Statement exceeds maximum length
    - **Given** a Coach or Admin is authenticated
    - **When** they submit a problem with a `statement` field exceeding 150,000 characters
    - **Then** the system rejects with 400 Bad Request and field `statement`

8. **Scenario**: Slug already exists
   - **Given** a problem with slug "sum-two-numbers" already exists
   - **When** a Coach creates a new problem with slug "sum-two-numbers"
   - **Then** the system rejects with 409 Conflict (SLUG_ALREADY_EXISTS)
   - **And** returns a message indicating the slug is already in use

9. **Scenario**: Slug too short
   - **Given** a Coach or Admin is authenticated
   - **When** they submit a problem with slug "ab" (less than 3 characters)
   - **Then** the system rejects with 400 Bad Request (SLUG_TOO_SHORT)

10. **Scenario**: Slug too long
    - **Given** a Coach or Admin is authenticated
    - **When** they submit a problem with slug exceeding 70 characters
    - **Then** the system rejects with 400 Bad Request (SLUG_TOO_LONG)

11. **Scenario**: Slug with invalid format
    - **Given** a Coach or Admin is authenticated
    - **When** they submit a problem with slug containing invalid characters (uppercase, spaces, special chars)
    - **Then** the system rejects with 400 Bad Request (INVALID_SLUG_FORMAT)

---

### User Story 2 – Create problem by importing ZIP (Priority: P2)

As a Coach or Admin, I want to create a new problem by uploading a complete ICPC-format ZIP package so that I can quickly import existing problems.

**Why this priority**: Useful for migrating problems from other systems or batch importing. Secondary to manual creation flow.

**Independent Test**: This user story can be tested independently by consuming the `POST /problems/import` endpoint with a valid ICPC-format ZIP, validating that the problem is created with all data extracted from the package.

**Acceptance Scenarios**:

1. **Scenario**: Successful import from valid ZIP
   - **Given** a Coach or Admin is authenticated
   - **And** has a valid ICPC-format ZIP with all required files
   - **When** they upload the ZIP to import and provide a valid unique 'slug'
   - **Then** the system extracts all data (title from problem.yaml, statement, test cases, etc.)
   - **And** creates the problem with status `DRAFT` and accessibility `PRIVATE` under the provided slug
   - **And** returns the created problem data

2. **Scenario**: Import ZIP with missing required files
   - **Given** a Coach or Admin is authenticated
   - **And** the ZIP is missing required files (e.g., problem.yaml, test cases)
   - **When** they attempt to import
   - **Then** the system rejects with 400 Bad Request
   - **And** returns detailed logs of missing files

3. **Scenario**: Import ZIP with invalid structure
   - **Given** a Coach or Admin is authenticated
   - **And** the ZIP has an invalid directory structure
   - **When** they attempt to import
   - **Then** the system rejects with 400 Bad Request indicating structure issues

4. **Scenario**: Contestant attempts to import
   - **Given** a Contestant is authenticated
   - **When** they attempt to import a problem
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

5. **Scenario**: ZIP file too large
   - **Given** a Coach or Admin is authenticated
   - **And** the ZIP exceeds the maximum allowed size (200 MB)
   - **When** they attempt to upload
   - **Then** the system rejects with 400 Bad Request (FILE_TOO_LARGE)

6. **Scenario**: Import ZIP with already existing slug
   - **Given** a Coach or Admin is authenticated
   - **When** they attempt to upload a ZIP with a 'slug' that already exists in the database
   - **Then** the system rejects with 409 Conflict (SLUG_ALREADY_EXISTS) before unzipping
   
---

### Edge Cases

- Problem title with Unicode characters requiring normalization (NFKC).
- User-provided slug with invalid format (uppercase, special chars, consecutive hyphens).
- User-provided slug that's too short (< 3 chars) or too long (> 70 chars).
- User-provided slug that already exists (duplicate detection).
- Concurrent creation requests with same slug (race condition handling).
- Import ZIP with corrupted files.
- Import ZIP with unsupported file formats.
- Import ZIP with very large test case files.

---

## API Contract

### POST /problems

Create a new problem with metadata.

> **Important**: Only users with Coach or Admin roles can create problems. The minimum required field is `title`. All other fields can be added later via the Update Problem spec (`PUT /problems/p/{slug}`).

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |
| Idempotency-Key | string | No | Optional UUID for idempotent retries |

**Request Body**:

```json
{
  "slug": "sum-two-numbers",
  "title": "Sum of Two Numbers",
  "statement": "Given two integers a and b, return their sum.",
  "timeLimit": 2000,
  "memoryLimit": 256,
  "languageOverrides": [
    { "language": "python310", "timeLimit": 4000 },
    { "language": "java17", "memoryLimit": 512 }
  ],
  "tags": ["math", "beginner"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| slug | string | Yes | Unique problem identifier (3-70 chars, lowercase alphanumeric with hyphens only) |
| title | string | Yes | Problem title (normalized Unicode NFKC) |
| statement | string | No | Problem statement in LaTeX format |
| timeLimit | integer | No | Default time limit in milliseconds for all languages (> 0, max from Virtual Object `maxTimeLimitGlobal`) |
| memoryLimit | integer | No | Default memory limit in MiB for all languages (> 0, max from Virtual Object `maxMemoryLimitGlobal`) |
| languageOverrides | array | No | Language-specific overrides. Only specify languages that need different limits than the defaults. |
| tags | string[] | No | Array of tags from system's predefined list (always optional) |

**languageOverrides array structure**:

Each entry specifies overrides for a specific language. Only include languages that need different limits.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| language | string | Yes | Language identifier (e.g., `cpp20`, `java17`, `python310`) |
| timeLimit | integer | No | Override time limit for this language (> 0, must not exceed max from Virtual Object for this language) |
| memoryLimit | integer | No | Override memory limit for this language (> 0, must not exceed max from Virtual Object for this language) |

> **Note**: See the Platform README for Virtual Object configuration including `maxTimeLimitGlobal`, `maxMemoryLimitGlobal`, and language-specific maximums.

**Responses**:

#### 201 Created
Problem created successfully.

```json
{
  "slug": "sum-two-numbers",
  "title": "Sum of Two Numbers",
  "statement": "Given two integers a and b, return their sum.",
  "timeLimit": 2000,
  "memoryLimit": 256,
  "languageOverrides": [
    { "language": "python310", "timeLimit": 4000 },
    { "language": "java17", "memoryLimit": 512 }
  ],
  "tags": ["math", "beginner"],
  "status": "DRAFT",
  "accessibility": "PRIVATE",
  "author": {
    "nickname": "coach_john",
    "name": "John Smith"
  },
  "modifiers": [],
  "files": {
    "testCases": false,
    "solutions": [],
    "checker": false,
    "validator": false
  },
  "createdAt": "2025-12-20T10:00:00Z",
  "updatedAt": "2025-12-20T10:00:00Z"
}
```

#### 400 Bad Request
Validation error (missing fields, invalid slug format, invalid limits, invalid tags).

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "slug",
      "message": "Slug is required"
    }
  ]
}
```

```json
{
  "error": "SLUG_TOO_SHORT",
  "message": "Slug must be at least 3 characters long"
}
```

```json
{
  "error": "SLUG_TOO_LONG",
  "message": "Slug must not exceed 70 characters"
}
```

```json
{
  "error": "INVALID_SLUG_FORMAT",
  "message": "Slug must contain only lowercase letters, numbers, and hyphens. Cannot start/end with hyphen or have consecutive hyphens."
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "title",
      "message": "Title is required"
    }
  ]
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "timeLimit",
      "message": "Time limit must be between 1 and 300000 milliseconds"
    }
  ]
}
```

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "tags",
      "message": "Invalid tag 'unknown-tag'. Valid tags: math, beginner, dp, graphs, ..."
    }
  ]
}
```

#### 409 Conflict
Slug already exists.

```json
{
  "error": "SLUG_ALREADY_EXISTS",
  "message": "A problem with slug 'sum-two-numbers' already exists"
}
```

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission to create problems.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only Coach and Admin users can create problems"
}
```

---

### POST /problems/import

Create a new problem by importing an ICPC-format ZIP package.

> **Important**: Only users with Coach or Admin roles can import problems. The ZIP must follow the ICPC problem package format.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | multipart/form-data |
| Idempotency-Key | string | No | Optional UUID for idempotent retries |

**Form Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| slug | string | Yes | Unique problem identifier (3-70 chars, lowercase alphanumeric with hyphens only). Overrides any slug found in `problem.yaml` |
| file | file | Yes | The ICPC-format ZIP file to import |

**Expected ZIP Structure (ICPC Format)**:
```
problem-package.zip/
├── problem.yaml              # Problem metadata (name, time limit, etc.)
├── problem_statement/
│   └── problem.en.tex        # LaTeX statement
├── data/
│   ├── sample/               # Sample test cases
│   │   ├── 1.in
│   │   ├── 1.ans
│   │   └── ...
│   └── secret/               # Secret test cases
│       ├── 01.in
│       ├── 01.ans
│       └── ...
├── solutions/              # Optional solutions
│   └── solution_1.cpp
│   └── solution_2.py
├── checker.cpp               # Optional custom checker
└── validator.cpp             # Optional input validator
```

**problem.yaml Example**:
```yaml
name: Sum of Two Numbers
slug: sum-two-numbers
time_limit: 2.0
memory_limit: 256
author: coach_john
source: Training Camp 2025
```

**Responses**:

#### 201 Created
Problem imported successfully.

```json
{
  "slug": "sum-of-two-numbers",
  "title": "Sum of Two Numbers",
  "statement": "Given two integers...",
  "timeLimit": 2000,
  "memoryLimit": 256,
  "tags": [],
  "status": "DRAFT",
  "accessibility": "PRIVATE",
  "author": {
    "nickname": "coach_john",
    "name": "John Smith"
  },
  "modifiers": [],
  "files": {
    "testCases": true,
    "solutions": ["solution.cpp"],
    "checker": false,
    "validator": true
  },
  "importLogs": [
    "✓ problem.yaml parsed successfully",
    "✓ Statement extracted from problem.en.tex",
    "✓ 3 sample test cases imported",
    "✓ 15 secret test cases imported",
    "✓ 1 accepted solution imported",
    "✓ Input validator imported"
  ],
  "createdAt": "2025-12-20T10:00:00Z",
  "updatedAt": "2025-12-20T10:00:00Z"
}
```

#### 400 Bad Request
Validation errors (missing fields, invalid slug format) or invalid ZIP file.

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Invalid request data",
  "details": [
    {
      "field": "slug",
      "message": "Slug is required"
    }
  ]
}
```

```json
{
  "error": "INVALID_PACKAGE",
  "message": "Invalid ICPC problem package",
  "importLogs": [
    "✗ problem.yaml not found (required)",
    "✗ No test cases found in data/ directory"
  ]
}
```

```json
{
  "error": "FILE_TOO_LARGE",
  "message": "File exceeds maximum size of 200 MB"
}
```

#### 401 Unauthorized
Authentication failed.

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only Coach and Admin users can import problems"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Problem Creation (Minimal)**
- **FR-001**: The system MUST allow only users with Coach or Admin roles to create problems.
- **FR-002**: The system MUST require `slug` and `title` fields to create a problem.
- **FR-003**: The system MUST validate that the slug is unique across all problems.
- **FR-004**: If the slug already exists, the system MUST reject with 409 Conflict (SLUG_ALREADY_EXISTS).
- **FR-005**: The slug MUST be between 3 and 70 characters.
- **FR-005.1**: The slug MUST contain only lowercase letters (a-z), numbers (0-9), and hyphens (-).
- **FR-005.2**: The slug MUST NOT start or end with a hyphen.
- **FR-005.3**: The slug MUST NOT contain consecutive hyphens.
- **FR-006**: The system MUST set problem status to `DRAFT` and accessibility to `PRIVATE` when created.
- **FR-007**: The system MUST set the authenticated user as the problem author.
- **FR-008**: The system MUST normalize the title using Unicode NFKC normalization.
- **FR-009**: The system MUST validate timeLimit as positive integer ≤ `maxTimeLimitGlobal` from Virtual Object if provided.
- **FR-010**: The system MUST validate memoryLimit as positive integer ≤ `maxMemoryLimitGlobal` from Virtual Object if provided.
- **FR-010.1**: The system MUST validate languageOverrides array if provided.
- **FR-010.2**: The system MUST validate that each entry's `language` field is a valid language identifier supported by the platform (present in Virtual Object `supportedLanguages`). If not supported, reject with VALIDATION_ERROR.
- **FR-010.3**: The system MUST validate that each override's timeLimit (if provided) does not exceed the maximum from Virtual Object for that language.
- **FR-010.4**: The system MUST validate that each override's memoryLimit (if provided) does not exceed the maximum from Virtual Object for that language.
- **FR-010.5**: The system MUST validate that timeLimit and memoryLimit in languageOverrides are positive integers when provided.
- **FR-010.6**: The system MUST reject with VALIDATION_ERROR if the same language appears more than once in `languageOverrides`.
- **FR-011**: The system MUST validate tags against the system's predefined tag list if provided.
- **FR-012**: Tags MUST always be optional (not required for creation or publication).
- **FR-025**: The system MUST validate that `statement` does not exceed 150,000 characters when provided.

**Problem Import (ZIP)**
- **FR-013**: The system MUST accept ICPC-format ZIP packages for problem import.
- **FR-014**: The system MUST extract problem metadata from `problem.yaml`.
- **FR-015**: The system MUST extract the problem statement from LaTeX files in `problem_statement/`.
- **FR-016**: The system MUST import test cases from the `data/` directory structure.
- **FR-017**: The system MUST import solutions from `submissions/accepted/` if present.
- **FR-018**: The system MUST import validators from appropriate directories if present.
- **FR-019**: The system MUST return detailed logs of what was imported or what failed.
- **FR-020**: The system MUST enforce a maximum ZIP size of 200 MB.

**General**
- **FR-021**: The system MUST NOT return internal IDs in any response.
- **FR-022**: The system MUST record createdAt and updatedAt timestamps.
- **FR-023**: The system MUST return validation errors with consistent structure.
- **FR-024**: The system SHOULD support Idempotency-Key header for safe retries.

### Key Entities

- **Problem**: Represents a programming problem.  
  Key attributes:
  - `slug` (string, unique, user-provided, lowercase alphanumeric with hyphens, 3-70 chars)
  - `title` (string, required, normalized NFKC)
  - `statement` (string, LaTeX format, nullable)
  - `timeLimit` (integer, milliseconds, nullable, default for all languages, max from Virtual Object)
  - `memoryLimit` (integer, MiB, nullable, default for all languages, max from Virtual Object)
  - `languageOverrides` (array, nullable, language-specific limit overrides)
  - `tags` (array of strings, always optional, from predefined list)
  - `status` (enum: `DRAFT` | `PUBLISHED`, default: `DRAFT`)
  - `accessibility` (enum: `PUBLIC` | `PRIVATE`, default: `PRIVATE`)
  - `authorId` (string, UUID, FK to User)
  - `modifierIds` (array of UUIDs, FK to User, empty on creation)
  - `testCasesFileKey` (string, nullable, reference to test cases ZIP)
  - `solutionFileKeys` (array of strings, references to solution files)
  - `checkerFileKey` (string, nullable, reference to checker file)
  - `validatorFileKey` (string, nullable, reference to validator file)
  - `problemJudgingUpdatedAt` (timestamp, nullable, updated when judging components are uploaded)
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp)

> **problemJudgingUpdatedAt**: This timestamp is automatically updated whenever any judging component is uploaded (test cases, checker, or validator). Used by the Rejudge system to determine which submissions need rejudging. See Rejudge Submissions spec for details.

> **Problem Status** (publication state):
> - `DRAFT`: Problem is being built. Can have partial data. Can be modified. Not available for contests/practice.
> - `PUBLISHED`: Problem is complete and published. Cannot be modified (must unpublish first). Available for contests/practice.

> **Problem Accessibility** (who can add it to contests):
> - `PRIVATE`: Only the problem's modifiers (author + assigned modifiers) can add this problem to a contest. Default for all new problems.
> - `PUBLIC`: Any contest creator can add this problem to their contest.

### Slug Validation Rules

The slug is provided by the user and must follow these rules:

| Rule | Description |
|------|-------------|
| Length | 3-70 characters |
| Characters | Only lowercase letters (a-z), numbers (0-9), and hyphens (-) |
| Format | Cannot start or end with hyphen |
| Format | Cannot contain consecutive hyphens (--) |
| Uniqueness | Must be unique across all problems |

**Valid examples**:
- `sum-two-numbers` ✅
- `prob-001` ✅
- `dp-knapsack` ✅
- `a1b` ✅

**Invalid examples**:
- `ab` ❌ (too short, < 3 chars)
- `Sum-Two-Numbers` ❌ (uppercase not allowed)
- `-sum-two-` ❌ (cannot start/end with hyphen)
- `sum--two` ❌ (consecutive hyphens not allowed)
- `sum two` ❌ (spaces not allowed)
- `probléma` ❌ (special characters not allowed)

### Tags

Tags are loaded from an external configuration file at application startup. They are always optional and not required for problem creation or publication.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Coach and Admin users can create problems with slug and title via `POST /problems` with HTTP 201.
- **SC-002**: Slug is provided by user and validated (3-70 chars, lowercase alphanumeric with hyphens).
- **SC-003**: Duplicate slugs are rejected with 409 Conflict (SLUG_ALREADY_EXISTS).
- **SC-004**: Problems are created with status `DRAFT` and accessibility `PRIVATE`.
- **SC-005**: Authenticated user is set as the problem author.
- **SC-006**: Optional fields (statement, timeLimit, memoryLimit, tags) can be provided at creation.
- **SC-007**: ICPC-format ZIP can be imported via `POST /problems/import` with HTTP 201.
- **SC-008**: Import logs detail what was imported or failed.
- **SC-009**: Contestant users attempting creation receive HTTP 403.
- **SC-010**: No internal IDs are returned in any response.
- **SC-011**: Tags are always optional.
- **SC-012**: Invalid tags are rejected with descriptive error message.

---

## Optional Notes

- **Idempotency**: If `Idempotency-Key` header is provided and a request with the same key was already processed, return the same response without creating a duplicate.
- **Title length**: Consider enforcing a maximum title length (e.g., 200 characters).
- **Import vs Create**: Import creates a more complete problem in one step, but still requires publication to become `PUBLISHED`.
- **Related specs**:
  - Update Problem: For modifying metadata, uploading files, publishing/unpublishing
  - Delete Problem: For removing problems (to be defined)

