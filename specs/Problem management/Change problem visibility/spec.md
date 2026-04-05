# Feature Specification: Change Problem Visibility

**Created**: 2025-12-20

> This spec covers two aspects of problem visibility:
> - **Status** (`DRAFT` ↔ `PUBLISHED`): Controls whether the problem can be modified and used in contests
> - **Accessibility** (`PRIVATE` ↔ `PUBLIC`): Controls who can add the problem to contests

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Publish problem (Priority: P1)

As a Coach or Admin with a complete problem, I want to publish it so that it becomes `PUBLISHED` and available for use in contests and practice sessions.

**Why this priority**: Publishing is the final step that makes problems available to contestants. The validation ensures problem quality before publication.

**Independent Test**: This user story can be tested independently by consuming the `POST /problems/p/{slug}/publish` endpoint for a complete problem, validating that status changes to `PUBLISHED` after successful validation.

**Acceptance Scenarios**:

1. **Scenario**: Successful publication
   - **Given** a problem exists with status `DRAFT`
   - **And** has all required data (title, statement, timeLimit, memoryLimit, test cases, at least one solution)
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** publish is triggered
   - **Then** the system validates the problem:
     - Compiles checker/validator if provided
     - Executes solution against all test cases
     - Verifies solution produces correct output
   - **And** if validation passes, changes status to `PUBLISHED`
   - **And** returns success response with validation logs

2. **Scenario**: Publication fails - missing required fields
   - **Given** a problem exists with status `DRAFT`
   - **And** is missing required fields (e.g., statement, test cases)
   - **When** publish is triggered
   - **Then** the system rejects with 400 Bad Request
   - **And** returns detailed validation logs indicating which fields are missing

3. **Scenario**: Publication fails - solution doesn't pass test cases
   - **Given** a problem has all required fields
   - **And** the solution fails one or more test cases
   - **When** publish is triggered
   - **Then** the system rejects with 400 Bad Request
   - **And** returns detailed logs showing which test cases failed and why

4. **Scenario**: Publication fails - checker compilation error
   - **Given** a problem has a custom checker that doesn't compile
   - **When** publish is triggered
   - **Then** the system rejects with 400 Bad Request
   - **And** returns detailed compilation error logs

5. **Scenario**: Publication fails - invalid ZIP structure
   - **Given** a problem has test cases ZIP with invalid ICPC format
   - **When** publish is triggered
   - **Then** the system rejects with 400 Bad Request
   - **And** returns detailed logs indicating the structure issues

6. **Scenario**: Publication fails - solution timeout
   - **Given** a problem has all required fields
   - **And** the solution exceeds the time limit on one or more test cases
   - **When** publish is triggered
   - **Then** the system rejects with 400 Bad Request
   - **And** returns detailed logs indicating which test cases timed out

7. **Scenario**: Publication fails - validator rejects test input
   - **Given** a problem has a validator uploaded
   - **And** some test inputs don't pass validation
   - **When** publish is triggered
   - **Then** the system rejects with 400 Bad Request
   - **And** returns logs indicating which inputs failed validation

8. **Scenario**: Publish already PUBLISHED problem
   - **Given** a problem exists with status `PUBLISHED`
   - **When** publish is attempted again
   - **Then** the system rejects with 409 Conflict (ALREADY_PUBLISHED)

9. **Scenario**: Unauthorized publish attempt
   - **Given** a problem exists with status `DRAFT`
   - **And** the authenticated user is neither the author, an Admin, nor a modifier
   - **When** they attempt to publish
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

10. **Scenario**: Unauthenticated request
    - **Given** the request does not include valid authentication credentials
    - **When** a publish request is submitted
    - **Then** the system rejects with 401 Unauthorized

---

### User Story 2 – Unpublish problem (Priority: P2)

As a Coach or Admin, I want to unpublish a problem so that I can make changes to it before republishing.

**Why this priority**: Allows corrections and updates to published problems. Lower priority as it's not part of the initial creation flow.

**Independent Test**: This user story can be tested independently by consuming the `POST /problems/p/{slug}/unpublish` endpoint, validating that status changes from `PUBLISHED` to `DRAFT`.

**Acceptance Scenarios**:

1. **Scenario**: Successful unpublish
   - **Given** a problem exists with status `PUBLISHED`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** unpublish is triggered
   - **Then** the system changes status to `DRAFT`
   - **And** returns success response

