# Feature Specification: Update Problem

**Created**: 2025-12-20

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Update problem metadata (Priority: P1)

As a Coach or Admin who has created a problem, I want to update the problem's metadata (title, statement, limits, tags) so that I can refine the problem before publishing.

**Why this priority**: Allows iterative refinement of problems. Essential for the incremental problem creation workflow.

**Independent Test**: This user story can be tested independently by consuming the `PUT /problems/{slug}` endpoint with valid authentication and partial updates, validating that only provided fields are modified.

**Acceptance Scenarios**:

1. **Scenario**: Successful metadata update
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they submit an update request with partial data (e.g., only statement)
   - **Then** the system updates only the provided fields
   - **And** returns the updated problem data

2. **Scenario**: Update multiple fields at once
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is the author or Admin
   - **When** they submit an update with title, statement, timeLimit, and memoryLimit
   - **Then** the system updates all provided fields
   - **And** `updatedAt` timestamp is refreshed

3. **Scenario**: Update PUBLISHED problem blocked
   - **Given** a problem exists with status `PUBLISHED`
   - **When** the author attempts to update metadata
   - **Then** the system rejects with 400 Bad Request (PROBLEM_IS_PUBLISHED)
   - **And** suggests unpublishing first to make changes

4. **Scenario**: Unauthorized update attempt
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is neither the author, an Admin, nor a modifier
   - **When** they attempt to update metadata
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

5. **Scenario**: Modifier updates metadata
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user has been assigned as a modifier for this problem
   - **When** they update the statement
   - **Then** the system accepts the update

9. **Scenario**: Update accessibility from PRIVATE to PUBLIC
   - **Given** a problem exists with status `DRAFT` and accessibility `PRIVATE`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they update accessibility to `PUBLIC`
   - **Then** the system updates the accessibility
   - **And** returns the updated problem data

6. **Scenario**: Invalid time/memory limit
   - **Given** a Coach or Admin is authenticated
   - **And** timeLimit or memoryLimit is zero, negative, or exceeds maximum allowed value
   - **When** they submit the update request
   - **Then** the system rejects with 400 Bad Request indicating invalid limits

7. **Scenario**: Invalid tags provided
   - **Given** a Coach or Admin is authenticated
   - **And** the request includes tags that are not in the system's predefined tag list
   - **When** they submit the update request
   - **Then** the system rejects with 400 Bad Request indicating invalid tags

8. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** an update request is submitted
   - **Then** the system rejects with 401 Unauthorized

---

### User Story 2 – Upload problem files (Priority: P1)

As a Coach or Admin who has created a problem, I want to upload test cases, solution, checker, and validator files so that the problem can be validated and eventually published.

**Why this priority**: Files are essential to complete problem setup. Direct upload to backend simplifies the flow and allows incremental file uploads.

**Independent Test**: This user story can be tested independently by uploading files to `POST /problems/{slug}/files` endpoint, validating that files are stored and associated with the problem.

**Acceptance Scenarios**:

1. **Scenario**: Upload test cases file
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they upload a test cases ZIP file
   - **Then** the system stores the file and associates it with the problem
   - **And** validates ZIP structure follows ICPC format

2. **Scenario**: Upload solution file
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they upload a solution file (Python, C++, or Java)
   - **Then** the system stores the file as a solution for validation

3. **Scenario**: Upload optional checker
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they upload a checker file
   - **Then** the system stores the file as the custom checker

4. **Scenario**: Upload optional validator
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they upload a validator file
   - **Then** the system stores the file as the input validator

5. **Scenario**: Replace existing file
   - **Given** a problem already has a test cases file uploaded
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they upload a new test cases file
   - **Then** the system replaces the previous file with the new one

6. **Scenario**: Upload to PUBLISHED problem
   - **Given** a problem exists with status `PUBLISHED`
   - **When** the author attempts to upload files
   - **Then** the system rejects with 400 Bad Request (PROBLEM_IS_PUBLISHED)
   - **And** suggests unpublishing first to make changes

7. **Scenario**: Unauthorized upload attempt
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is neither the author, an Admin, nor a modifier
   - **When** they attempt to upload files
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

---

### User Story 3 – Delete problem file (Priority: P2)

As a Coach or Admin, I want to delete a specific file from a problem so that I can remove incorrect or outdated files before uploading new ones.

**Why this priority**: Allows correction of file uploads without needing to replace. Secondary to upload functionality.

