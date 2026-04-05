# Feature Specification: Problem Statistics

**Created**: 2026-02-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – View problem statistics (Priority: P2)

As an authenticated user with access to a problem, I want to view comprehensive statistics about that problem so that I can evaluate its difficulty, understand submission patterns, and make informed decisions about attempting or using the problem.

**Why this priority**: This feature significantly improves user experience by helping users evaluate problem difficulty before attempting, assists authors in improving problems based on data, and provides valuable information for coaches when selecting problems for training. However, it's not critical for basic system functionality as users can still view and solve problems without statistics.

**Independent Test**: This user story can be tested independently by consuming the `GET /problems/p/{slug}/statistics` endpoint with valid authentication and different problem visibility scenarios, validating that statistics are correctly calculated and access control is properly enforced.

**Acceptance Scenarios**:

1. **Scenario**: View statistics for published problem with submissions
   - **Given** a problem with status PUBLISHED exists
   - **And** the problem has multiple submissions from different users and languages
   - **And** a user is authenticated (any role)
   - **When** the user requests GET /problems/p/{slug}/statistics
   - **Then** the system returns HTTP 200 with complete statistics including:
     - Total submissions count
     - Unique users who attempted (at least one submission)
     - Unique users who solved (at least one ACCEPTED submission)
     - Acceptance rate by language (ordered by highest rate first)
     - Verdict distribution with counts for all verdicts that have submissions
   - **And** all calculations include submissions from both active and deactivated users

2. **Scenario**: View statistics for problem with no submissions
   - **Given** a problem with status PUBLISHED exists
   - **And** the problem has zero submissions
   - **And** a user is authenticated
   - **When** the user requests GET /problems/p/{slug}/statistics
   - **Then** the system returns HTTP 200 with message indicating no statistics available
   - **And** the response includes: `{"message": "No submissions yet for this problem", "totalSubmissions": 0}`

3. **Scenario**: Attempt to view statistics for draft problem
   - **Given** a problem with status DRAFT exists
   - **And** a user is authenticated (any role, including modifier or Admin)
   - **When** the user requests GET /problems/p/{slug}/statistics
   - **Then** the system rejects with HTTP 403 Forbidden
   - **And** returns error code PROBLEM_NOT_PUBLISHED
   - **And** message indicates statistics are only available for published problems

4. **Scenario**: Unauthenticated request
   - **Given** a problem exists (any status)
   - **When** a request is made without valid authentication credentials
   - **Then** the system rejects with HTTP 401 Unauthorized

5. **Scenario**: Problem not found
   - **Given** no problem exists with the provided slug
   - **When** a user requests GET /problems/p/{slug}/statistics
   - **Then** the system returns HTTP 404 Not Found

6. **Scenario**: Statistics with multiple languages
   - **Given** a problem has submissions in cpp20, python310, and java17
   - **And** cpp20 has 50 users attempted, 30 solved (60% rate)
   - **And** python310 has 40 users attempted, 20 solved (50% rate)
   - **And** java17 has 30 users attempted, 21 solved (70% rate)
   - **When** statistics are requested
   - **Then** languages are ordered: java17 (70%), cpp20 (60%), python310 (50%)

7. **Scenario**: Statistics include all verdict types
   - **Given** a problem has submissions with verdicts: ACCEPTED, WRONG_ANSWER, TIME_LIMIT_EXCEEDED, COMPILATION_ERROR
   - **When** statistics are requested
   - **Then** verdict distribution includes all four verdicts with their respective counts
   - **And** verdicts with zero submissions are NOT included

8. **Scenario**: Statistics include deactivated users' submissions
    - **Given** a problem has 100 total submissions
    - **And** 20 submissions are from users who are now deactivated
    - **When** statistics are requested
    - **Then** totalSubmissions shows 100
    - **And** all 100 submissions are included in verdict distribution and acceptance rate calculations

---

### Edge Cases

- Problem with only one submission
- Problem with submissions from only one user (multiple submissions)
- Problem with all submissions having same verdict
- Problem with submissions in only one language
- All users who attempted also solved (100% acceptance rate)
- No users solved the problem (0% acceptance rate)
- User with multiple ACCEPTED submissions (should count as 1 solved user)
- User with submissions in multiple languages
- Very large number of submissions (performance consideration)
- Concurrent requests to statistics endpoint
- Problem visibility changes while viewing statistics
- Unicode characters in problem slug
- Statistics calculation with floating point precision

## API Contract

### GET /problems/p/{slug}/statistics

Retrieve comprehensive statistics for a specific problem including submission counts, user metrics, acceptance rates by language, and verdict distribution.