2. **Scenario**: Unpublish DRAFT problem
   - **Given** a problem exists with status `DRAFT`
   - **When** unpublish is attempted
   - **Then** the system rejects with 409 Conflict (ALREADY_DRAFT)

3. **Scenario**: Unauthorized unpublish attempt
   - **Given** a problem exists with status `PUBLISHED`
   - **And** the authenticated user is neither the author, an Admin, nor a modifier
   - **When** they attempt to unpublish
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

4. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** an unpublish request is submitted
   - **Then** the system rejects with 401 Unauthorized

5. **Scenario**: Problem not found
   - **Given** the slug does not match any existing problem
   - **When** unpublish is attempted
   - **Then** the system rejects with 404 Not Found

6. **Scenario**: Unpublish problem in active contest
   - **Given** a problem exists with status `PUBLISHED`
   - **And** the problem is part of an ongoing/active contest
   - **When** unpublish is attempted
   - **Then** the system rejects with 409 Conflict (PROBLEM_IN_ACTIVE_CONTEST)
   - **And** returns an error message indicating the problem is currently used in an active contest

---

### User Story 3 – Change problem accessibility (Priority: P2)

As a Coach or Admin, I want to change a problem's accessibility between `PRIVATE` and `PUBLIC` so that I can control who can add the problem to contests, regardless of the problem's publication status.

**Why this priority**: Allows flexible control over problem distribution. Can be done independently of the publish/unpublish flow.

**Independent Test**: This user story can be tested independently by consuming the `PATCH /problems/p/{slug}/accessibility` endpoint, validating that accessibility changes without affecting the problem's status.

**Acceptance Scenarios**:

1. **Scenario**: Change accessibility from PRIVATE to PUBLIC (DRAFT problem)
   - **Given** a problem exists with status `DRAFT` and accessibility `PRIVATE`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they change accessibility to `PUBLIC`
   - **Then** the system updates the accessibility
   - **And** the problem status remains `DRAFT`
   - **And** returns success response with updated accessibility

2. **Scenario**: Change accessibility from PRIVATE to PUBLIC (PUBLISHED problem)
   - **Given** a problem exists with status `PUBLISHED` and accessibility `PRIVATE`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they change accessibility to `PUBLIC`
   - **Then** the system updates the accessibility
   - **And** the problem status remains `PUBLISHED`
   - **And** any contest creator can now add this problem to their contests

3. **Scenario**: Change accessibility from PUBLIC to PRIVATE (DRAFT problem)
   - **Given** a problem exists with status `DRAFT` and accessibility `PUBLIC`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they change accessibility to `PRIVATE`
   - **Then** the system updates the accessibility
   - **And** the problem status remains `DRAFT`

4. **Scenario**: Change accessibility from PUBLIC to PRIVATE (PUBLISHED problem)
   - **Given** a problem exists with status `PUBLISHED` and accessibility `PUBLIC`
   - **And** the authenticated user is the author, Admin, or a modifier
   - **When** they change accessibility to `PRIVATE`
   - **Then** the system updates the accessibility
   - **And** the problem status remains `PUBLISHED`
   - **And** only modifiers can now add this problem to new contests
   - **And** existing contest associations are NOT affected
   - **And** existing submissions continue to execute normally

5. **Scenario**: Problem in contest remains accessible after changing to PRIVATE
   - **Given** a problem exists with accessibility `PUBLIC` and status `PUBLISHED`
   - **And** the problem is added to contest C1 (by a non-modifier)
   - **And** user U1 is registered to contest C1
   - **When** the problem's accessibility is changed to `PRIVATE`
   - **Then** user U1 can still submit solutions to the problem within contest C1
   - **And** user U1 cannot access the problem outside of contest C1 (e.g., in public problem listings)
   - **And** the problem cannot be added to new contests by non-modifiers

6. **Scenario**: Change accessibility to same value (no-op)
   - **Given** a problem exists with accessibility `PUBLIC`
   - **When** they change accessibility to `PUBLIC`
   - **Then** the system returns success (idempotent operation)
   - **And** no changes are made

7. **Scenario**: Unauthorized accessibility change attempt
   - **Given** a problem exists
   - **And** the authenticated user is neither the author, an Admin, nor a modifier
   - **When** they attempt to change accessibility
   - **Then** the system rejects with 403 Forbidden (INSUFFICIENT_PERMISSIONS)

