# Feature Specification: Download Submission

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Download submission source code (Priority: P3)

As a user with access to a submission, I want to download the source code file so that I can save it locally, analyze it offline, or use it for educational purposes.

**Why this priority**: This is a convenience feature that improves user experience but is not essential since the code is already viewable through the View submission endpoint. Users can copy-paste code if needed, making this a "nice to have" feature.

**Independent Test**: This user story can be tested independently by consuming the `GET /submissions/{id}/download` endpoint with valid authentication and proper permissions, validating that the file is downloaded with correct filename, extension, and content.

**Acceptance Scenarios**:

1. **Scenario**: Download own submission
   - **Given** a user has submitted a solution to a problem
   - **And** the user is authenticated
   - **When** they request GET /submissions/{id}/download
   - **Then** the system returns the source code file
   - **And** filename is `{nickname}_{submissionId}.{ext}` (e.g., `johndoe_12345.cpp`)
   - **And** Content-Disposition header is set to `attachment`
   - **And** Content-Type is appropriate for the file type
   - **And** file content matches the submission source code

2. **Scenario**: Download PUBLIC submission from another user
   - **Given** another user has a PUBLIC submission
   - **And** the requesting user is authenticated
   - **When** they request GET /submissions/{id}/download
   - **Then** the system returns the source code file
   - **And** filename uses the original author's nickname

3. **Scenario**: Download submission from deactivated user
   - **Given** a submission exists from a user who is now DEACTIVATED
   - **And** the requesting user has permission to view the submission
   - **When** they request GET /submissions/{id}/download
   - **Then** the system returns the source code file
   - **And** filename is `anonymous_{submissionId}.{ext}` (generic name)

4. **Scenario**: Admin downloads any submission
   - **Given** a submission exists (PUBLIC or PRIVATE)
   - **And** the requesting user is Admin
   - **When** they request GET /submissions/{id}/download
   - **Then** the system returns the source code file with appropriate filename

5. **Scenario**: Lead downloads submission in their group's contest
   - **Given** a submission exists in a contest within the Lead's group
   - **And** the requesting user is a Lead of that group
   - **When** they request GET /submissions/{id}/download
   - **Then** the system returns the source code file

6. **Scenario**: Team member downloads teammate's submission
   - **Given** a user is part of a team in a contest
   - **And** a teammate submitted a solution
   - **When** the user requests the teammate's submission download
   - **Then** the system returns the source code file

7. **Scenario**: Attempt to download PRIVATE submission without permission
   - **Given** another user has a PRIVATE submission
   - **And** the requesting user is NOT the author, Admin, Lead, or teammate
   - **When** they request GET /submissions/{id}/download
   - **Then** the system rejects with HTTP 403 Forbidden

8. **Scenario**: Unauthenticated download attempt
   - **Given** a submission exists
   - **When** a request is made without valid authentication credentials
   - **Then** the system rejects with HTTP 401 Unauthorized

9. **Scenario**: Submission not found
   - **Given** no submission exists with the provided ID
   - **When** a download request is submitted
   - **Then** the system returns HTTP 404 Not Found

10. **Scenario**: Download submission with different languages
    - **Given** submissions exist in CPP, JAVA, and PYTHON
    - **When** each is downloaded
    - **Then** CPP submission has .cpp extension
    - **And** JAVA submission has .java extension
    - **And** PYTHON submission has .py extension

11. **Scenario**: Download submission from deleted problem
    - **Given** a submission exists for a problem that was deleted
    - **And** the requesting user has permission to view the submission
    - **When** they request GET /submissions/{id}/download
    - **Then** the system returns the source code file
    - **And** filename uses the submitter's nickname and submission ID (no problem slug needed)

---

### Edge Cases

- Download submission with very long nickname (truncation handling)
- Download submission with special characters in nickname
- Concurrent download requests for same submission
- Download submission with large source code file (near size limit)
- Download submission with non-UTF-8 characters in code
- Download submission immediately after it was submitted
- Browser compatibility with Content-Disposition header
- File extension mapping for new/unsupported languages

## API Contract

### GET /submissions/{id}/download

Download the source code file of a specific submission.

> **Important**: Access permissions follow the same rules as View submission endpoint. Only users with permission to view a submission can download it. The file is returned with appropriate headers to trigger browser download.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | string | Yes | The unique identifier of the submission (UUID) |

**Responses**:

#### 200 OK
Source code file returned successfully.

**Response Headers**:
```
Content-Type: text/x-c++src (or appropriate for language)
Content-Disposition: attachment; filename="johndoe_12345.cpp"
Content-Length: 1024
```

**Response Body**: Raw source code file content (UTF-8 encoded text)

**Filename Format**:
- Active user: `{nickname}_{submissionId}.{ext}` (e.g., `johndoe_12345.cpp`)
- Deactivated user: `anonymous_{submissionId}.{ext}` (e.g., `anonymous_12345.cpp`)

**Language to Extension Mapping**:
| Language | Extension |
|----------|-----------|
| CPP, CPP20 | .cpp |
| JAVA, JAVA17 | .java |
| PYTHON, PYTHON310 | .py |