> **Important**: Statistics are only available for PUBLISHED problems. DRAFT problems return 403 Forbidden regardless of user role. Statistics include all submissions from both active and deactivated users. 
> 
> **Note on DRAFT restriction for Admin/Modifiers**: Even if the authenticated user is an Admin or assigned modifier who can normally view and edit the DRAFT problem, they are **explicitly blocked** from viewing statistics because DRAFT problems are not organically open to submissions.

**Headers**:

| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for authentication |

**Path Parameters**:

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| slug | string | Yes | Unique problem identifier (3-70 chars, lowercase alphanumeric with hyphens) |

**Responses**:

#### 200 OK
Statistics retrieved successfully.

**When problem has submissions**:
```json
{
  "totalSubmissions": 1250,
  "uniqueUsers": {
    "attempted": 234,
    "solved": 89
  },
  "acceptanceRateByLanguage": [
    {
      "language": "java17",
      "usersAccepted": 35,
      "usersAttempted": 50
    },
    {
      "language": "cpp20",
      "usersAccepted": 30,
      "usersAttempted": 110
    },
    {
      "language": "python310",
      "usersAccepted": 24,
      "usersAttempted": 74
    }
  ],
  "verdictDistribution": [
    {
      "verdict": "ACCEPTED",
      "count": 450
    },
    {
      "verdict": "WRONG_ANSWER",
      "count": 520
    },
    {
      "verdict": "TIME_LIMIT_EXCEEDED",
      "count": 180
    },
    {
      "verdict": "COMPILATION_ERROR",
      "count": 65
    },
    {
      "verdict": "RUNTIME_EXCEPTION",
      "count": 35
    }
  ]
}
```

**When problem has no submissions**:
```json
{
  "message": "No submissions yet for this problem",
  "totalSubmissions": 0
}
```

**Field Descriptions**:

| Field | Type | Description |
|-------|------|-------------|
| totalSubmissions | integer | Total number of submissions to this problem (all time) |
| uniqueUsers | object | User metrics |
| uniqueUsers.attempted | integer | Number of unique users who made at least one submission |
| uniqueUsers.solved | integer | Number of unique users who have at least one ACCEPTED submission |
| acceptanceRateByLanguage | array | Acceptance rate breakdown by programming language |
| acceptanceRateByLanguage[].language | string | Programming language (cpp20, java17, python310, etc.) |
| acceptanceRateByLanguage[].usersAccepted | integer | Number of users who solved using this language |
| acceptanceRateByLanguage[].usersAttempted | integer | Number of users who attempted using this language |
| verdictDistribution | array | Distribution of submission verdicts |
| verdictDistribution[].verdict | string | Verdict type (ACCEPTED, WRONG_ANSWER, etc.) |
| verdictDistribution[].count | integer | Number of submissions with this verdict |
| message | string | Message when no submissions exist (only in no-submissions case) |

**Notes**:
- `acceptanceRateByLanguage` is ordered by acceptance rate descending (highest rate first)
- Acceptance rate can be calculated in frontend: `(usersAccepted / usersAttempted) * 100`
- Only languages with at least one submission are included in `acceptanceRateByLanguage`
- Only verdicts with at least one submission are included in `verdictDistribution`
- A user who submits in multiple languages is counted separately for each language
- A user with multiple ACCEPTED submissions counts as 1 solved user
- Statistics include submissions from both active and deactivated users

#### 401 Unauthorized
Authentication failed (invalid or missing token).

```json
{
  "error": "UNAUTHORIZED",
  "message": "Invalid or missing authentication token"
}
```

#### 403 Forbidden
Problem is not published (statistics only available for PUBLISHED problems).

```json
{
  "error": "PROBLEM_NOT_PUBLISHED",
  "message": "Statistics are only available for published problems"
}
```

#### 404 Not Found
Problem with the specified slug does not exist.

