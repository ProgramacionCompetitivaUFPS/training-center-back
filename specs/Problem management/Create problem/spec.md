# Feature Specification: Create Problem

**Created**: 2025-12-20

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Create a new problem with minimal data (Priority: P1)

As a Coach or Admin, I want to create a new problem by providing at least a title so that I can start building the problem incrementally and publish it when ready.

**Why this priority**: Problem creation is the foundation for the platform's core functionality. Allowing incremental creation with minimal initial data provides a better user experience and enables coaches to work on problems over time.

**Independent Test**: This user story can be tested independently by consuming the `POST /problems` endpoint with valid authentication (Coach or Admin role), validating that a problem is created with status `DRAFT`, accessibility `PRIVATE`, and a unique auto-generated slug.

**Acceptance Scenarios**:

1. **Scenario**: Successful problem creation with only title
   - **Given** a Coach or Admin is authenticated
   - **When** they submit a problem creation request with only a title
   - **Then** the system creates the problem with status `DRAFT` and accessibility `PRIVATE`
   - **And** generates a unique slug from the title (lowercase, alphanumeric, hyphens)
   - **And** sets the authenticated user as the problem author
   - **And** returns the created problem data

2. **Scenario**: Successful problem creation with full metadata
   - **Given** a Coach or Admin is authenticated
   - **When** they submit a problem creation request with title, statement, timeLimit, memoryLimit, and tags
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

5. **Scenario**: Missing title
   - **Given** a Coach or Admin is authenticated
   - **When** the request omits the title field
   - **Then** the system rejects with 400 Bad Request indicating title is required

6. **Scenario**: Invalid time limit or memory limit
   - **Given** a Coach or Admin is authenticated
   - **And** timeLimit or memoryLimit is zero, negative, or exceeds maximum allowed value
   - **When** they submit the problem creation request
   - **Then** the system rejects with 400 Bad Request indicating invalid limits

7. **Scenario**: Invalid tags provided
   - **Given** a Coach or Admin is authenticated
   - **And** the request includes tags that are not in the system's predefined tag list
   - **When** they submit the problem creation request
   - **Then** the system rejects with 400 Bad Request indicating invalid tags

8. **Scenario**: Title generates duplicate slug
   - **Given** a problem with slug "sum-of-two-numbers" already exists
   - **When** a Coach creates a new problem with title "Sum of Two Numbers"
   - **Then** the system generates a unique slug (e.g., "sum-of-two-numbers-1")
   - **And** the problem is created successfully

---

### User Story 2 – Create problem by importing ZIP (Priority: P2)

As a Coach or Admin, I want to create a new problem by uploading a complete ICPC-format ZIP package so that I can quickly import existing problems.

**Why this priority**: Useful for migrating problems from other systems or batch importing. Secondary to manual creation flow.

**Independent Test**: This user story can be tested independently by consuming the `POST /problems/import` endpoint with a valid ICPC-format ZIP, validating that the problem is created with all data extracted from the package.

**Acceptance Scenarios**:

1. **Scenario**: Successful import from valid ZIP
   - **Given** a Coach or Admin is authenticated
   - **And** has a valid ICPC-format ZIP with all required files
   - **When** they upload the ZIP to import
   - **Then** the system extracts all data (title from problem.yaml, statement, test cases, etc.)
   - **And** creates the problem with status `DRAFT` and accessibility `PRIVATE`
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

---

### Edge Cases

- Problem title with Unicode characters requiring normalization (NFKC).
- Slug auto-generation from title with special characters (e.g., "¿Qué pasa?" → "que-pasa").
- Problem title that generates very long slug (truncation may be needed, max 100 chars).
- Title with only special characters (should be rejected or fallback to UUID-based slug).
- Concurrent creation requests with same title (should generate unique slugs).
- Import ZIP with corrupted files.
- Import ZIP with unsupported file formats.
- Import ZIP with very large test case files.

---

## API Contract

### POST /problems

Create a new problem with metadata.

> **Important**: Only users with Coach or Admin roles can create problems. The minimum required field is `title`. All other fields can be added later via the Update Problem spec (`PUT /problems/{slug}`).

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |
| Idempotency-Key | string | No | Optional UUID for idempotent retries |

**Request Body**:

```json
{
  "title": "Sum of Two Numbers",
  "statement": "Given two integers a and b, return their sum.",
  "timeLimit": 2000,
  "memoryLimit": 256,
  "tags": ["math", "beginner"]
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| title | string | Yes | Problem title (normalized Unicode NFKC) |
| statement | string | No | Problem statement in LaTeX format |
| timeLimit | integer | No | Time limit in milliseconds (> 0, max: 300000) |
| memoryLimit | integer | No | Memory limit in MiB (> 0, max: 2048) |
| tags | string[] | No | Array of tags from system's predefined list (always optional) |

**Responses**:

#### 201 Created
Problem created successfully.

```json
{
  "slug": "sum-of-two-numbers",
  "title": "Sum of Two Numbers",
  "statement": "Given two integers a and b, return their sum.",
  "timeLimit": 2000,
  "memoryLimit": 256,
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
Validation error (missing title, invalid limits, invalid tags).

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
├── submissions/              # Optional solutions
│   └── accepted/
│       └── solution.cpp
├── output_validators/        # Optional custom checker
│   └── validator.cpp
└── input_validators/         # Optional input validator
    └── validator.cpp
```

**problem.yaml Example**:
```yaml
name: Sum of Two Numbers
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
Invalid ZIP file or missing required files.

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
- **FR-002**: The system MUST require only the `title` field to create a problem.
- **FR-003**: The system MUST auto-generate a unique slug from the title (lowercase, alphanumeric, hyphens only).
- **FR-004**: If the generated slug already exists, the system MUST append a numeric suffix to make it unique.
- **FR-005**: The slug MUST be limited to 100 characters maximum.
- **FR-006**: The system MUST set problem status to `DRAFT` and accessibility to `PRIVATE` when created.
- **FR-007**: The system MUST set the authenticated user as the problem author.
- **FR-008**: The system MUST normalize the title using Unicode NFKC normalization.
- **FR-009**: The system MUST validate timeLimit as positive integer ≤ 300000 milliseconds if provided.
- **FR-010**: The system MUST validate memoryLimit as positive integer ≤ 2048 MiB if provided.
- **FR-011**: The system MUST validate tags against the system's predefined tag list if provided.
- **FR-012**: Tags MUST always be optional (not required for creation or publication).

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
  - `slug` (string, unique, auto-generated, lowercase alphanumeric with hyphens, max 100 chars)
  - `title` (string, required, normalized NFKC)
  - `statement` (string, LaTeX format, nullable)
  - `timeLimit` (integer, milliseconds, nullable, max: 300000)
  - `memoryLimit` (integer, MiB, nullable, max: 2048)
  - `tags` (array of strings, always optional, from predefined list)
  - `status` (enum: `DRAFT` | `PUBLISHED`, default: `DRAFT`)
  - `accessibility` (enum: `PUBLIC` | `PRIVATE`, default: `PRIVATE`)
  - `authorId` (string, UUID, FK to User)
  - `modifierIds` (array of UUIDs, FK to User, empty on creation)
  - `testCasesFileKey` (string, nullable, reference to test cases ZIP)
  - `solutionFileKeys` (array of strings, references to solution files)
  - `checkerFileKey` (string, nullable, reference to checker file)
  - `validatorFileKey` (string, nullable, reference to validator file)
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp)

> **Problem Status** (publication state):
> - `DRAFT`: Problem is being built. Can have partial data. Can be modified. Not available for contests/practice.
> - `PUBLISHED`: Problem is complete and published. Cannot be modified (must unpublish first). Available for contests/practice.

> **Problem Accessibility** (who can add it to contests):
> - `PRIVATE`: Only the problem's modifiers (author + assigned modifiers) can add this problem to a contest. Default for all new problems.
> - `PUBLIC`: Any contest creator can add this problem to their contest.

### Slug Generation Algorithm

1. Normalize title using Unicode NFKC
2. Convert to lowercase
3. Replace spaces and underscores with hyphens
4. Remove all characters except alphanumeric and hyphens
5. Collapse multiple consecutive hyphens into single hyphen
6. Trim leading/trailing hyphens
7. Truncate to 100 characters
8. If slug already exists, append "-1", "-2", etc. until unique

**Examples**:
- "Sum of Two Numbers" → "sum-of-two-numbers"
- "¿Qué pasa?" → "que-pasa"
- "Problem   #1!!!" → "problem-1"
- "Sum of Two Numbers" (duplicate) → "sum-of-two-numbers-1"

### Tags

Tags are loaded from an external configuration file at application startup. They are always optional and not required for problem creation or publication.

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Coach and Admin users can create problems with only a title via `POST /problems` with HTTP 201.
- **SC-002**: Slug is auto-generated from title (lowercase, alphanumeric, hyphens).
- **SC-003**: Duplicate slugs are handled with numeric suffixes.
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