8. **Scenario**: Unauthenticated request
   - **Given** the request does not include valid authentication credentials
   - **When** an accessibility change request is submitted
   - **Then** the system rejects with 401 Unauthorized

9. **Scenario**: Problem not found
   - **Given** the slug does not match any existing problem
   - **When** accessibility change is attempted
   - **Then** the system rejects with 404 Not Found

10. **Scenario**: Invalid accessibility value
    - **Given** a problem exists
    - **And** the authenticated user is authorized
    - **When** they submit an invalid accessibility value (e.g., "RESTRICTED")
    - **Then** the system rejects with 400 Bad Request (INVALID_ACCESSIBILITY)

---

### Edge Cases

**Publication:**
- Concurrent publish requests for the same problem.
- Solution timeout during validation (recommended max: 10 minutes total).
- Checker/validator compilation errors with non-ASCII error messages.
- Solution produces correct output but with trailing whitespace differences.
- Multiple solutions uploaded (all must pass all test cases).
- Very large number of test cases (e.g., 1000+ cases).
- Test case output is very large (memory considerations).
- Problem deletion while validation is in progress.
- Unpublish problem that's currently in use in an active contest.
- Network interruption during validation.
- Validation server unavailable.

**Accessibility:**
- Change accessibility while problem is being used in an active contest.
- Change from PUBLIC to PRIVATE when the problem is already added to contests by non-modifiers:
  - The problem is NOT removed from existing contests (association remains intact)
  - Registered participants can still submit solutions to the problem within those contests
  - Users lose access to the problem from outside the contest (cannot view it in public problem lists)
  - Only new contest additions are restricted to modifiers
- Concurrent accessibility change requests for the same problem.
- User submits to a problem that was PUBLIC when they started but became PRIVATE before submission completed (submission should succeed if user has access via contest registration).

---

## API Contract

### POST /problems/p/{slug}/publish

Validate and publish a problem, changing status from `DRAFT` to `PUBLISHED`.

> **Important**: This endpoint triggers full validation:
> - Verifies all required fields are present
> - Validates test cases ZIP structure (ICPC format)
> - Compiles checker/validator if provided
> - Runs validator against all test inputs (if validator exists)
> - Executes solution(s) against all test cases
> - Solution must produce correct output for all cases within time/memory limits

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | The unique slug of the problem |

**Required Fields for Publication**:

| Field | Description |
|-------|-------------|
| title | Problem title (always present) |
| statement | Problem statement in LaTeX format |
| timeLimit | Time limit in milliseconds |
| memoryLimit | Memory limit in MiB |
| testCases | Test cases ZIP file (ICPC format) |
| solution | At least one solution file |

**Optional Fields**:

| Field | Description |
|-------|-------------|
| tags | Array of tags (always optional) |
| checker | Custom output checker (default: exact match) |
| validator | Input validator |

**Responses**:

#### 200 OK
Problem published successfully.

```json
{
  "slug": "sum-of-two-numbers",
  "status": "PUBLISHED",
  "message": "Problem published successfully",
  "validationLogs": [
    "✓ Required fields validated",
    "✓ Test cases ZIP structure valid (5 sample, 20 secret cases)",
    "✓ Solution compiled successfully",
    "✓ Solution passed all 25 test cases",
    "✓ Problem is now PUBLISHED"
  ],
  "validationSummary": {
    "sampleCases": 5,
    "secretCases": 20,
    "solutionsTested": 1,
    "allPassed": true
  }
}
```

#### 400 Bad Request
Validation failed. Returns detailed logs of what failed.

**Missing required fields:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Problem validation failed",
  "validationLogs": [
    "✓ Title: Sum of Two Numbers",
    "✗ Statement: Missing (required)",
    "✓ Time limit: 2000ms",
    "✓ Memory limit: 256 MiB",
    "✗ Test cases: Not uploaded (required)",
    "✗ Solution: Not uploaded (required)"
  ],
  "missingFields": ["statement", "testCases", "solution"]
}
```

**Solution failed test cases:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Solution failed test cases",
  "validationLogs": [
    "✓ Required fields validated",
    "✓ Test cases ZIP structure valid",
    "✓ Solution compiled successfully",
    "✗ Solution failed test case: secret/05",
    "  Expected: 42",
    "  Got: 41",
    "✗ Solution failed test case: secret/12",
    "  Expected: 100",
    "  Got: Runtime Error (SIGSEGV)"
  ],
  "failedTestCases": [
    {
      "case": "secret/05",
      "verdict": "WRONG_ANSWER",
      "expected": "42",
      "actual": "41"
    },
    {
      "case": "secret/12",
      "status": "RUNTIME_EXCEPTION",
      "details": "SIGSEGV"
    }
  ]
}
```

