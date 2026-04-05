# Feature Specification: Submit Solution

**Created**: 2026-01-03

## User Scenarios & Testing *(mandatory)*

### User Story 1 – User submits solution to problem (outside contest) (Priority: P1)

As a user, I want to submit a solution to a problem outside of a contest so that I can practice and improve my programming skills.

**Why this priority**: Submitting solutions is the core functionality of the platform. Users need to be able to practice on problems without being in a contest.

**Independent Test**: This user story can be tested independently by consuming the `POST /api/problems/{problemSlug}/submissions` endpoint with valid authentication, validating that the submission is created and queued for judging.

**Acceptance Scenarios**:

1. **Scenario**: Successful submission to PUBLIC problem
   * **Given** a problem exists with status PUBLISHED and accessibility PUBLIC
   * **And** the authenticated user is any user (Member, Lead, Admin, Contestant)
   * **When** they submit a solution file with valid language and compiler
   * **Then** the system creates the submission with status PENDING
   * **And** stores the source code file
   * **And** calculates and stores the file hash
   * **And** queues the submission for judging
   * **And** returns the submission data

2. **Scenario**: Successful submission to PRIVATE problem by modifier
   * **Given** a problem exists with status PUBLISHED and accessibility PRIVATE
   * **And** the authenticated user is the problem author or a modifier
   * **When** they submit a solution file
   * **Then** the system creates the submission successfully
   * **And** returns the submission data

3. **Scenario**: Submission fails - problem not PUBLISHED
   * **Given** a problem exists with status DRAFT
   * **And** the authenticated user is authenticated
   * **When** they attempt to submit a solution
   * **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_PUBLISHED)
   * **And** indicates that only PUBLISHED problems can receive submissions

4. **Scenario**: Submission fails - PRIVATE problem, user not modifier
   * **Given** a problem exists with status PUBLISHED and accessibility PRIVATE
   * **And** the authenticated user is NOT the author or a modifier
   * **When** they attempt to submit a solution
   * **Then** the system rejects with 403 Forbidden (PROBLEM_NOT_ACCESSIBLE)
   * **And** indicates that only modifiers can submit to PRIVATE problems

5. **Scenario**: Submission fails - duplicate file (same hash, same user, same problem)
   * **Given** a submission exists with a specific file hash for the authenticated user and problem
   * **And** the user attempts to submit the same file again
   * **When** they submit the solution
   * **Then** the system rejects with 409 Conflict (DUPLICATE_SUBMISSION)
   * **And** indicates that this file was already submitted

6. **Scenario**: Submission fails - rate limit exceeded
   * **Given** the authenticated user submitted a solution to the same problem less than 1 second ago
   * **When** they attempt to submit again
   * **Then** the system rejects with 429 Too Many Requests (RATE_LIMIT_EXCEEDED)
   * **And** indicates the rate limit restriction

7. **Scenario**: Submission fails - file too large
   * **Given** the authenticated user attempts to submit a file larger than 1MB
   * **When** they submit the solution
   * **Then** the system rejects with 400 Bad Request (FILE_TOO_LARGE)
   * **And** indicates the maximum file size allowed

8. **Scenario**: Submission fails - invalid compiler for extension
   * **Given** the user submits a .cpp file but selects Python compiler
   * **When** they submit the solution
   * **Then** the system rejects with 400 Bad Request (COMPILER_MISMATCH)
   * **And** indicates that the compiler must match the file extension

9. **Scenario**: Submission fails - problem not found
   * **Given** no problem exists with the provided slug
   * **When** the user attempts to submit
   * **Then** the system rejects with 404 Not Found
   * **And** indicates that the problem does not exist

---

### User Story 2 – Registered user submits solution in contest (Priority: P1)

As a registered user in a contest, I want to submit a solution to a problem within the contest so that I can compete and appear in the standings.

**Why this priority**: Contest submissions are essential for competitive programming. Users need to submit solutions during contests to participate and be ranked.

**Acceptance Scenarios**:

1. **Scenario**: Successful submission during ACTIVE contest
   * **Given** a contest exists with status ACTIVE
   * **And** the authenticated user is registered to the contest
   * **And** the problem exists and is part of the contest
   * **And** the problem has status PUBLISHED
   * **When** they submit a solution file
   * **Then** the system creates the submission with status PENDING
   * **And** associates the submission with the contest
   * **And** stores the source code file in contest-specific path
   * **And** queues the submission for judging
   * **And** returns the submission data including contest info