```json
{
  "error": "NOT_FOUND",
  "message": "Problem not found"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

**Access Control**
- **FR-001**: The system MUST allow authenticated users to view statistics for PUBLISHED problems via GET /problems/p/{slug}/statistics.
- **FR-002**: The system MUST reject requests for DRAFT problems with HTTP 403 Forbidden, regardless of user role.
- **FR-003**: The system MUST reject unauthenticated requests with HTTP 401 Unauthorized.
- **FR-004**: The system MUST return HTTP 404 Not Found for non-existent problem slugs.
- **FR-005**: Statistics MUST only be calculated and returned for problems with status PUBLISHED.

**Total Submissions**
- **FR-006**: The system MUST return the total count of all submissions to the problem.
- **FR-007**: The system MUST include submissions from both active and deactivated users in the total count.
- **FR-008**: The system MUST count all submissions regardless of verdict.

**Unique Users Metrics**
- **FR-009**: The system MUST calculate and return the number of unique users who attempted the problem (made at least one submission).
- **FR-010**: The system MUST calculate and return the number of unique users who solved the problem (have at least one ACCEPTED submission).
- **FR-011**: The system MUST count each user only once in the attempted metric, regardless of number of submissions.
- **FR-012**: The system MUST count each user only once in the solved metric, regardless of number of ACCEPTED submissions.
- **FR-013**: The system MUST include both active and deactivated users in unique user counts.

**Acceptance Rate by Language**
- **FR-014**: The system MUST calculate acceptance rate for each programming language that has at least one submission.
- **FR-015**: For each language, the system MUST return the number of users who solved using that language (usersAccepted).
- **FR-016**: For each language, the system MUST return the number of users who attempted using that language (usersAttempted).
- **FR-017**: A user is considered to have "attempted" a language if they made at least one submission in that language.
- **FR-018**: A user is considered to have "solved" with a language if they have at least one ACCEPTED submission in that language.
- **FR-019**: The system MUST order languages by acceptance rate in descending order (highest rate first).
- **FR-020**: The system MUST NOT include languages with zero submissions.
- **FR-021**: A user who submits in multiple languages MUST be counted separately for each language.

**Verdict Distribution**
- **FR-022**: The system MUST return the count of submissions for each verdict type.
- **FR-023**: The system MUST include all verdict types that have at least one submission.
- **FR-024**: The system MUST NOT include verdict types with zero submissions.
- **FR-025**: The system MUST count all submissions regardless of user status (active or deactivated).
- **FR-026**: Verdict types include but are not limited to: ACCEPTED, WRONG_ANSWER, TIME_LIMIT_EXCEEDED, MEMORY_LIMIT_EXCEEDED, RUNTIME_EXCEPTION, COMPILATION_ERROR, SYSTEM_ERROR.

**No Submissions Case**
- **FR-027**: When a problem has zero submissions, the system MUST return HTTP 200 with a message indicating no statistics are available.
- **FR-028**: The no-submissions response MUST include: message field and totalSubmissions: 0.
- **FR-029**: The no-submissions response MUST NOT include uniqueUsers, acceptanceRateByLanguage, or verdictDistribution fields.

**Data Accuracy**
- **FR-030**: All statistics MUST be calculated from all submissions to the problem (all time, not time-limited).
- **FR-031**: Statistics MUST include submissions made during contests and practice submissions.
- **FR-032**: Statistics MUST be calculated in real-time or near real-time (no stale cached data).
- **FR-033**: The system MUST handle concurrent submissions without corrupting statistics.

**Response Format**
- **FR-034**: The system MUST return HTTP 200 with complete statistics for successful requests.
- **FR-035**: The system MUST return appropriate HTTP status codes (401, 403, 404) with clear error messages for failures.
- **FR-036**: The system MUST return consistent JSON format for all responses.
- **FR-037**: The system MUST use consistent field names and data types.

### Key Entities

📝 **Please Refer to `README.md`**

For the canonical documentation of the `Problem`, `Submission`, and `User` entities, please refer to the `README.md` at the root of the Problem management directory.

> **Note on Statistics Calculation**: Statistics must be calculated by aggregating data from the Submission table, joining with Problem table for access control, and potentially joining with User table for role-based access checks. Deactivated users' submissions are included in all calculations.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Authenticated users can view statistics for PUBLISHED problems via GET /problems/p/{slug}/statistics with HTTP 200.
- **SC-002**: Requests for DRAFT problems receive HTTP 403 Forbidden regardless of user role.
- **SC-003**: Unauthenticated requests receive HTTP 401 Unauthorized.
- **SC-004**: Non-existent problem slugs return HTTP 404 Not Found.
- **SC-005**: Statistics include accurate total submissions count.
- **SC-006**: Statistics include accurate unique users attempted and solved counts.
- **SC-007**: Acceptance rate by language is correctly calculated and ordered by highest rate first.
- **SC-008**: Only languages with submissions are included in acceptance rate breakdown.
- **SC-009**: Verdict distribution includes all verdicts with submissions and their accurate counts.
- **SC-010**: Only verdicts with submissions are included in distribution.
- **SC-011**: Statistics include submissions from both active and deactivated users.
- **SC-012**: Problems with no submissions return appropriate message with totalSubmissions: 0.
- **SC-013**: Users with multiple submissions in same language are counted once per language.
- **SC-014**: Users with submissions in multiple languages are counted separately for each language.
- **SC-015**: Acceptance rate can be calculated in frontend using provided absolute numbers.
- **SC-016**: Statistics reflect all submissions (contest and practice) without time limitation.
- **SC-017**: Statistics are only available for PUBLISHED problems.
- **SC-018**: Error responses follow consistent JSON format with error code and message.
- **SC-019**: Statistics are calculated accurately even with concurrent submissions.