**Solution timeout:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Solution exceeded time limit",
  "validationLogs": [
    "✓ Required fields validated",
    "✓ Test cases ZIP structure valid",
    "✓ Solution compiled successfully",
    "✗ Solution exceeded time limit on: secret/08",
    "  Time limit: 2000ms",
    "  Execution time: > 2000ms (killed)"
  ],
  "failedTestCases": [
    {
      "case": "secret/08",
      "verdict": "TIME_LIMIT_EXCEEDED",
      "timeLimit": 2000
    }
  ]
}
```

**Checker compilation failed:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Checker compilation failed",
  "validationLogs": [
    "✓ Required fields validated",
    "✓ Test cases ZIP structure valid",
    "✗ Checker compilation failed:",
    "  checker.cpp:15:10: error: 'strcmp' was not declared in this scope",
    "  checker.cpp:20:5: error: 'cout' was not declared in this scope"
  ],
  "compilationErrors": {
    "file": "checker.cpp",
    "errors": [
      "checker.cpp:15:10: error: 'strcmp' was not declared in this scope",
      "checker.cpp:20:5: error: 'cout' was not declared in this scope"
    ]
  }
}
```

**Invalid ZIP structure:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Invalid test cases structure",
  "validationLogs": [
    "✓ Required fields validated",
    "✗ Test cases ZIP structure invalid:",
    "  Missing 'data/' directory",
    "  Expected structure: data/sample/*.in, data/sample/*.ans, data/secret/*.in, data/secret/*.ans"
  ]
}
```

**Validator rejected inputs:**
```json
{
  "error": "VALIDATION_FAILED",
  "message": "Validator rejected test inputs",
  "validationLogs": [
    "✓ Required fields validated",
    "✓ Test cases ZIP structure valid",
    "✓ Validator compiled successfully",
    "✗ Validator rejected input: secret/03.in",
    "  Error: Value 1000001 exceeds constraint (max: 1000000)"
  ],
  "failedInputs": [
    {
      "file": "secret/03.in",
      "reason": "Value 1000001 exceeds constraint (max: 1000000)"
    }
  ]
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
  "message": "Only the problem author, Admin, or assigned modifiers can publish this problem"
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

#### 409 Conflict
Problem is already PUBLISHED.

```json
{
  "error": "ALREADY_PUBLISHED",
  "message": "Problem is already published"
}
```

---

### POST /problems/p/{slug}/unpublish

Unpublish a problem, changing status from `PUBLISHED` to `DRAFT`.

> **Important**: Unpublishing allows modifications to the problem. The problem will no longer be available for contests/practice until republished.

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
Problem unpublished successfully.

```json
{
  "slug": "sum-of-two-numbers",
  "status": "DRAFT",
  "message": "Problem unpublished successfully. You can now make changes."
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
  "message": "Only the problem author, Admin, or assigned modifiers can unpublish this problem"
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

#### 409 Conflict
Problem is already DRAFT or used in an active contest.

```json
{
  "error": "ALREADY_DRAFT",
  "message": "Problem is already unpublished"
}
```

```json
{
  "error": "PROBLEM_IN_ACTIVE_CONTEST",
  "message": "Cannot unpublish a problem that is currently being used in an active contest"
}
```

---

### PATCH /problems/p/{slug}/accessibility

Change a problem's accessibility between `PRIVATE` and `PUBLIC`.

> **Important**: This endpoint can be used regardless of the problem's status (`DRAFT` or `PUBLISHED`). It does NOT require unpublishing the problem first. Changing accessibility does not affect the problem's status.

> **Behavior when changing from PUBLIC to PRIVATE**:
> - Existing contest associations are NOT affected (problems already added to contests remain there)
> - Existing submissions continue to execute normally
> - Users who submitted to this problem retain access to their submissions
> - Only new contest additions are restricted to modifiers

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
  "accessibility": "PUBLIC"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| accessibility | string | Yes | New accessibility value: `PUBLIC` or `PRIVATE` |

**Responses**:

#### 200 OK
Accessibility changed successfully.

```json
{
  "slug": "sum-of-two-numbers",
  "accessibility": "PUBLIC",
  "status": "PUBLISHED",
  "message": "Problem accessibility changed to PUBLIC"
}
```

#### 400 Bad Request
Invalid accessibility value.

```json
{
  "error": "INVALID_ACCESSIBILITY",
  "message": "Invalid accessibility value. Allowed values: PUBLIC, PRIVATE"
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
  "message": "Only the problem author, Admin, or assigned modifiers can change this problem's accessibility"
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

## Requirements *(mandatory)*

### Functional Requirements

**Publication Validation**
- **FR-001**: The system MUST validate all required fields before publishing:
  - title (required, always present)
  - statement (required)
  - timeLimit (required)
  - memoryLimit (required)
  - testCases file (required)
  - At least one solution file (required)
- **FR-002**: The system MUST validate test cases ZIP follows ICPC format structure.
- **FR-003**: The system MUST compile checker if provided during publication.
- **FR-004**: The system MUST compile validator if provided during publication.
- **FR-005**: If validator exists, the system MUST run it against all test inputs.
- **FR-006**: The system MUST execute all solutions against all test cases.
- **FR-007**: The system MUST require all solutions to pass all test cases for successful publication.
- **FR-008**: The system MUST enforce time limits during solution execution.
- **FR-009**: The system MUST enforce memory limits during solution execution.
- **FR-010**: If no checker is provided, the system MUST use exact string comparison for output validation.

**Publication Results**
- **FR-011**: If validation fails, the system MUST return detailed logs indicating what failed.
- **FR-012**: Validation logs MUST include specific test case names that failed.
- **FR-013**: Validation logs MUST include expected vs actual output for wrong answers.
- **FR-014**: Validation logs MUST include compilation errors with line numbers.
- **FR-015**: The system MUST change status to `PUBLISHED` only after successful validation.
- **FR-016**: The system MUST NOT allow re-publishing an already `PUBLISHED` problem.

**Unpublication**
- **FR-017**: The system MUST allow changing status from `PUBLISHED` to `DRAFT`.
- **FR-018**: The system MUST NOT allow unpublishing an already `DRAFT` problem.

**Accessibility**
- **FR-019**: The system MUST allow changing accessibility between `PUBLIC` and `PRIVATE`.
- **FR-020**: The system MUST allow changing accessibility regardless of problem status (`DRAFT` or `PUBLISHED`).
- **FR-021**: Changing accessibility MUST NOT affect the problem's status.
- **FR-022**: When changing from `PUBLIC` to `PRIVATE`, existing contest associations MUST NOT be affected.
- **FR-023**: When changing from `PUBLIC` to `PRIVATE`, existing submissions MUST continue to execute normally.
- **FR-024**: When changing from `PUBLIC` to `PRIVATE`, users who submitted to this problem MUST retain access to their submissions.
- **FR-025**: When changing from `PUBLIC` to `PRIVATE`, registered participants in contests containing the problem MUST retain the ability to submit solutions within those contests.
- **FR-026**: The system MUST validate that accessibility is either `PUBLIC` or `PRIVATE`.

**Permissions**
- **FR-027**: The system MUST only allow the problem author, Admin, or assigned modifiers to publish.
- **FR-028**: The system MUST only allow the problem author, Admin, or assigned modifiers to unpublish.
- **FR-029**: The system MUST only allow the problem author, Admin, or assigned modifiers to change accessibility.

**General**
- **FR-030**: The system MUST NOT return internal IDs in any response.
- **FR-031**: The system MUST update the `updatedAt` timestamp on status or accessibility change.

### Key Entities

📝 **Please Refer to `README.md`**

For the canonical documentation of the `Problem` entity and its properties (including verification steps, accessibility meaning, etc), please refer to the `README.md` at the root of the Problem management directory.

### Validation Process

The publish endpoint triggers a comprehensive validation pipeline:

```
1. Check required fields
   ├─ title ✓ (always present)
   ├─ statement
   ├─ timeLimit
   ├─ memoryLimit
   ├─ testCases file
   └─ solution file(s)

2. Validate test cases structure (ICPC format)
   ├─ data/sample/*.in, *.ans
   └─ data/secret/*.in, *.ans

3. Compile checker (if provided)
   └─ Report compilation errors if any

4. Compile validator (if provided)
   └─ Report compilation errors if any

5. Run validator against all inputs (if validator exists)
   └─ Report invalid inputs if any

6. Compile solution(s)
   └─ Report compilation errors if any

7. Execute solution(s) against all test cases
   ├─ Check output correctness (using checker or exact match)
   ├─ Enforce time limit
   └─ Enforce memory limit

8. If all pass → status = PUBLISHED
   If any fail → return detailed error logs
```

### Permission Matrix

| Action | Author | Admin | Modifier | Contestant |
|--------|--------|-------|----------|------------|
| Publish | ✅ | ✅ | ✅ | ❌ |
| Unpublish | ✅ | ✅ | ✅ | ❌ |
| Change accessibility | ✅ | ✅ | ✅ | ❌ |

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

**Publication:**
- **SC-001**: Complete problems can be published via `POST /problems/p/{slug}/publish` with HTTP 200.
- **SC-002**: Publication validates all required fields.
- **SC-003**: Publication validates test cases ZIP structure (ICPC format).
- **SC-004**: Publication compiles and runs checker/validator if provided.
- **SC-005**: Publication executes solutions against all test cases.
- **SC-006**: All solutions must pass all test cases for successful publication.
- **SC-007**: Failed validation returns detailed logs with specific failures.
- **SC-008**: Successful publication changes status to `PUBLISHED`.
- **SC-009**: Already PUBLISHED problems return 409 Conflict on publish attempt.
- **SC-010**: Problems can be unpublished via `POST /problems/p/{slug}/unpublish` with HTTP 200.
- **SC-011**: Unpublishing changes status to `DRAFT`.
- **SC-012**: Already DRAFT problems return 409 Conflict on unpublish attempt.
- **SC-013**: Unpublishing active contest problems returns 409 Conflict on unpublish attempt.

**Accessibility:**
- **SC-014**: Problem accessibility can be changed via `PATCH /problems/p/{slug}/accessibility` with HTTP 200.
- **SC-014**: Accessibility can be changed for both `DRAFT` and `PUBLISHED` problems.
- **SC-015**: Changing accessibility does NOT affect problem status.
- **SC-016**: Invalid accessibility values return HTTP 400.
- **SC-017**: When changing from PUBLIC to PRIVATE, existing contest associations remain intact.
- **SC-018**: When changing from PUBLIC to PRIVATE, existing submissions continue to work.
- **SC-019**: When changing from PUBLIC to PRIVATE, registered contest participants can still submit to the problem within those contests.

**Permissions & General:**
- **SC-020**: Only author, Admin, or modifiers can publish/unpublish/change accessibility.
- **SC-021**: Contestants receive HTTP 403 on publish/unpublish/accessibility change attempts.
- **SC-022**: No internal IDs are returned in any response.

---

## Optional Notes

- **Validation timeout**: Recommended maximum total validation time of 10 minutes. If exceeded, validation should fail with appropriate error message.
- **Concurrent validation**: Consider queuing publish requests to avoid overloading validation servers.
- **Checker behavior**: If no checker is provided, exact string comparison (ignoring trailing whitespace on lines) is used.
- **Multiple solutions**: If multiple solutions are uploaded, all must pass all test cases.
- **Partial re-validation**: Future optimization could cache compilation results to speed up re-publication.
- **Accessibility vs Status**: These are independent dimensions:
  - **Status** (`DRAFT`/`PUBLISHED`) controls whether the problem can be modified and used in contests.
  - **Accessibility** (`PRIVATE`/`PUBLIC`) controls WHO can add the problem to contests.
  - A problem can be `PUBLISHED` + `PRIVATE` (only modifiers can add it to contests) or `PUBLISHED` + `PUBLIC` (anyone can add it).
- **Accessibility change is non-destructive**: Changing from `PUBLIC` to `PRIVATE`:
  - Does NOT remove the problem from existing contests
  - Does NOT invalidate existing submissions
  - Does NOT affect registered participants' ability to submit within contests
  - Only restricts who can ADD the problem to NEW contests (modifiers only)
  - Users lose public access to the problem (cannot view outside of contests they're registered to)
- **Related specs**:
  - Create Problem: Initial problem creation
  - Update Problem: Modifying metadata and files (includes accessibility change for DRAFT problems)
  - Submit Solution: Submission access rules based on problem accessibility
  - Re-judge: Triggered when problem is modified after being used in submissions