**Independent Test**: This user story can be tested independently by consuming the `DELETE /problems/{slug}/files/{fileType}` endpoint, validating that the file is removed from the problem.

**Acceptance Scenarios**:

1. **Scenario**: Delete test cases file
   - **Given** a problem exists with status `DRAFT`
   - **And** has a test cases file uploaded
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they delete the test cases file
   - **Then** the system removes the file association

2. **Scenario**: Delete specific solution file
   - **Given** a problem has multiple solution files
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they delete a specific solution by filename
   - **Then** only that solution is removed

3. **Scenario**: Delete file from PUBLISHED problem
   - **Given** a problem exists with status `PUBLISHED`
   - **When** the author attempts to delete a file
   - **Then** the system rejects with 400 Bad Request (PROBLEM_IS_PUBLISHED)

4. **Scenario**: Unauthorized delete attempt
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is neither the author, an Admin, nor a modifier
   - **When** they attempt to delete a file
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

---

### User Story 4 – Manage problem modifiers (Priority: P2)

As a problem author or Admin, I want to assign other users as modifiers so that they can help edit and maintain the problem.

**Why this priority**: Enables collaborative problem creation. Secondary to basic CRUD operations.

**Independent Test**: This user story can be tested independently by consuming the modifier management endpoints, validating that modifiers can be added/removed and have appropriate permissions.

**Acceptance Scenarios**:

1. **Scenario**: Add modifier by author
   - **Given** a problem exists
   - **And** the authenticated user is the author
   - **When** they add another user as a modifier by nickname
   - **Then** the modifier is associated with the problem
   - **And** the modifier can now edit the problem

2. **Scenario**: Add modifier by Admin
   - **Given** a problem exists
   - **And** the authenticated user is an Admin (not the author)
   - **When** they add a user as a modifier
   - **Then** the modifier is associated with the problem

3. **Scenario**: Remove modifier
   - **Given** a problem has a modifier assigned
   - **And** the authenticated user is the author or Admin
   - **When** they remove the modifier
   - **Then** the modifier no longer has edit permissions

4. **Scenario**: Modifier cannot add other modifiers
   - **Given** a problem exists
   - **And** the authenticated user is a modifier (not author/Admin)
   - **When** they attempt to add another modifier
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

5. **Scenario**: Add non-existent user as modifier
   - **Given** a problem exists
   - **And** the authenticated user is the author
   - **When** they attempt to add a non-existent user as modifier
   - **Then** the system rejects with 400 Bad Request (USER_NOT_FOUND)

6. **Scenario**: Add already existing modifier
   - **Given** a problem already has user X as modifier
   - **When** the author attempts to add user X again
   - **Then** the system rejects with 400 Bad Request (ALREADY_MODIFIER)

7. **Scenario**: List modifiers
   - **Given** a problem exists with modifiers assigned
   - **And** the authenticated user is the author, Admin, or modifier
   - **When** they request the list of modifiers
   - **Then** the system returns all modifiers with their nickname and name

---

### Edge Cases

- Concurrent update requests for the same problem.
- Update title to something that conflicts with existing slug (slug is NOT regenerated on update).
- File size limits exceeded.
- Network interruption during file uploads.
- Invalid ZIP structure in test cases file (validation on upload).
- Modifier's account is deactivated while they have modifier permissions.
- Upload file with same name as existing (replace behavior).
- Delete last solution file (allowed, but publish will fail).

---

## API Contract

### PUT /problems/{slug}

Update problem metadata.

> **Important**: Only the problem author, Admin, or modifier can update. Problem must be in `DRAFT` status. Use partial updates - only provided fields are modified. The slug is NOT regenerated if title changes.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Request Body** (all fields optional, only provided fields are updated):

