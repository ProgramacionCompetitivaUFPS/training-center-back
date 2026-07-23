# Feature Specification: Change Submission Visibility

**Created**: 2026-06-22

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Change submission visibility (Priority: P2)

As a user, I want to change the visibility of my own submission between public and private so that I can control who can view my source code.

**Why this priority**: Visibility control is a useful feature for sharing solutions after contests or practice, but it is not essential for the core functionality of the platform.

**Independent Test**: This user story can be tested independently by consuming the `PATCH /api/submissions/{submissionId}/visibility` endpoint with valid authentication, validating that only the submission author can change visibility and that the change is reflected in subsequent view requests.

**Acceptance Scenarios**:

1. **Scenario**: Author makes submission public
   * **Given** a user has a PRIVATE submission
   * **And** the authenticated user is the author of the submission
   * **When** they send PATCH `/api/submissions/{submissionId}/visibility` with `{ "visibility": "PUBLIC" }`
   * **Then** the system updates the submission visibility to PUBLIC
   * **And** returns 200 OK with updated submission data

2. **Scenario**: Author makes submission private
   * **Given** a user has a PUBLIC submission
   * **And** the authenticated user is the author of the submission
   * **When** they send PATCH `/api/submissions/{submissionId}/visibility` with `{ "visibility": "PRIVATE" }`
   * **Then** the system updates the submission visibility to PRIVATE
   * **And** returns 200 OK with updated submission data

3. **Scenario**: Non-author attempts to change visibility
   * **Given** a submission exists authored by User A
   * **And** the authenticated user is NOT the author (User B, even if Admin)
   * **When** they attempt to change the visibility
   * **Then** the system rejects with 403 Forbidden (ACCESS_DENIED)

4. **Scenario**: Submission not found
   * **Given** no submission exists with the provided ID
   * **When** the user attempts to change visibility
   * **Then** the system rejects with 404 Not Found

---

## Requirements *(mandatory)*

### Functional Requirements

* **FR-CSV-001**: The system MUST only allow the author of a submission to change its visibility.
* **FR-CSV-002**: The system MUST support two visibility values: PUBLIC and PRIVATE.
* **FR-CSV-003**: The system MUST reject visibility change requests from non-authors with 403 Forbidden.
* **FR-CSV-004**: The system MUST return 404 Not Found if the submission does not exist.
* **FR-CSV-005**: The default visibility for new submissions MUST be PRIVATE (enforced at creation, not here).

### Key Entities

* **Submission**
  * `visibility` (enum: PUBLIC | PRIVATE, default: PRIVATE)

---

## API Contract

### PATCH /api/submissions/{submissionId}/visibility

Update the visibility of a submission.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| submissionId | UUID | Yes | The unique identifier of the submission |

**Request Body**:

```json
{
  "visibility": "PUBLIC"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| visibility | enum | Yes | PUBLIC or PRIVATE |

**Responses**:

#### 200 OK

Visibility updated successfully.

```json
{
  "id": "submission-uuid",
  "visibility": "PUBLIC",
  "message": "Visibility updated successfully"
}
```

#### 403 Forbidden

Only the author can change visibility.

```json
{
  "error": "ACCESS_DENIED",
  "message": "Only the submission author can change visibility"
}
```

#### 404 Not Found

Submission not found.

```json
{
  "error": "SUBMISSION_NOT_FOUND",
  "message": "Submission not found"
}
```

---

## Success Criteria *(mandatory)*

### Measurable Outcomes

* **SC-CSV-001**: Authors can change their submission from PRIVATE to PUBLIC via PATCH with HTTP 200.
* **SC-CSV-002**: Authors can change their submission from PUBLIC to PRIVATE via PATCH with HTTP 200.
* **SC-CSV-003**: Non-authors attempting to change visibility receive HTTP 403 (ACCESS_DENIED).
* **SC-CSV-004**: Requests for non-existent submissions return HTTP 404.
* **SC-CSV-005**: After changing to PUBLIC, any authenticated user can view the submission.
* **SC-CSV-006**: After changing to PRIVATE, only authorized users (author, Admin, Lead, teammate) can view the submission.
