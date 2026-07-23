# Feature Specification: View Submission

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View my own submission (Priority: P1)

As a user, I want to view the details of my own submission so that I can review my code and understand the result.

**Why this priority**: Users need to see their submission details to debug and improve their solutions.

**Independent Test**: Authenticated user GET `/api/submissions/{id}`. Verify submission details including source code are returned.

**Acceptance Scenarios**:

1. **Scenario**: User views their own submission (outside contest)
   * **Given** user has submitted a solution to a problem
   * **When** they request the submission details
   * **Then** system returns full submission details
   * **And** includes source code
   * **And** includes verdict, execution time, and memory used

2. **Scenario**: User views their own submission (in contest)
   * **Given** user has submitted a solution in a contest
   * **When** they request the submission details
   * **Then** system returns full submission details with contest info
   * **And** includes source code

3. **Scenario**: User views a PUBLIC submission from another user
   * **Given** another user has a PUBLIC submission
   * **When** any authenticated user requests the submission details
   * **Then** system returns the submission details with source code

4. **Scenario**: User tries to view a PRIVATE submission from another user
   * **Given** another user has a PRIVATE submission
   * **And** the requesting user is NOT admin
   * **When** they request the submission details
   * **Then** system rejects with 403 Forbidden

5. **Scenario**: Admin views any submission
   * **Given** a submission exists (PUBLIC or PRIVATE)
   * **And** the requesting user is Admin
   * **When** they request the submission details
   * **Then** system returns the full submission details

6. **Scenario**: Lead views submission in their group's contest
   * **Given** a submission exists in a contest within the Lead's group
   * **When** the Lead requests the submission details
   * **Then** system returns the full submission details

7. **Scenario**: Team member views teammate's submission in contest
   * **Given** a user is part of a team in a contest
   * **And** a teammate submitted a solution
   * **When** the user requests the teammate's submission details
   * **Then** system returns the full submission details

8. **Scenario**: Submission not found
   * **Given** no submission exists with the provided ID
   * **When** user requests the submission
   * **Then** system rejects with 404 Not Found

---

## Requirements *(mandatory)*

### Functional Requirements

**Visibility Rules**

* **FR-VS-001**: Submissions MUST have a visibility attribute (`PUBLIC` or `PRIVATE`).
* **FR-VS-002**: `PUBLIC` submissions MUST be viewable by any authenticated user.
* **FR-VS-003**: `PRIVATE` submissions MUST only be viewable by:
  * The author of the submission
  * Admin users
  * Lead users (for submissions in contests within their group)
  * Team members (for submissions in the same contest)
* **FR-VS-004**: Default visibility for new submissions MUST be `PRIVATE`.

**Submission Details**

* **FR-VS-005**: System MUST return the submission verdict (ACCEPTED, WRONG_ANSWER, etc.).
* **FR-VS-006**: System MUST return execution time in milliseconds.
* **FR-VS-007**: System MUST return memory used in KB.
* **FR-VS-008**: System MUST return the source code content.
* **FR-VS-009**: System MUST return problem information (slug, title).
* **FR-VS-010**: System MUST return contest information if submission is associated with a contest.
* **FR-VS-011**: System MUST return submitter information (userId, nickname).

**Source Code Access**

* **FR-VS-012**: Source code MUST be retrieved from storage using the `filePath`.
* **FR-VS-013**: Source code MUST be returned as text content in the response.

### Key Entities

* **Submission**: Extended with visibility attribute
  * `visibility` (enum: PUBLIC | PRIVATE, default: PRIVATE)

### Permission Matrix

| Viewer | Own Submission | PUBLIC Submission | PRIVATE Submission | Contest (Lead) | Contest (Team) |
|--------|----------------|-------------------|-------------------|----------------|----------------|
| Author | ✅ | ✅ | ✅ | ✅ | ✅ |
| Any User | ✅ | ✅ | ❌ | ❌ | ❌ |
| Admin | ✅ | ✅ | ✅ | ✅ | ✅ |
| Lead (group) | ✅ | ✅ | ❌ | ✅ | ❌ |
| Teammate | ✅ | ✅ | ❌ | ❌ | ✅ |

---

## API Contract

### GET /api/submissions/{submissionId}

Retrieve details of a specific submission.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| submissionId | UUID | Yes | The unique identifier of the submission |

**Success Response (200 OK)**:

```json
{
  "id": "submission-uuid",
  "status": "ACCEPTED",
  "visibility": "PRIVATE",
  "submittedAt": "2026-02-07T14:30:00Z",
  "judgedAt": "2026-02-07T14:30:05Z",
  "problem": {
    "slug": "sum-of-two-numbers",
    "title": "Sum of Two Numbers"
  },
  "contest": {
    "id": "contest-uuid",
    "name": "Weekly Contest #1"
  },
  "submittedBy": {
    "id": "user-uuid",
    "nickname": "john_doe"
  },
  "language": "cpp20",
  "compiler": "g++",
  "executionTime": 45,
  "memoryKb": 12288,
  "sourceCode": "#include <iostream>\nint main() { ... }"
}
```

**Response Fields**:

| Field | Type | Description |
|-------|------|-------------|
| id | UUID | Submission identifier |
| status | enum | Verdict: PENDING, RUNNING, ACCEPTED, WRONG_ANSWER, etc. |
| visibility | enum | PUBLIC or PRIVATE |
| submittedAt | timestamp | When the submission was created |
| judgedAt | timestamp | When judging completed (null if not judged) |
| problem | object | Problem slug and title |
| contest | object | Contest info (null if not in contest) |
| submittedBy | object | User who submitted |
| language | string | Language identifier |
| compiler | string | Compiler used |
| executionTime | integer | Execution time in milliseconds (null if not judged) |
| memoryKb | integer | Memory used in KB (null if not judged) |
| sourceCode | string | Full source code content |

**Error Responses**:

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 403 Forbidden

```json
{
  "error": "ACCESS_DENIED",
  "message": "You do not have permission to view this submission"
}
```

#### 404 Not Found

```json
{
  "error": "SUBMISSION_NOT_FOUND",
  "message": "Submission not found"
}
```

---

## Notes / Implementation hints

* Source code is stored in cloud storage, not in the database
* Retrieve source code by reading from `filePath`
* For contest submissions, check team membership via `selectedMembers` lookup
* Default visibility is PRIVATE for security
* Consider caching source code retrieval for frequently accessed submissions