2. **Scenario**: Successful submission during postcompetition phase
   * **Given** a contest exists with status FINISHED
   * **And** the contest has `enablePostContest = true`
   * **And** the authenticated user is registered to the contest
   * **And** the problem exists and is part of the contest
   * **When** they submit a solution after endTime
   * **Then** the system creates the submission with status PENDING
   * **And** associates the submission with the contest
   * **And** the submission does NOT affect standings (submittedAt > endTime)
   * **And** returns the submission data

3. **Scenario**: Submission fails - user not registered
   * **Given** a contest exists with status ACTIVE
   * **And** the authenticated user is NOT registered to the contest
   * **When** they attempt to submit a solution
   * **Then** the system rejects with 403 Forbidden (NOT_REGISTERED)
   * **And** indicates that registration is required

4. **Scenario**: Submission fails - problem not in contest
   * **Given** a contest exists with status ACTIVE
   * **And** the authenticated user is registered
   * **And** the problem exists but is NOT part of the contest
   * **When** they attempt to submit
   * **Then** the system rejects with 400 Bad Request (PROBLEM_NOT_IN_CONTEST)
   * **And** indicates that the problem is not part of this contest

5. **Scenario**: Submission fails - contest SCHEDULED
   * **Given** a contest exists with status SCHEDULED
   * **And** the authenticated user is registered
   * **When** they attempt to submit
   * **Then** the system rejects with 400 Bad Request (CONTEST_NOT_STARTED)
   * **And** indicates that submissions are only allowed during ACTIVE contests or postcompetition

6. **Scenario**: Submission fails - contest FINISHED without postcompetition
   * **Given** a contest exists with status FINISHED
   * **And** the contest has `enablePostContest = false`
   * **And** the authenticated user is registered
   * **When** they attempt to submit
   * **Then** the system rejects with 400 Bad Request (CONTEST_FINISHED)
   * **And** indicates that the contest has ended and postcompetition is not enabled

7. **Scenario**: Submission fails - contest not found
   * **Given** no contest exists with the provided contestId
   * **When** the user attempts to submit
   * **Then** the system rejects with 404 Not Found
   * **And** indicates that the contest does not exist

---

### Edge Cases

* Concurrent submissions from the same user to the same problem (rate limiting).
* File upload interrupted mid-transfer.
* Submission created but file storage fails (rollback needed).
* Problem becomes DRAFT after submission is created:
  * Submissions with status RUNNING continue execution until completion
  * Submissions with status PENDING are paused/blocked
  * No new submissions are accepted
  * When problem returns to PUBLISHED, paused submissions can be rejudged
* Problem test cases, checker, or validator are updated:
  * All RUNNING submissions must complete first
  * PENDING submissions are paused
  * When problem returns to PUBLISHED, paused submissions can be rejudged
* Problem changes from PUBLIC to PRIVATE:
  * Submissions continue executing normally
  * Users cannot access the problem but can access their own submissions
* Contest ends while submission is being judged (standing should not update if submittedAt > endTime).
* Multiple submissions with same content but different file names (hash should detect duplicate).
* Very large number of submissions queued (system should handle load).
* Compiler version changes after submission is created (should use compiler at submission time).
* Problem limits change after submission is created (should use limits at submission time).

---

## API Contract

### POST /api/problems/{problemSlug}/submissions

Submit a solution to a problem outside of a contest.

> **Important**: 
> - Only PUBLISHED problems can receive submissions
> - PUBLIC problems: any authenticated user can submit
> - PRIVATE problems: only modifiers (author + assigned modifiers) can submit
> - Rate limit: 1 second between submissions from same user to same problem
> - Duplicate detection: same file hash from same user to same problem is rejected
> - File size limit: 1MB (configurable via Virtual Object)
> - Compiler must match file extension

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | multipart/form-data |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| problemSlug | string | Yes | The slug of the problem |

**Request Body** (multipart/form-data):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| file | file | Yes | Source code file (.cpp, .java, .py) |
| language | string | Yes | Language identifier (cpp20, java17, python310) |
| compiler | string | Yes | Compiler identifier (g++, javac, py) |

**Responses**:

#### 201 Created
Submission created successfully and queued for judging.

