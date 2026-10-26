# Feature Specification: View and Export Problem Files

**Created**: 2026-10-26

> A problem's files — test cases, solutions, checker, validator — can be uploaded and deleted, but until now never read back: the only way to check what was uploaded was to delete it and upload it again. This spec covers reading them: browsing the test cases, downloading the uploaded ZIP, and exporting the whole problem as an ICPC package (the inverse of **Problem Import**).

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Browse the test cases of a problem (Priority: P1)

As the author, a modifier or an Admin, I want to see which test cases a problem has and read any of them, so that I can check what I uploaded without re-uploading it.

**Independent Test**: Upload a test cases ZIP, then call `GET /problems/p/{slug}/testcases` and `GET /problems/p/{slug}/testcases/{kind}/{name}`.

**Acceptance Scenarios**:

1. **Scenario**: List the cases
   - **Given** a problem with an uploaded test cases ZIP
   - **When** the author lists the test cases
   - **Then** the response has every case of `data/sample` and `data/secret`, samples first, each with its name and the size of its `.in` and `.ans`
   - **And** names that are plain numbers are ordered numerically (`2` before `10`)

2. **Scenario**: Read one case
   - **Given** a listed case
   - **When** the author requests it by kind and name
   - **Then** the response has its input and answer as text
   - **And** a file longer than 64 KiB is cut at 64 KiB and flagged `inputTruncated` / `answerTruncated`

3. **Scenario**: No test cases uploaded yet
   - **Given** a problem without test cases
   - **When** the author lists them
   - **Then** the response is `hasTestCases: false` with an empty `cases` array (not an error)

4. **Scenario**: Incomplete case
   - **Given** the ZIP has a `.in` without its `.ans` (or the opposite)
   - **Then** the case is listed with the missing side as `null`, so the problem is visible instead of hidden

### User Story 2 – Download the uploaded test cases (Priority: P1)

As the author, a modifier or an Admin, I want to download the test cases ZIP exactly as it was uploaded.

**Acceptance Scenarios**:

1. **Scenario**: Download
   - **Given** a problem with an uploaded test cases ZIP
   - **When** `GET /problems/p/{slug}/files/testcases` is called
   - **Then** the ZIP is returned byte for byte, as an attachment named `{slug}-testcases.zip`

2. **Scenario**: Nothing uploaded
   - **Then** the system responds `404 PROBLEM_FILE_NOT_FOUND`

### User Story 3 – Read the code of a solution, checker or validator (Priority: P1)

As the author, a modifier or an Admin, I want to read the code of the problem's solutions, checker and validator, so that I can check what the judge will actually run.

**Acceptance Scenarios**:

1. **Scenario**: Read a solution
   - **Given** a problem with several solutions
   - **When** `GET /problems/p/{slug}/files/solution?fileName=sol.cpp` is called
   - **Then** the response has the file name as uploaded, its language and its source

2. **Scenario**: Read the checker or the validator
   - **When** `GET /problems/p/{slug}/files/checker` (or `validator`) is called
   - **Then** the response has the same shape

3. **Scenario**: A file the problem does not have
   - **Then** the system responds `404 PROBLEM_FILE_NOT_FOUND`

4. **Scenario**: Very long file
   - **Then** the source is cut at 256 KiB, `truncated` is `true` and `sizeBytes` still reports the real size

### User Story 4 – Export a problem as an ICPC package (Priority: P2)

As the author, a modifier or an Admin, I want to download a problem as an ICPC package so that I can back it up, move it to another instance or edit it offline, and import it again.

**Acceptance Scenarios**:

1. **Scenario**: Export
   - **Given** a problem (draft or published)
   - **When** `GET /problems/p/{slug}/export` is called
   - **Then** a ZIP named `{slug}.zip` is returned with the layout below

2. **Scenario**: Round trip
   - **Given** a package exported from a problem
   - **When** it is imported under another slug
   - **Then** the new problem has the same title, limits, statement, test cases, solutions, checker and validator
   - **And** exporting that problem gives the same files again

3. **Scenario**: Partial problem
   - **Given** a draft with only a title
   - **Then** the package contains only `problem.yaml`; unset limits are omitted rather than exported as `0`

### Package layout

```
{slug}.zip
├── problem.yaml                      name, time_limit (seconds), memory_limit (MiB)
├── problem_statement/problem.en.tex  the statement as stored (omitted if empty)
├── data/
│   ├── sample/{name}.in|.ans
│   └── secret/{name}.in|.ans
├── solutions/{filename}              every uploaded solution, original file name
├── checker.{cpp|py}                  if the problem has a custom checker
└── validator.{cpp|py}                if the problem has an input validator
```