```json
{
  "title": "Sum of Two Numbers - Updated",
  "statement": "Updated statement in LaTeX...",
  "timeLimit": 3000,
  "memoryLimit": 512,
  "tags": ["math", "beginner", "implementation"],
  "accessibility": "PUBLIC"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| title | string | No | Problem title (normalized Unicode NFKC) |
| statement | string | No | Problem statement in LaTeX format |
| timeLimit | integer | No | Time limit in milliseconds (> 0, max: 300000) |
| memoryLimit | integer | No | Memory limit in MiB (> 0, max: 2048) |
| tags | string[] | No | Array of tags from system's predefined list |
| accessibility | string | No | Problem accessibility: `PUBLIC` or `PRIVATE` |

**Responses**:

#### 200 OK
Problem updated successfully.

```json
{
  "slug": "sum-of-two-numbers",
  "title": "Sum of Two Numbers - Updated",
  "statement": "Updated statement in LaTeX...",
  "timeLimit": 3000,
  "memoryLimit": 512,
  "tags": ["math", "beginner", "implementation"],
  "status": "DRAFT",
  "accessibility": "PUBLIC",
  "author": {
    "nickname": "coach_john",
    "name": "John Smith"
  },
  "modifiers": [
    {
      "nickname": "coach_jane",
      "name": "Jane Doe"
    }
  ],
  "files": {
    "testCases": true,
    "solutions": ["solution.cpp"],
    "checker": false,
    "validator": false
  },
  "createdAt": "2025-12-20T10:00:00Z",
  "updatedAt": "2025-12-20T11:30:00Z"
}
```

#### 400 Bad Request
Validation error or problem is PUBLISHED.

```json
{
  "error": "PROBLEM_IS_PUBLISHED",
  "message": "Cannot update a published problem. Unpublish first to make changes."
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

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission to update this problem.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only the problem author, Admin, or assigned modifiers can update this problem"
}
```

#### 404 Not Found
Problem not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Problem not found"
}
```

---

### POST /problems/{slug}/files

Upload files for a problem (test cases, solution, checker, validator).

> **Important**: Only the problem author, Admin, or modifier can upload files. Problem must be in `DRAFT` status. Files are uploaded directly to the backend. Use multipart/form-data.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | multipart/form-data |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Form Fields**:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| fileType | string | Yes | Type of file: `testCases`, `solution`, `checker`, `validator` |
| file | file | Yes | The file to upload |

**File Requirements**:

| File Type | Format | Max Size | Description |
|-----------|--------|----------|-------------|
| testCases | ZIP | 200 MB | ICPC format ZIP with data/sample/ and data/secret/ |
| solution | .py, .cpp, .java | 10 MB | Solution that should pass all test cases |
| checker | .py, .cpp | 10 MB | Custom output checker (optional) |
| validator | .py, .cpp | 10 MB | Input validator (optional) |

**Test Cases ZIP Structure (ICPC Format)**:
```
testcases.zip/
├── data/
│   ├── sample/           # Example cases (shown to users)
│   │   ├── 1.in
│   │   ├── 1.ans
│   │   ├── 2.in
│   │   └── 2.ans
│   └── secret/           # Hidden cases (for judging)
│       ├── 01.in
│       ├── 01.ans
│       ├── 02.in
│       └── 02.ans
```

**Responses**:

#### 200 OK
File uploaded successfully.

```json
{
  "message": "File uploaded successfully",
  "fileType": "testCases",
  "fileName": "testcases.zip",
  "files": {
    "testCases": true,
    "solutions": ["solution.cpp"],
    "checker": false,
    "validator": false
  }
}
```

#### 400 Bad Request
Invalid file, problem is PUBLISHED, or file too large.

```json
{
  "error": "INVALID_FILE_TYPE",
  "message": "Invalid file type. Allowed: testCases, solution, checker, validator"
}
```

```json
{
  "error": "PROBLEM_IS_PUBLISHED",
  "message": "Cannot upload files to a published problem. Unpublish first."
}
```

```json
{
  "error": "FILE_TOO_LARGE",
  "message": "File exceeds maximum size of 200 MB for test cases"
}
```

#### 401 Unauthorized
Authentication failed.

#### 403 Forbidden
User does not have permission.

#### 404 Not Found
Problem not found.

---

### DELETE /problems/{slug}/files/{fileType}

Delete a specific file from a problem.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |
| fileType | string | Yes | Type of file to delete: `testCases`, `solution`, `checker`, `validator` |

**Query Parameters** (for solution only):

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| fileName | string | No | Specific solution filename to delete (if multiple solutions exist) |

**Responses**:

#### 200 OK
File deleted successfully.

```json
{
  "message": "File deleted successfully",
  "fileType": "checker",
  "files": {
    "testCases": true,
    "solutions": ["solution.cpp"],
    "checker": false,
    "validator": false
  }
}
```

#### 400 Bad Request
Problem is PUBLISHED.

```json
{
  "error": "PROBLEM_IS_PUBLISHED",
  "message": "Cannot delete files from a published problem. Unpublish first."
}
```

#### 401 Unauthorized
Authentication failed.

#### 403 Forbidden
User does not have permission.

#### 404 Not Found
Problem or file not found.

---