**Content-Type Mapping**:
| Language | Content-Type |
|----------|--------------|
| CPP, CPP20 | text/x-c++src |
| JAVA, JAVA17 | text/x-java |
| PYTHON, PYTHON310 | text/x-python |

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
User does not have permission to download this submission.

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "You do not have permission to download this submission"
}
```

#### 404 Not Found
Submission with the specified ID does not exist.

```json
{
  "error": "NOT_FOUND",
  "message": "Submission not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Access Control**
- **FR-001**: The system MUST apply the same permission rules as the View submission endpoint.
- **FR-002**: The system MUST allow users to download their own submissions.
- **FR-003**: The system MUST allow any authenticated user to download PUBLIC submissions.
- **FR-004**: The system MUST restrict access to PRIVATE submissions to: author, Admin, group Lead (for contest submissions), and teammates (for contest submissions).
- **FR-005**: The system MUST reject unauthenticated requests with HTTP 401 Unauthorized.
- **FR-006**: The system MUST reject unauthorized requests with HTTP 403 Forbidden.
- **FR-007**: The system MUST return HTTP 404 Not Found for non-existent submissions.

**File Generation**
- **FR-008**: The system MUST generate filename in format `{nickname}_{submissionId}.{ext}` for active users.
- **FR-009**: The system MUST generate filename in format `anonymous_{submissionId}.{ext}` for deactivated users.
- **FR-010**: The system MUST map programming language to appropriate file extension (.cpp, .java, .py).
- **FR-011**: The system MUST retrieve source code content from storage using the submission's filePath.
- **FR-012**: The system MUST encode file content as UTF-8.

**HTTP Headers**
- **FR-013**: The system MUST set Content-Disposition header to `attachment; filename="{generated_filename}"`.
- **FR-014**: The system MUST set Content-Type header based on the programming language.
- **FR-015**: The system MUST set Content-Length header with the file size in bytes.
- **FR-016**: The system MUST return HTTP 200 OK with file content in response body.

**Language Support**
- **FR-017**: The system MUST support CPP/CPP20 with .cpp extension and text/x-c++src content type.
- **FR-018**: The system MUST support JAVA/JAVA17 with .java extension and text/x-java content type.
- **FR-019**: The system MUST support PYTHON/PYTHON310 with .py extension and text/x-python content type.

**Edge Cases**
- **FR-020**: The system MUST handle submissions from deleted problems (use nickname and submission ID only).
- **FR-021**: The system MUST handle nicknames with special characters by sanitizing them for filename safety.
- **FR-022**: The system MUST handle very long nicknames by truncating if necessary (max 50 chars recommended).
- **FR-023**: The system MUST handle concurrent download requests without corruption.

### Key Entities

- **Submission**: Code submission for a problem.  
  Relevant attributes:
  - `id` (string, UUID, internal only)
  - `submittedBy` (string, UUID, FK to User)
  - `language` (enum: CPP, CPP20, JAVA, JAVA17, PYTHON, PYTHON310)
  - `filePath` (string, storage path/key)
  - `fileSize` (integer, bytes)
  - `visibility` (enum: PUBLIC | PRIVATE)
  - `contestId` (string, UUID, FK to Contest, nullable)

- **User**: Represents a user.  
  Relevant attributes:
  - `id` (string, UUID, internal only)
  - `nickname` (string, unique, lowercase)
  - `status` (enum: ACTIVE | DEACTIVATED)
  - `role` (enum: ADMIN | COACH | CONTESTANT)

> **Note on Permissions**: Download permissions exactly match View submission permissions. The same access control logic applies.

### Permission Matrix

| Viewer | Own Submission | PUBLIC Submission | PRIVATE Submission | Contest (Lead) | Contest (Team) |
|--------|----------------|-------------------|-------------------|----------------|----------------|
| Author | ✅ | ✅ | ✅ | ✅ | ✅ |
| Any User | ✅ | ✅ | ❌ | ❌ | ❌ |
| Admin | ✅ | ✅ | ✅ | ✅ | ✅ |
| Lead (group) | ✅ | ✅ | ❌ | ✅ | ❌ |
| Teammate | ✅ | ✅ | ❌ | ❌ | ✅ |

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can successfully download their own submissions with HTTP 200.
- **SC-002**: Users can download PUBLIC submissions from other users with HTTP 200.
- **SC-003**: Admin can download any submission (PUBLIC or PRIVATE) with HTTP 200.
- **SC-004**: Group Leads can download submissions from contests in their group with HTTP 200.
- **SC-005**: Team members can download teammates' submissions in the same contest with HTTP 200.
- **SC-006**: Unauthorized users attempting to download PRIVATE submissions receive HTTP 403 Forbidden.
- **SC-007**: Unauthenticated requests receive HTTP 401 Unauthorized.
- **SC-008**: Non-existent submission IDs return HTTP 404 Not Found.
- **SC-009**: Downloaded files have correct filename format: `{nickname}_{submissionId}.{ext}`.
- **SC-010**: Submissions from deactivated users use filename format: `anonymous_{submissionId}.{ext}`.
- **SC-011**: File extensions correctly map to languages (.cpp, .java, .py).
- **SC-012**: Content-Disposition header is set to `attachment` to trigger download.
- **SC-013**: Content-Type header matches the programming language.
- **SC-014**: Content-Length header accurately reflects file size.
- **SC-015**: File content matches the original submission source code.
- **SC-016**: Files are encoded as UTF-8.
- **SC-017**: Submissions from deleted problems can be downloaded successfully.
- **SC-018**: Special characters in nicknames are handled safely in filenames.
- **SC-019**: Concurrent downloads of the same submission work correctly.
- **SC-020**: All supported languages (CPP, JAVA, PYTHON) download with correct extensions.