This is the layout **Problem Import** reads. A root folder inside the uploaded test cases ZIP is dropped on export. Not part of the package, because the import does not read them either: tags, accessibility, modifiers and per-language limit overrides.

## API Contract

All these endpoints require authentication and are restricted to the problem's author, its modifiers and Admin.

| Endpoint | Success | Notes |
|----------|---------|-------|
| `GET /problems/p/{slug}/testcases` | `200` `{ hasTestCases, sampleCount, secretCount, cases: [{ kind, name, inputBytes, answerBytes }] }` | `kind` is `sample` or `secret`; a size is `null` if that file is missing |
| `GET /problems/p/{slug}/testcases/{kind}/{name}` | `200` `{ kind, name, input, answer, inputTruncated, answerTruncated }` | `400 VALIDATION_ERROR` if `kind` is not `sample`/`secret`; `404 PROBLEM_FILE_NOT_FOUND` for an unknown case |
| `GET /problems/p/{slug}/files/testcases` | `200` `application/zip` | `404 PROBLEM_FILE_NOT_FOUND` if no test cases were uploaded |
| `GET /problems/p/{slug}/files/{fileType}?fileName=` | `200` `{ fileType, filename, language, content, truncated, sizeBytes }` | `fileType` is `solution`, `checker` or `validator` (case-insensitive); `fileName` is required for `solution`. `400 PROBLEM_MISSING_FILENAME` / `400 PROBLEM_INVALID_FILE_TYPE` (also for `testcases`, which is read with the two endpoints above); `404 PROBLEM_FILE_NOT_FOUND` if the problem has no such file |
| `GET /problems/p/{slug}/export` | `200` `application/zip` | |

Errors common to all: `401` without a valid token, `403 INSUFFICIENT_PERMISSIONS` for anyone else, `404` if the problem does not exist. If the problem points at a file that is no longer in storage, the response is `404 PROBLEM_FILE_NOT_FOUND` with a message naming it (it is not reported as a missing problem).

ZIP responses carry `Content-Disposition: attachment; filename="..."` and `Cache-Control: no-store`.

## Requirements *(mandatory)*

### Functional Requirements

**Access**
* **FR-VF-001**: The system MUST restrict reading a problem's test cases, downloading its test cases ZIP and exporting it to its author, modifiers and Admin, whatever the problem's status or accessibility. A published public problem does not make its test cases readable by anyone else.
* **FR-VF-002**: A forbidden response MUST NOT include any test data.

**Test cases**
* **FR-VF-003**: The system MUST list the cases found in `data/sample` and `data/secret` of the stored ZIP, wherever the ZIP's root folder is, ignoring `__MACOSX` entries and files that are not `.in`/`.ans`.
* **FR-VF-004**: The list MUST order samples before secret cases, and order names numerically when they are numbers.
* **FR-VF-005**: A preview MUST return at most 64 KiB of each file, MUST say when it was cut, and MUST NOT return invalid UTF-8.
* **FR-VF-006**: The download MUST return the uploaded ZIP unchanged.

**Judging files**
* **FR-VF-011**: The system MUST return the source of a solution, checker or validator, with the file name the author uploaded and its language.
* **FR-VF-012**: The system MUST return at most 256 KiB of a file, MUST say when it was cut, and MUST still report the real size.
* **FR-VF-013**: A forbidden response MUST NOT include any file content.

**Export**
* **FR-VF-007**: The export MUST write only what **Problem Import** reads, in the layout above, so that importing the export recreates the problem.
* **FR-VF-008**: The export MUST write `time_limit` in seconds and `memory_limit` in MiB, and MUST omit a limit that is not set.
* **FR-VF-009**: The export MUST fail (not produce a partial package) if a file the problem refers to is missing from storage.

**Responses**
* **FR-VF-010**: ZIP responses MUST NOT be cacheable, since they hold secret test data and solutions.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-VF-001**: The author can see every uploaded case and read any of them without re-uploading.
- **SC-VF-002**: An export imported under another slug exports again to the same files with identical content.
- **SC-VF-003**: A user who is not author, modifier or Admin gets `403` on all these endpoints, including for published public problems.
- **SC-VF-005**: The author can read the code of every solution, the checker and the validator of a problem, on drafts and on published problems.
- **SC-VF-004**: Downloading the test cases returns exactly the bytes that were uploaded.