### POST /problems/{slug}/modifiers

Add a modifier to a problem.

> **Important**: Only the problem author or Admin can add modifiers. Modifiers can edit the problem but cannot add other modifiers or delete the problem.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | application/json |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Request Body**:

```json
{
  "userNickname": "coach_jane"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| userNickname | string | Yes | Nickname of the user to add as modifier |

**Responses**:

#### 200 OK
Modifier added successfully.

```json
{
  "message": "Modifier added successfully",
  "modifiers": [
    {
      "nickname": "coach_jane",
      "name": "Jane Doe"
    }
  ]
}
```

#### 400 Bad Request
User is already a modifier or user not found.

```json
{
  "error": "ALREADY_MODIFIER",
  "message": "User is already a modifier for this problem"
}
```

```json
{
  "error": "USER_NOT_FOUND",
  "message": "User with nickname 'coach_jane' not found"
}
```

#### 401 Unauthorized
Authentication failed.

#### 403 Forbidden
Only author or Admin can add modifiers.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only the problem author or Admin can add modifiers"
}
```

#### 404 Not Found
Problem not found.

---

### DELETE /problems/{slug}/modifiers/{nickname}

Remove a modifier from a problem.

> **Important**: Only the problem author or Admin can remove modifiers.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |
| nickname | string | Yes | Nickname of the modifier to remove |

**Responses**:

#### 200 OK
Modifier removed successfully.

```json
{
  "message": "Modifier removed successfully",
  "modifiers": []
}
```

#### 401 Unauthorized
Authentication failed.

#### 403 Forbidden
Only author or Admin can remove modifiers.

#### 404 Not Found
Problem or modifier not found.

---

### GET /problems/{slug}/modifiers

List all modifiers for a problem.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Responses**:

#### 200 OK
List of modifiers.

```json
{
  "modifiers": [
    {
      "nickname": "coach_jane",
      "name": "Jane Doe"
    },
    {
      "nickname": "coach_bob",
      "name": "Bob Smith"
    }
  ]
}
```

#### 401 Unauthorized
Authentication failed.

#### 404 Not Found
Problem not found.

---

## Requirements *(mandatory)*

### Functional Requirements

**Metadata Updates**
- **FR-001**: The system MUST allow partial updates to problem metadata (only provided fields are modified).
- **FR-002**: The system MUST only allow updates to problems with status `DRAFT`.
- **FR-003**: The system MUST only allow the problem author, Admin, or assigned modifiers to update a problem.
- **FR-004**: The system MUST NOT regenerate the slug when title is updated.
- **FR-005**: The system MUST validate timeLimit as positive integer ≤ 300000 milliseconds if provided.
- **FR-006**: The system MUST validate memoryLimit as positive integer ≤ 2048 MiB if provided.
- **FR-007**: The system MUST validate tags against the system's predefined tag list if provided.
- **FR-008**: Tags MUST always be optional.
- **FR-008b**: The system MUST allow updating accessibility (`PUBLIC` or `PRIVATE`) by the author, Admin, or assigned modifiers.

**File Uploads**
- **FR-009**: The system MUST accept direct file uploads via multipart/form-data.
- **FR-010**: The system MUST validate test cases ZIP follows ICPC format structure on upload.
- **FR-011**: The system MUST accept solution files in Python (.py), C++ (.cpp), or Java (.java).
- **FR-012**: The system MUST accept checker and validator files in Python or C++.
- **FR-013**: The system MUST enforce file size limits (200 MB for test cases, 10 MB for others).
- **FR-014**: The system MUST allow replacing existing files with new uploads.
- **FR-015**: The system MUST allow multiple solution files to be uploaded.
- **FR-016**: The system MUST only allow file uploads when problem status is `DRAFT`.

**File Deletion**
- **FR-017**: The system MUST allow deleting individual files from problems.
- **FR-018**: The system MUST only allow file deletion when problem status is `DRAFT`.
- **FR-019**: For solutions, the system MUST allow specifying which solution to delete by filename.

**Modifiers**
- **FR-020**: The system MUST allow the problem author or Admin to assign other users as modifiers.
- **FR-021**: Modifiers MUST have permission to update metadata, upload/delete files.
- **FR-022**: Modifiers MUST NOT be able to assign other modifiers (only author/Admin can).
- **FR-023**: Modifiers MUST NOT be able to delete the problem (only author/Admin can).
- **FR-024**: The system MUST allow listing all modifiers for a problem.
- **FR-025**: The system MUST allow removing modifiers by the author or Admin.

