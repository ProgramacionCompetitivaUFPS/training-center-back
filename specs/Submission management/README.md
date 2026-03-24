# Submission Management

## Overview

Submission Management handles the core functionality of submitting solutions to programming problems, both within contests and in practice mode. This module is responsible for accepting code files, validating them, queuing them for judging, and managing the submission lifecycle.

## Key Features

### Submission Types

* **Practice Submissions**: Submissions to problems outside of contests
  * Available to all authenticated users for PUBLIC problems
  * Only modifiers can submit to PRIVATE problems
  * Stored in path: `{problemId}/{userId}/general/{submissionId}.{ext}`

* **Contest Submissions**: Submissions to problems within contests
  * Requires registration to the contest (individually or as team member)
  * Only allowed during ACTIVE contests or postcompetition (if enabled)
  * Stored in path: `{problemId}/{submittedBy}/{contestId}/{submissionId}.{ext}`
  * May or may not affect standings depending on submission time
  
  **Team Support**: For team-based contests, the system automatically resolves:
  1. Check if user is registered individually (fast O(1) lookup)
  2. Check if user is in a team's `selectedMembers` (slower lookup)
  3. Submission linked to user via `submittedBy`, standing updates use `standingId` (userId or teamId)

### Supported Languages

* **C++20**: Using g++ compiler
* **Java 17**: Using javac compiler
* **Python 3.10**: Using PyPy compiler

### Submission States

* `PENDING`: Submission is created and queued for judging (not yet started)
* `RUNNING`: Submission is currently being judged/executed
* `ACCEPTED`: Solution passed all test cases
* `RUNTIME_EXCEPTION`: Solution crashed during execution
* `TIME_LIMIT_EXCEEDED`: Solution exceeded time limit
* `MEMORY_LIMIT_EXCEEDED`: Solution exceeded memory limit
* `COMPILATION_ERROR`: Solution failed to compile
* `WRONG_ANSWER`: Solution produced incorrect output
* `PRESENTATION_ERROR`: Solution output format is incorrect

### File Handling

* **File Size Limit**: 1MB (configurable via Virtual Object)
* **Supported Extensions**: .cpp, .java, .py
* **Storage**: Files stored in cloud storage (S3, GCS, etc.)
* **Hash**: SHA256 hash calculated for duplicate detection
* **Encoding**: UTF-8 (validated during compilation)

### Rate Limiting

* **Limit**: 1 second between submissions from same user to same problem
* **Configurable**: Via Virtual Object
* **Error**: HTTP 429 Too Many Requests

### Duplicate Detection

* **Method**: SHA256 hash comparison
* **Scope**: Same user, same problem
* **Action**: Reject with HTTP 409 Conflict

### Judging

* **Process**: Asynchronous judging after submission creation
* **Limits**: Uses problem's language-specific limits from `languageOverrides` array
* **Standing Updates**: Only for ACTIVE contests when `submittedAt ≤ endTime`
* **Postcompetition**: Submissions after `endTime` do NOT affect standings

## Implemented Specs

1. **[Submit solution](Submit%20solution/spec.md)** - Submit solutions to problems (practice and contest mode)
2. **[View submission](View%20submission/spec.md)** - View submission details and source code
3. **[View submission list](View%20submission%20list/spec.md)** - List user's submissions with filtering
4. **[Download submission](Download%20submission/spec.md)** - Download source code file

## Future Specs (Planned)

* **View contest submissions** - List submissions in a contest (see [Contest management](../Contest%20management/README.md))

## Submission Visibility

Submissions have a visibility attribute (`PUBLIC` or `PRIVATE`):

| Visibility | Who Can View |
|------------|-------------|
| **PUBLIC** | Any authenticated user |
| **PRIVATE** | Author, Admin, Lead (in their contests), Team members (same contest) |

* Default visibility: **PRIVATE**
* Only the author can change visibility

## Virtual Object: System Configuration

The Submission Management module uses a Virtual Object for system-wide configuration. See [Submit solution spec](Submit%20solution/spec.md) for details on the Virtual Object structure.

**Key Configuration Values**:
* Maximum file size: 1MB
* Rate limit: 1 second
* Language-specific maximum limits (timeLimit, memoryLimit)

## Related Modules

* **Problem Management**: Problem creation with languageOverrides
* **Contest Management**: Contest registration and standings
* **[Judge System](../Judge%20System/README.md)**: Asynchronous judging of submissions (microservice)

## Key Design Decisions

### Why language-specific limits?

* **Flexibility**: Different languages have different performance characteristics
* **Fairness**: Allows problems to be fair across languages (e.g., Python gets more time)
* **Extensibility**: Easy to add new languages with their own limits

### Why Virtual Object for configuration?

* **Flexibility**: Limits can be adjusted without code changes
* **Maintainability**: Centralized configuration management
* **Scalability**: Easy to add new languages and adjust limits

### Why SHA256 for duplicate detection?

* **Security**: Better collision resistance than MD5
* **Reliability**: Detects exact duplicate files
* **Performance**: Fast enough for real-time validation

### Why asynchronous judging?

* **Performance**: Doesn't block submission creation
* **Scalability**: Can handle high submission volumes
* **User Experience**: Users get immediate confirmation

---

## Implementation Dependencies

```
Problem Management (base)
    ↓
Create Problem with languageOverrides ✅
    ↓
Submit Solution ✅ ← (requires languageOverrides)
    ↓
Judge System ✅ ← (microservice, evaluates submissions)
    ↓
Standing Updates (in Contest Management)
```