```json
{
  "id": "s1d2e3f4-g5h6-7890-1234-567890123456",
  "status": "PENDING",
  "submittedAt": "2026-01-10T14:30:00Z",
  "problem": {
    "slug": "sum-of-two-numbers",
    "title": "Sum of Two Numbers"
  },
  "language": "cpp20",
  "compiler": "g++",
  "fileSize": 2048,
  "fileHash": "a1b2c3d4e5f6..."
}
```

#### 400 Bad Request
Validation error.

```json
{
  "error": "PROBLEM_NOT_PUBLISHED",
  "message": "Only PUBLISHED problems can receive submissions"
}
```

```json
{
  "error": "FILE_TOO_LARGE",
  "message": "File size exceeds maximum allowed size of 1MB"
}
```

```json
{
  "error": "COMPILER_MISMATCH",
  "message": "Compiler does not match file extension"
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
User doesn't have permission.

```json
{
  "error": "PROBLEM_NOT_ACCESSIBLE",
  "message": "Only modifiers can submit to PRIVATE problems"
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
Duplicate submission.

```json
{
  "error": "DUPLICATE_SUBMISSION",
  "message": "This file has already been submitted to this problem"
}
```

#### 429 Too Many Requests
Rate limit exceeded.

```json
{
  "error": "RATE_LIMIT_EXCEEDED",
  "message": "Please wait before submitting again. Rate limit: 1 second between submissions to the same problem"
}
```

---

### POST /api/groups/{groupId}/contests/{contestId}/problems/{problemSlug}/submissions

Submit a solution to a problem within a contest.

> **Important**: 
> - User must be registered to the contest
> - Problem must be part of the contest
> - Submissions only allowed during ACTIVE contests or postcompetition (if enabled)
> - Submissions during postcompetition (submittedAt > endTime) do NOT affect standings
> - All other validations from outside-contest submissions apply

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |
| Content-Type | string | Yes | multipart/form-data |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | string (UUID) | Yes | The unique identifier of the group |
| contestId | string (UUID) | Yes | The unique identifier of the contest |
| problemSlug | string | Yes | The slug of the problem |

**Request Body** (multipart/form-data):

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| file | file | Yes | Source code file (.cpp, .java, .py) |
| language | string | Yes | Language identifier (cpp20, java17, python310) |
| compiler | string | Yes | Compiler identifier (g++, javac, py) |

**Responses**:

#### 201 Created
Submission created successfully and queued for judging.

```json
{
  "id": "s1d2e3f4-g5h6-7890-1234-567890123456",
  "status": "PENDING",
  "submittedAt": "2026-01-10T14:30:00Z",
  "problem": {
    "slug": "sum-of-two-numbers",
    "title": "Sum of Two Numbers"
  },
  "contest": {
    "id": "c1d2e3f4-g5h6-7890-1234-567890123456",
    "name": "Weekly Contest #1"
  },
  "language": "cpp20",
  "compiler": "g++",
  "fileSize": 2048,
  "fileHash": "a1b2c3d4e5f6..."
}
```

#### 400 Bad Request
Validation error.

```json
{
  "error": "CONTEST_NOT_STARTED",
  "message": "Submissions are only allowed during ACTIVE contests or postcompetition"
}
```

```json
{
  "error": "CONTEST_FINISHED",
  "message": "The contest has ended and postcompetition is not enabled"
}
```

```json
{
  "error": "PROBLEM_NOT_IN_CONTEST",
  "message": "This problem is not part of this contest"
}
```

#### 403 Forbidden
User doesn't have permission.

```json
{
  "error": "NOT_REGISTERED",
  "message": "You must be registered to the contest to submit solutions"
}
```

#### 404 Not Found
Contest, group, or problem not found.

```json
{
  "error": "NOT_FOUND",
  "message": "Contest not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Submission Creation (Outside Contest)**
* **FR-SS-001**: The system MUST allow any authenticated user to submit solutions to PUBLISHED PUBLIC problems.
* **FR-SS-002**: The system MUST allow only modifiers (author + assigned modifiers) to submit solutions to PUBLISHED PRIVATE problems.
* **FR-SS-003**: The system MUST reject submissions to DRAFT problems.
* **FR-SS-004**: The system MUST validate that the problem exists before accepting submission.
* **FR-SS-005**: The system MUST validate file size against maximum allowed size (1MB, configurable via Virtual Object).
* **FR-SS-006**: The system MUST validate that compiler matches file extension.
* **FR-SS-007**: The system MUST calculate and store file hash (SHA256) for duplicate detection.
* **FR-SS-008**: The system MUST reject submissions with duplicate file hash from the same user to the same problem.
* **FR-SS-009**: The system MUST enforce rate limiting: 1 second between submissions from same user to same problem.
* **FR-SS-010**: The system MUST store source code file in storage with path: `{problemId}/{userId}/general/{submissionId}.{ext}`.
* **FR-SS-011**: The system MUST create submission with status PENDING.
* **FR-SS-011.1**: The system MUST capture `submittedAt` timestamp IMMEDIATELY when the request is received, before any validation or processing.
* **FR-SS-012**: The system MUST queue submission for asynchronous judging.
* **FR-SS-012.1**: The system MUST only accept new submissions to PUBLISHED problems.
* **FR-SS-012.2**: When a problem changes from PUBLISHED to DRAFT, the system MUST pause all PENDING submissions.
* **FR-SS-012.3**: When a problem changes from PUBLISHED to DRAFT, the system MUST allow RUNNING submissions to complete.
* **FR-SS-012.4**: When a problem's test cases, checker, or validator are updated, the system MUST wait for all RUNNING submissions to complete before applying changes.
* **FR-SS-012.5**: When a problem's test cases, checker, or validator are updated, the system MUST pause all PENDING submissions.
* **FR-SS-012.6**: When a problem returns to PUBLISHED status, paused PENDING submissions can be rejudged using the Rejudge Submissions feature.
* **FR-SS-012.7**: When a problem changes from PUBLIC to PRIVATE, existing submissions MUST continue executing normally.
* **FR-SS-012.8**: Users MUST be able to access their own submissions even if the problem becomes PRIVATE.

**Submission Creation (In Contest)**
* **FR-SS-013**: The system MUST validate that the user is a registered participant (individually or as part of a team's selectedMembers) before accepting submission.
* **FR-SS-013.1**: The system MUST first check if user is registered individually (fast O(1) lookup by userId).
* **FR-SS-013.2**: If user is registered individually, submission MUST use `standingId = userId`.
* **FR-SS-013.3**: If user is NOT registered individually, the system MUST check if user is in `selectedMembers` of any team registered to the contest.
* **FR-SS-013.4**: If user is in a team's selectedMembers, submission MUST use `standingId = teamId`.
* **FR-SS-013.5**: If user is neither registered individually nor in a team, reject with `NOT_REGISTERED`.
* **FR-SS-014**: The system MUST validate that the problem is part of the contest.
* **FR-SS-015**: The system MUST reject submissions when contest status is SCHEDULED.
* **FR-SS-016**: The system MUST allow submissions when contest status is ACTIVE.
* **FR-SS-017**: The system MUST reject submissions when contest status is FINISHED and `enablePostContest = false`.
* **FR-SS-018**: The system MUST allow submissions when contest status is FINISHED and `enablePostContest = true` (postcompetition).
* **FR-SS-019**: The system MUST store source code file in storage with path: `{problemId}/{submittedBy}/{contestId}/{submissionId}.{ext}` for contest submissions.
* **FR-SS-020**: The system MUST associate submission with contest (`contest_id` field).
* **FR-SS-020.1**: The system MUST capture `submittedAt` timestamp IMMEDIATELY when the request is received, before any validation or processing.
* **FR-SS-020.2**: The system MUST store `submittedBy` (userId) as the FK to User - this is the primary link.
* **FR-SS-020.3**: The system MUST store `standingId` (userId or teamId) to identify which standing document to update.
* **FR-SS-021**: The system MUST determine if submission affects standings by comparing `submittedAt` (captured at request receipt) with `contest.endTime`.

**File Handling**
* **FR-SS-022**: The system MUST accept files with extensions: .cpp, .java, .py.
* **FR-SS-023**: The system MUST validate file encoding (UTF-8) during compilation (not before submission).
* **FR-SS-024**: The system MUST store complete source code file in storage.
* **FR-SS-025**: The system MUST store file path/key in database.
* **FR-SS-026**: The system MUST store file hash (SHA256) in database.

**Language and Compiler**
* **FR-SS-027**: The system MUST support languages: cpp20, java17, python310.
* **FR-SS-028**: The system MUST support compilers: g++ (C++20), javac (Java 17), py (PyPy 3.10).
* **FR-SS-029**: The system MUST allow multiple compilers per file extension (e.g., .cpp can use g++).
* **FR-SS-030**: The system MUST validate compiler compatibility with file extension.
* **FR-SS-031**: The system MUST store compiler name and version in submission record.

**Judging**
* **FR-SS-032**: The system MUST use problem's language-specific limits for judging (from `languageOverrides` array).
* **FR-SS-033**: The system MUST use problem's `timeLimit` and `memoryLimit` defaults, or the override for the specific language if defined.
* **FR-SS-034**: The system MUST judge submission asynchronously after creation.
* **FR-SS-034.1**: The system MUST change submission status from PENDING to RUNNING when judging starts.
* **FR-SS-034.2**: The system MUST change submission status from RUNNING to final status when judging completes.
* **FR-SS-035**: The system MUST update submission status from RUNNING to final status after judging completes.
* **FR-SS-036**: The system MUST store processing time (duration in milliseconds) after judging completes.
* **FR-SS-037**: The system MUST store result verdict: ACCEPTED, WRONG_ANSWER, RUNTIME_EXCEPTION, TIME_LIMIT_EXCEEDED, MEMORY_LIMIT_EXCEEDED, COMPILATION_ERROR, PRESENTATION_ERROR.
* **FR-SS-038**: The system MUST store `judgedAt` timestamp when judging completes.
* **FR-SS-038.1**: The system MUST NOT start judging PENDING submissions when problem status is DRAFT.
* **FR-SS-038.2**: The system MUST allow RUNNING submissions to complete even if problem status becomes DRAFT.

**Standing Updates**
* **FR-SS-039**: The system MUST update contest Standing using `standingId` (userId or teamId) when submission is judged in ACTIVE contest.
* **FR-SS-040**: The system MUST NOT update contest Standing when submission is judged in FINISHED contest.
* **FR-SS-041**: The system MUST NOT update contest Standing for submissions during postcompetition (submittedAt > endTime).

**Validation**
* **FR-SS-042**: The system MUST validate that language is supported.
* **FR-SS-043**: The system MUST validate that compiler is supported for the selected language.
* **FR-SS-044**: The system MUST validate that problem has test cases before accepting submission.

**Response**
* **FR-SS-045**: The system MUST return submission ID, status, submittedAt, problem info, and contest info (if applicable).
* **FR-SS-046**: The system MUST NOT return source code in response.
* **FR-SS-047**: The system MUST return appropriate error codes for all validation failures.

### Key Entities

* **Submission**: Represents a user's solution attempt for a problem.
  * `id` (string, UUID, PK)
  * `problem_id` (string, UUID, FK to Problem)
  * `problemTitle` (string, preserved title of the problem)
  * `contest_id` (string, UUID, FK to Contest, nullable)
  * `submittedBy` (string, UUID, FK to User) - **primary link to user who submitted**
  * `standingId` (string, UUID, nullable) - userId OR teamId, determines which standing document to update
  * `status` (enum: PENDING | RUNNING | ACCEPTED | WRONG_ANSWER | RUNTIME_EXCEPTION | TIME_LIMIT_EXCEEDED | MEMORY_LIMIT_EXCEEDED | COMPILATION_ERROR | PRESENTATION_ERROR)
  * `language` (string: cpp20, java17, python310)
  * `compiler` (string: g++, javac, py - includes version info)
  * `filePath` (string, storage path/key)
  * `fileHash` (string, SHA256 hash)
  * `fileSize` (integer, bytes)
  * `submittedAt` (timestamp, captured immediately when request is received, before any processing)
  * `judgedAt` (timestamp, nullable)
  * `processingTime` (integer, milliseconds, nullable)
  * `result` (enum: same as status, nullable until judged)
  * `sourceCode` (text, stored in storage, not in DB)
  * **Deletion**: NOT deleted, but `contest_id` set to `null` when contest is deleted

> **Submission Status**:
> * `PENDING`: Submission is created and queued for judging (not yet started)
> * `RUNNING`: Submission is currently being judged/executed
> * `ACCEPTED`: Solution passed all test cases
> * `WRONG_ANSWER`: Solution produced incorrect output
> * `RUNTIME_EXCEPTION`: Solution crashed during execution
> * `TIME_LIMIT_EXCEEDED`: Solution exceeded time limit
> * `MEMORY_LIMIT_EXCEEDED`: Solution exceeded memory limit
> * `COMPILATION_ERROR`: Solution failed to compile
> * `PRESENTATION_ERROR`: Solution output format is incorrect

> **Status Transitions**:
> * `PENDING` → `RUNNING`: When judging starts
> * `RUNNING` → Final status: When judging completes
> * `PENDING` → (paused): When problem becomes DRAFT or test cases/checker/validator are updated
> * Paused `PENDING` → `RUNNING`: When problem returns to PUBLISHED and rejudge is triggered

> **Problem Status Impact on Submissions**:
> * Problem PUBLISHED → DRAFT:
>   * RUNNING submissions: Continue execution until completion
>   * PENDING submissions: Paused (not started)
>   * New submissions: Rejected
> * Problem DRAFT → PUBLISHED:
>   * Paused PENDING submissions: Can be rejudged via Rejudge Submissions feature
>   * New submissions: Accepted
> * Problem PUBLIC → PRIVATE:
>   * All submissions: Continue executing normally
>   * Users: Cannot access problem but can access their own submissions

> **File Storage Paths**:
> * Contest submission: `{problemId}/{userId}/{contestId}/{submissionId}.{ext}`
> * General submission: `{problemId}/{userId}/general/{submissionId}.{ext}`

> **Rate Limiting**:
> * 1 second between submissions from same user to same problem
> * Configurable via Virtual Object

> **Duplicate Detection**:
> * Same file hash (SHA256) from same user to same problem is rejected
> * Hash is calculated after file upload

### Virtual Object: System Configuration

**Purpose**: Centralized configuration for system-wide limits and constraints. This Virtual Object represents configuration that can be modified without code changes.

**Location**: Documented in spec and in main application README.md

**Configuration Values**:

```json
{
  "submissionLimits": {
    "maxFileSize": 1048576,
    "maxFileSizeUnit": "bytes",
    "rateLimitSeconds": 1
  },
  "languageOverrides": [
    { "language": "cpp20", "maxTimeLimit": 300000, "maxMemoryLimit": 2048 },
    { "language": "java17", "maxTimeLimit": 400000, "maxMemoryLimit": 2048 },
    { "language": "python310", "maxTimeLimit": 600000, "maxMemoryLimit": 2048 }
  ],
  "supportedLanguages": [
    { "language": "cpp20", "name": "C++20", "compilers": ["g++"], "extensions": [".cpp", ".cc", ".cxx"] },
    { "language": "java17", "name": "Java 17", "compilers": ["javac"], "extensions": [".java"] },
    { "language": "python310", "name": "Python 3.10", "compilers": ["py"], "extensions": [".py"] }
  ]
}
```

**Usage**:
* `submissionLimits.maxFileSize`: Maximum file size allowed for submissions (1MB)
* `submissionLimits.rateLimitSeconds`: Minimum seconds between submissions from same user to same problem
* `languageOverrides[].maxTimeLimit`: Maximum time limit (milliseconds) that can be set for a problem in this language
* `languageOverrides[].maxMemoryLimit`: Maximum memory limit (MiB) that can be set for a problem in this language
* `supportedLanguages`: Defines available languages, compilers, and file extensions

**Note**: These are maximum absolute limits. Problems can have lower limits, but cannot exceed these values.

### Permission Matrix

| Action | Public Problem | Private Problem (as Modifier) | Private Problem (as Non-Modifier) | Contest (Registered) | Contest (Not Registered) |
|--------|----------------|-------------------------------|----------------------------------|---------------------|--------------------------|
| Submit Solution | ✅ | ✅ | ❌ | ✅ | ❌ |
| View Own Submissions | ✅ | ✅ | ❌ | ✅ | ❌ |

### Submission Flow (Outside Contest)

```
POST /api/problems/{problemSlug}/submissions
    ↓
Capture submittedAt timestamp IMMEDIATELY (at request receipt)
    ↓
Validate user is authenticated
    ↓
Validate problem exists and is PUBLISHED
    ↓
Validate user can access problem (PUBLIC or modifier if PRIVATE)
    ↓
Validate file size ≤ maxFileSize (from Virtual Object)
    ↓
Validate compiler matches file extension
    ↓
Calculate file hash (SHA256)
    ↓
Check for duplicate hash (same user, same problem)
    ↓
Check rate limit (1 second since last submission to same problem)
    ↓
Store file in storage: {problemId}/{userId}/general/{submissionId}.{ext}
    ↓
Create Submission record:
  {
    problem_id: problemId,
    contest_id: null,
    contestant_id: userId,
    status: PENDING,
    language: language,
    compiler: compiler,
    filePath: storagePath,
    fileHash: hash,
    fileSize: fileSize,
    submittedAt: submittedAt (captured at request start)
  }
    ↓
If problem status is PUBLISHED:
    Queue submission for judging
Else:
    Submission remains PENDING (paused)
    ↓
Return 201 Created with submission data
```

### Submission Flow (In Contest)

```
POST /api/groups/{groupId}/contests/{contestId}/problems/{problemSlug}/submissions
    ↓
Capture submittedAt timestamp IMMEDIATELY (at request receipt)
    ↓
Validate user is authenticated
    ↓
Validate contest exists
    ↓
Validate user is registered to contest
    ↓
Validate contest status (must be ACTIVE or FINISHED with enablePostContest=true)
    ↓
Validate problem exists, is PUBLISHED, and is part of contest
    ↓
Validate file size, compiler, duplicate, rate limit (same as outside contest)
    ↓
Store file in storage: {problemId}/{userId}/{contestId}/{submissionId}.{ext}
    ↓
Create Submission record:
  {
    problem_id: problemId,
    contest_id: contestId,
    contestant_id: userId,
    status: PENDING,
    language: language,
    compiler: compiler,
    filePath: storagePath,
    fileHash: hash,
    fileSize: fileSize,
    submittedAt: submittedAt (captured at request start)
  }
    ↓
If problem status is PUBLISHED:
    Queue submission for judging
Else:
    Submission remains PENDING (paused)
    ↓
Return 201 Created with submission data (including contest info)
```

### Judging Flow

```
Submission queued for judging (status: PENDING)
    ↓
Check if problem status is PUBLISHED
    ↓
If problem status is DRAFT:
    Keep submission in PENDING (paused)
    Exit (wait for problem to return to PUBLISHED)
    ↓
If problem status is PUBLISHED:
    Change status to RUNNING
    ↓
    Retrieve problem's languageOverrides for submission language (if any)
    ↓
    Use problem's timeLimit and memoryLimit defaults, or override for this language
    ↓
    Compile solution using specified compiler
    ↓
    If compilation fails:
        ↓
        Update submission:
          status: COMPILATION_ERROR
          result: COMPILATION_ERROR
          judgedAt: now()
        ↓
    Else:
        ↓
        Run solution against all test cases
        ↓
        Check time limit (problem.timeLimit for language)
        Check memory limit (problem.memoryLimit for language)
        ↓
        If time exceeded:
          status: TIME_LIMIT_EXCEEDED
          result: TIME_LIMIT_EXCEEDED
        Else if memory exceeded:
          status: MEMORY_LIMIT_EXCEEDED
          result: MEMORY_LIMIT_EXCEEDED
        Else if runtime error:
          status: RUNTIME_EXCEPTION
          result: RUNTIME_EXCEPTION
        Else if wrong answer:
          status: WRONG_ANSWER
          result: WRONG_ANSWER
        Else if presentation error:
          status: PRESENTATION_ERROR
          result: PRESENTATION_ERROR
        Else:
          status: ACCEPTED
          result: ACCEPTED
        ↓
        Update submission:
          status: result
          result: result
          processingTime: executionTime
          judgedAt: now()
        ↓
        If submission has contest_id:
          ↓
          If contest is ACTIVE and submittedAt ≤ endTime:
            Update contest Standing (add problem solved, update penalty, recalculate ranking)
          Else:
            Do NOT update Standing (contest finished or postcompetition)
```

### Problem Status Change Flow

```
Problem status changes from PUBLISHED to DRAFT
    ↓
For each submission with status RUNNING:
    Allow to continue execution until completion
    ↓
For each submission with status PENDING:
    Pause submission (do not start judging)
    Keep status as PENDING
    ↓
Reject all new submission attempts
```

```
Problem test cases/checker/validator are updated
    ↓
Wait for all RUNNING submissions to complete
    ↓
For each submission with status PENDING:
    Pause submission (do not start judging)
    Keep status as PENDING
    ↓
Apply changes to problem
    ↓
Problem status may change to DRAFT (if required by Update Problem spec)
```

```
Problem status changes from DRAFT to PUBLISHED
    ↓
Paused PENDING submissions remain paused
    ↓
Author/modifiers can use Rejudge Submissions feature to:
    - Rejudge paused PENDING submissions
    - Rejudge completed submissions if test cases changed
    ↓
New submissions are now accepted
```

```
Problem accessibility changes from PUBLIC to PRIVATE
    ↓
All submissions continue executing normally
    ↓
Users cannot access problem details
    ↓
Users can still access their own submissions
```

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-SS-001**: Users can submit solutions to PUBLISHED PUBLIC problems via `POST /api/problems/{problemSlug}/submissions` with HTTP 201.
* **SC-SS-002**: Only modifiers can submit to PRIVATE problems - HTTP 403 for non-modifiers.
* **SC-SS-003**: Submissions to DRAFT problems are rejected - HTTP 400.
* **SC-SS-004**: Registered users can submit solutions in ACTIVE contests via `POST /api/groups/{groupId}/contests/{contestId}/problems/{problemSlug}/submissions` with HTTP 201.
* **SC-SS-005**: Non-registered users cannot submit in contests - HTTP 403.
* **SC-SS-006**: Submissions during SCHEDULED contests are rejected - HTTP 400.
* **SC-SS-007**: Submissions during FINISHED contests without postcompetition are rejected - HTTP 400.
* **SC-SS-008**: Submissions during postcompetition are accepted but do NOT affect standings.
* **SC-SS-009**: Duplicate file submissions (same hash, same user, same problem) are rejected - HTTP 409.
* **SC-SS-010**: Rate limit is enforced (1 second between submissions) - HTTP 429.
* **SC-SS-011**: File size validation works (files > 1MB rejected) - HTTP 400.
* **SC-SS-012**: Compiler-extension mismatch is rejected - HTTP 400.
* **SC-SS-013**: Submissions are created with status PENDING.
* **SC-SS-013.0**: `submittedAt` timestamp is captured immediately when request is received, before any validation or processing.
* **SC-SS-013.1**: Submissions change from PENDING to RUNNING when judging starts.
* **SC-SS-013.2**: Submissions change from RUNNING to final status when judging completes.
* **SC-SS-013.3**: When problem becomes DRAFT, RUNNING submissions complete execution.
* **SC-SS-013.4**: When problem becomes DRAFT, PENDING submissions are paused.
* **SC-SS-013.5**: When problem returns to PUBLISHED, paused submissions can be rejudged.
* **SC-SS-013.6**: When problem changes from PUBLIC to PRIVATE, submissions continue executing normally.
* **SC-SS-013.7**: Users can access their own submissions even if problem becomes PRIVATE.
* **SC-SS-014**: File hash (SHA256) is calculated and stored.
* **SC-SS-015**: Source code is stored in correct path (contest vs general).
* **SC-SS-016**: Submissions are queued for asynchronous judging.
* **SC-SS-017**: Problem's language-specific limits are used for judging.
* **SC-SS-018**: Standing is updated when submission is judged in ACTIVE contest.
* **SC-SS-019**: Standing is NOT updated for postcompetition submissions.
* **SC-SS-020**: Non-existent problems/contests return HTTP 404.

---

## Optional Notes

* **File Storage**: Consider using cloud storage (S3, GCS) for scalability. Path structure allows easy cleanup and organization.
* **Hash Algorithm**: SHA256 is used for duplicate detection. Consider MD5 for faster hashing if performance is critical, but SHA256 provides better collision resistance.
* **Rate Limiting**: Current implementation is per-user-per-problem. Consider implementing sliding window or token bucket for more sophisticated rate limiting.
* **Judging Queue**: Submissions are judged asynchronously. Consider implementing priority queue for contest submissions vs practice submissions.
* **Compiler Versions**: Compiler names should include version info (e.g., "g++-11.4.0"). This allows tracking which compiler version was used for each submission.
* **Language Limits**: The Virtual Object allows easy addition of new languages and adjustment of limits without code changes. This provides flexibility for platform administrators.
* **Future Enhancements**:
  * Support for additional languages (Rust, Go, JavaScript, etc.)
  * Support for multiple files per submission
  * Support for custom test cases in practice mode
  * Real-time submission status updates via WebSocket
  * Submission history and statistics
* **Related Specs**:
  * Create Problem: Problem creation with languageOverrides
  * Update Problem: Problem updates with languageOverrides
  * Rejudge Submissions: Rejudging submissions when test cases change
  * Register to Contest: Contest registration requirement
  * View Contest: Contest details and problem list