**General**
- **FR-026**: The system MUST NOT return internal IDs in any response.
- **FR-027**: The system MUST update the `updatedAt` timestamp on any modification.
- **FR-028**: The system MUST return validation errors with consistent structure.

### Key Entities

Referenced from Create Problem spec:

- **Problem**: Represents a programming problem.  
  Key attributes:
  - `slug` (string, unique, auto-generated, lowercase alphanumeric with hyphens)
  - `title` (string, required)
  - `statement` (string, LaTeX format, nullable)
  - `timeLimit` (integer, milliseconds, nullable, max: 300000)
  - `memoryLimit` (integer, MiB, nullable, max: 2048)
  - `tags` (array of strings, always optional, from predefined list)
  - `status` (enum: `DRAFT` | `PUBLISHED`)
  - `accessibility` (enum: `PUBLIC` | `PRIVATE`, default: `PRIVATE`)
  - `authorId` (string, UUID, FK to User)
  - `modifierIds` (array of UUIDs, FK to User, users with edit permissions)
  - `testCasesFileKey` (string, nullable, reference to test cases ZIP)
  - `solutionFileKeys` (array of strings, references to solution files)
  - `checkerFileKey` (string, nullable, reference to checker file)
  - `validatorFileKey` (string, nullable, reference to validator file)
  - `createdAt` (timestamp)
  - `updatedAt` (timestamp)

> **Problem Status** (publication state):
> - `DRAFT`: Problem is being built. Can have partial data. Can be updated.
> - `PUBLISHED`: Problem is complete and published. Cannot be modified (must unpublish first via Publish Problem spec).

> **Problem Accessibility** (who can add it to contests):
> - `PRIVATE`: Only the problem's modifiers (author + assigned modifiers) can add this problem to a contest. Default for all new problems.
> - `PUBLIC`: Any contest creator can add this problem to their contest.

### Supported File Types

| File Type | Extensions | Max Size | Required for Publication |
|-----------|------------|----------|-------------------------|
| testCases | .zip | 200 MB | ✅ Yes |
| solution | .py, .cpp, .java | 10 MB | ✅ Yes (at least 1) |
| checker | .py, .cpp | 10 MB | ❌ No (default: exact match) |
| validator | .py, .cpp | 10 MB | ❌ No |

### Permission Matrix

| Action | Author | Admin | Modifier | Contestant |
|--------|--------|-------|----------|------------|
| Update metadata | ✅ | ✅ | ✅ | ❌ |
| Update accessibility | ✅ | ✅ | ✅ | ❌ |
| Upload files | ✅ | ✅ | ✅ | ❌ |
| Delete files | ✅ | ✅ | ✅ | ❌ |
| Add modifier | ✅ | ✅ | ❌ | ❌ |
| Remove modifier | ✅ | ✅ | ❌ | ❌ |
| List modifiers | ✅ | ✅ | ✅ | ❌ |
| Delete problem | ✅ | ✅ | ❌ | ❌ |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Problem metadata can be updated via `PUT /problems/{slug}` with HTTP 200.
- **SC-002**: Only provided fields are modified during update (partial updates work).
- **SC-003**: Updates are blocked for problems with status `PUBLISHED` (HTTP 400).
- **SC-004**: Files can be uploaded via `POST /problems/{slug}/files` with HTTP 200.
- **SC-005**: Files can be deleted via `DELETE /problems/{slug}/files/{fileType}` with HTTP 200.
- **SC-006**: Test cases ZIP structure is validated against ICPC format on upload.
- **SC-007**: Only problem author, Admin, or assigned modifiers can modify problems.
- **SC-008**: Modifiers can be added/removed only by the author or Admin.
- **SC-009**: Modifiers can edit but cannot add other modifiers or delete the problem.
- **SC-010**: Contestant users attempting operations receive HTTP 403.
- **SC-011**: No internal IDs are returned in any response.
- **SC-012**: `updatedAt` is refreshed on every modification.

---

## Optional Notes

- **Slug immutability**: The slug is generated once at creation and never changes, even if title is updated.
- **File replacement**: Uploading a file of the same type replaces the existing file.
- **PUBLISHED state**: To modify a PUBLISHED problem, it must first be unpublished via the Publish Problem spec.
- **Related specs**:
  - Create Problem: Initial problem creation
  - Publish Problem: Publishing and unpublishing problems
  - Delete Problem: Removing problems (to be defined)

