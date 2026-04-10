# Feature Specification: Judge Submission

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Judge a Submission Successfully (Priority: P1)

As the Judge System, I want to evaluate a submission against all test cases so that the user receives a verdict for their solution.

**Why this priority**: Judging submissions is the core functionality of the platform. Without judging, users cannot know if their solutions are correct.

**Independent Test**: Worker receives submission from queue, compiles code, executes against test cases, compares output, and updates submission status in database.

**Acceptance Scenarios**:

1. **Scenario**: Successful judgment - ACCEPTED

* **Given** a submission is in PENDING status in the queue
* **And** the source code compiles successfully
* **When** the worker executes the code against all test cases
* **And** all outputs match expected outputs
* **Then** the submission status is updated to ACCEPTED
* **And** `executionTime` is set to the maximum time across all cases
* **And** `memoryUsed` is set to the maximum memory across all cases
* **And** `judgedAt` is set to current timestamp

2. **Scenario**: Judgment result - WRONG_ANSWER

* **Given** a submission is being judged
* **And** the source code compiles and executes successfully
* **When** the output does not match expected output for any test case
* **And** the difference is not just whitespace/formatting
* **Then** the submission status is updated to WRONG_ANSWER
* **And** execution stops at the first failing test case
* **And** `judgedAt` is set to current timestamp

3. **Scenario**: Judgment result - PRESENTATION_ERROR

* **Given** a submission is being judged
* **And** the source code compiles and executes successfully
* **When** the output content is correct but has formatting issues
* **And** issues are: extra spaces, extra newlines, wrong numeric format
* **Then** the submission status is updated to PRESENTATION_ERROR
* **And** `judgedAt` is set to current timestamp

4. **Scenario**: Judgment result - TIME_LIMIT_EXCEEDED

* **Given** a submission is being judged
* **And** the source code compiles successfully
* **When** execution time exceeds the problem's time limit for the language
* **Then** the submission status is updated to TIME_LIMIT_EXCEEDED
* **And** the process is killed
* **And** `executionTime` is set to the time limit
* **And** `judgedAt` is set to current timestamp

5. **Scenario**: Judgment result - MEMORY_LIMIT_EXCEEDED

* **Given** a submission is being judged
* **And** the source code compiles successfully
* **When** memory usage exceeds the problem's memory limit for the language
* **Then** the submission status is updated to MEMORY_LIMIT_EXCEEDED
* **And** the process is killed
* **And** `memoryUsed` is set to the memory limit
* **And** `judgedAt` is set to current timestamp

6. **Scenario**: Judgment result - RUNTIME_EXCEPTION

* **Given** a submission is being judged
* **And** the source code compiles successfully
* **When** the program crashes during execution (SIGSEGV, SIGFPE, etc.)
* **Then** the submission status is updated to RUNTIME_EXCEPTION
* **And** `judgedAt` is set to current timestamp

7. **Scenario**: Judgment result - COMPILATION_ERROR

* **Given** a submission is being judged
* **When** the source code fails to compile
* **Then** the submission status is updated to COMPILATION_ERROR
* **And** `compilationLog` is set to the compiler output (max 10KB)
* **And** no test cases are executed
* **And** `judgedAt` is set to current timestamp

---

### User Story 2 - Judge with Custom Checker (Priority: P1)

As the Judge System, I want to use a custom checker when the problem has one so that problems with multiple valid solutions can be judged correctly.

**Why this priority**: Many problems have multiple valid answers (e.g., "find any path", "output any valid solution"). Custom checkers are essential for these problems.

**Acceptance Scenarios**:

1. **Scenario**: Checker returns ACCEPTED

* **Given** a problem has a custom checker
* **And** a submission produces output
* **When** the checker is executed with (input, expected, actual)
* **And** the checker exits with code 0
* **Then** the test case is marked as ACCEPTED

2. **Scenario**: Checker returns WRONG_ANSWER

* **Given** a problem has a custom checker
* **And** a submission produces output
* **When** the checker is executed with (input, expected, actual)
* **And** the checker exits with code 1
* **Then** the test case is marked as WRONG_ANSWER

3. **Scenario**: Checker returns PRESENTATION_ERROR

* **Given** a problem has a custom checker
* **And** a submission produces output
* **When** the checker is executed with (input, expected, actual)
* **And** the checker exits with code 2
* **Then** the test case is marked as PRESENTATION_ERROR

4. **Scenario**: Checker crashes or times out

* **Given** a problem has a custom checker
* **When** the checker crashes or exceeds its time limit (30s)
* **Then** the system logs the error
* **And** falls back to exact comparison for this test case

---

### User Story 3 - Update Contest Standing (Priority: P1)

As the Judge System, I want to update the contest standing when a submission is judged so that rankings are kept current during contests.

**Why this priority**: Real-time standings are essential for competitive programming contests.

**Acceptance Scenarios**:

1. **Scenario**: Update standing for ACTIVE contest submission

* **Given** a submission belongs to a contest
* **And** the contest status is ACTIVE
* **And** submission.submittedAt ≤ contest.endTime
* **When** the submission is judged
* **Then** the standing document is updated atomically
* **And** problemsSolved and penalty are recalculated if ACCEPTED

2. **Scenario**: Do NOT update standing for postcompetition submission

* **Given** a submission belongs to a contest
* **And** submission.submittedAt > contest.endTime
* **When** the submission is judged
* **Then** the standing is NOT updated
* **And** the submission is still judged normally

3. **Scenario**: Do NOT update standing for FINISHED contest

* **Given** a submission belongs to a contest
* **And** the contest status is FINISHED
* **And** the standing is frozen
* **When** the submission is judged
* **Then** the standing is NOT updated

4. **Scenario**: Do NOT update standing for practice submission

* **Given** a submission has no contest_id (practice submission)
* **When** the submission is judged
* **Then** no standing update is performed

---

### User Story 4 - Handle Queue Priority (Priority: P2)

As the Judge System, I want to process high-priority submissions first so that contest submissions are judged faster than practice submissions.

**Why this priority**: During contests, users expect quick feedback. Practice submissions can wait.

**Acceptance Scenarios**:

1. **Scenario**: Contest submissions processed before practice

* **Given** the queue has both contest and practice submissions
* **When** a worker is available
* **Then** it picks up the contest submission first
* **And** practice submissions wait until no contest submissions are pending

2. **Scenario**: Postcompetition has medium priority

* **Given** the queue has postcompetition and practice submissions
* **When** a worker is available
* **Then** it picks up the postcompetition submission first

3. **Scenario**: Bulk rejudge has lowest priority

* **Given** the queue has practice submissions and bulk rejudge submissions
* **When** a worker is available
* **Then** it picks up the practice submission first

---

### User Story 5 - Handle Transient Errors (Priority: P2)

As the Judge System, I want to retry on transient errors so that temporary failures don't cause permanent submission failures.

**Why this priority**: Network glitches, container startup failures, and database hiccups should be handled gracefully.

**Acceptance Scenarios**:

1. **Scenario**: Retry on container startup failure

* **Given** the Docker container fails to start
* **When** the worker detects the failure
* **Then** it retries up to 3 times with exponential backoff
* **And** if all retries fail, marks submission as SYSTEM_ERROR

2. **Scenario**: Retry on database connection error

* **Given** the database connection is temporarily unavailable
* **When** the worker tries to update the submission
* **Then** it retries with exponential backoff
* **And** the submission result is not lost

3. **Scenario**: Retry on file download error

* **Given** the source code download from Cloud Storage fails
* **When** the worker detects the failure
* **Then** it retries up to 3 times
* **And** if all retries fail, re-queues the submission

---

### Edge Cases

- Submission deleted while being judged (check before updating).
- Problem becomes DRAFT while submission is judged (complete judging, pause future).
- Contest ends during judging (check submittedAt, not current time).
- Very large output from user program (truncate at 64MB).
- Infinite loop that uses minimal memory (caught by time limit).
- Fork bomb attempt (limited by container process count = 1).
- Network access attempt (blocked by container).
- Filesystem write attempt outside /tmp (blocked by read-only filesystem).
- Multiple workers pick same submission (use message acknowledgment).

---

## Internal Processing Contract

### Message Format (Pub/Sub)

```json
{
  "submissionId": "abc123-def456-...",
  "priority": 1,
  "enqueuedAt": "2026-01-24T10:30:00Z",
  "metadata": {
    "contestId": "contest-123",
    "problemId": "problem-456",
    "userId": "user-789",
    "language": "cpp20"
  }
}
```

### Priority Values

| Priority | Type |
|----------|------|
| 1 | Contest ACTIVE (normal + rejudge) |
| 2 | Postcompetition |
| 3 | Practice |
| 4 | Bulk rejudge |

---

## Judging Process (Detailed)

### Phase 1: Receive and Validate

```
1. Receive message from queue
2. Acknowledge message (prevents duplicate processing)
3. Fetch submission from database
4. Validate submission exists and is PENDING
5. Update status to RUNNING
6. Download source code from Cloud Storage
7. Fetch problem data (limits, test cases, checker)
8. Claim container from pool (by language)
```

### Phase 2: Compilation

```
For compiled languages (C++, Java):

1. Copy source code to /sandbox/ in claimed container
2. Execute compile command with limits:
   - Time: 30 seconds (from VObject.compilation.timeoutSeconds)
   - Memory: 512 MB (from VObject.compilation.memoryLimitMB)
   - Output: 10 MB (from VObject.compilation.outputLimitMB)
3. Capture stdout, stderr, exit code
4. If exit code != 0:
   - Extract compilation log (truncate to 10KB)
   - Set status = COMPILATION_ERROR
   - Clean sandbox: rm -rf /sandbox/*
   - Return container to pool
   - Update database
   - DONE
5. If exit code == 0:
   - Keep compiled binary in /sandbox/
   - Continue to Phase 3

For interpreted languages (Python):

1. Copy source code to /sandbox/
2. Run syntax check: `pypy3 -m py_compile /sandbox/solution.py`
3. If syntax error:
   - Set status = COMPILATION_ERROR
   - Store error message
   - Clean sandbox, return container to pool
   - DONE
4. Continue to Phase 3
```

### Phase 3: Execution

```
For each test case (in order):

1. Container already has limits configured:
   - CPU: 1 core
   - Memory: {problem.memoryLimit} MiB (or language override)
   - Time: {problem.timeLimit} ms (or language override)
   - Processes: 1 (seccomp)
   - Network: disabled
   - Filesystem: read-only except /sandbox

2. Prepare files in /sandbox/:
   - Input: /sandbox/input.txt
   - Solution binary/source already present

3. Execute with timeout:
   - Start timer
   - Run: /sandbox/solution < /sandbox/input.txt > /sandbox/output.txt 2> /sandbox/stderr.txt
   - Wait for completion or timeout

4. Capture metrics:
   - Wall clock time (ms)
   - Peak memory (KB)
   - Exit code
   - Exit signal (if crashed)

5. Determine test case result:
   - If timeout → TLE (stop execution)
   - If memory exceeded → MLE (stop execution)
   - If crashed (signal) → RE (stop execution)
   - If exit code != 0 → RE (stop execution)
   - Else → Compare output

6. Compare output:
   - If problem has checker → Run checker
   - Else → Default comparison
   - Result: AC, WA, or PE

7. If not AC → Stop execution, record verdict
8. If AC → Continue to next test case

9. After all test cases:
   - If all AC → Final verdict = ACCEPTED
   - Else → Final verdict = first non-AC result
```

### Phase 4: Finalization

```
1. Calculate final metrics:
   - executionTime = max(all case times)
   - memoryUsed = max(all case memories)

2. Update submission in database:
   - status = final verdict
   - executionTime
   - memoryUsed
   - judgedAt = now()
   - compilationLog (if CE)

3. If contest submission with submittedAt ≤ endTime:
   - Update standing document atomically

4. Clean up sandbox:
   - rm -rf /sandbox/* (fast cleanup, ~10-20ms)
   - Return container to pool

5. DONE
```

---

## Container Pool Architecture

### Overview

Instead of creating a new container for each submission (high overhead), the system maintains a **persistent pool of containers** organized by language. This eliminates container startup time (~300ms) and replaces it with fast sandbox cleanup (~10-20ms).

```
┌─────────────────────────────────────────────────────────────┐
│                  Container Pool Manager                      │
│                                                              │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐          │
│  │ cpp20 Pool  │  │ java17 Pool │  │python310 Pool│          │
│  │ [min: 1]    │  │ [min: 1]    │  │ [min: 1]    │          │
│  │ [max: 10]   │  │ [max: 5]    │  │ [max: 5]    │          │
│  └─────────────┘  └─────────────┘  └─────────────┘          │
└─────────────────────────────────────────────────────────────┘
```

### Container Lifecycle

```
1. CLAIM    → Worker takes idle container from pool
2. COPY     → Files copied to /sandbox/
3. COMPILE  → (if needed) Compile in /sandbox/
4. EXECUTE  → Run against all test cases
5. CLEANUP  → rm -rf /sandbox/* (10-20ms)
6. RETURN   → Container returned to pool as "idle"
```

### Pool Configuration (Dynamic via Admin API)

Pool settings can be modified at runtime without redeployment via Admin API:

```json
// GET/PUT /admin/judge/pool-config
{
  "containerPool": {
    "cpp20":     { "min": 1, "max": 10 },
    "java17":    { "min": 1, "max": 5 },
    "python310": { "min": 1, "max": 5 }
  },
  "scaling": {
    "scaleUpThreshold": 3,
    "scaleDownDelayMinutes": 5,
    "cooldownSeconds": 30
  }
}
```

| Field | Description |
|-------|-------------|
| `containerPool.<lang>.min` | Minimum containers always running (warm pool) |
| `containerPool.<lang>.max` | Maximum containers during peak load |
| `scaling.scaleUpThreshold` | Create new container if queue depth exceeds this |
| `scaling.scaleDownDelayMinutes` | Wait time before scaling down idle containers |
| `scaling.cooldownSeconds` | Minimum time between scaling operations |

### Pool Status Endpoint

```json
// GET /admin/judge/pool-status
{
  "cpp20": { "active": 2, "idle": 1, "queued": 5 },
  "java17": { "active": 0, "idle": 1, "queued": 0 },
  "python310": { "active": 1, "idle": 0, "queued": 3 }
}
```

### Auto-Scaling Behavior

| Condition | Action |
|-----------|--------|
| `queuedSubmissions > scaleUpThreshold` | Create new container (up to max) |
| `idleContainers > 0` for `scaleDownDelayMinutes` | Destroy excess containers (down to min) |
| Config change detected | Apply changes gradually (drain → destroy → create) |

---

## Docker Container Specification

### Persistent Container Configuration

```yaml
# docker-compose.yml (for development)
version: "3.8"
services:
  judge-cpp20:
    image: judge-runner:cpp20
    runtime: runsc  # gVisor for security
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    read_only: true
    tmpfs:
      - /sandbox:size=64M,mode=1777,uid=1000,gid=1000
    network_mode: none
    deploy:
      resources:
        limits:
          cpus: "1"
          memory: 2048M  # Max from VObject
        reservations:
          cpus: "0.5"
          memory: 128M
    restart: unless-stopped

  judge-java17:
    image: judge-runner:java17
    # ... same security config ...
    
  judge-python310:
    image: judge-runner:python310
    # ... same security config ...
```

### Dockerfile (per language)

```dockerfile
FROM ubuntu:22.04

# Install compiler (example: C++20)
RUN apt-get update && apt-get install -y --no-install-recommends \
    g++-12 \
    && rm -rf /var/lib/apt/lists/* \
    && ln -s /usr/bin/g++-12 /usr/bin/g++

# Create judge user
RUN useradd -m -s /bin/bash -u 1000 judge

# Create sandbox directory (will be tmpfs mount)
RUN mkdir -p /sandbox && chown judge:judge /sandbox

WORKDIR /sandbox
USER judge

# Keep container running, waiting for work
CMD ["sleep", "infinity"]
```

### Security Layers

| Layer | Purpose |
|-------|--------|
| **gVisor (runsc)** | Syscall interception and filtering |
| **read-only filesystem** | Prevent system modification |
| **tmpfs /sandbox** | RAM-based, size-limited, instant cleanup |
| **seccomp** | Block dangerous syscalls (fork, socket, etc.) |
| **rlimits** | CPU time, memory, file size limits |
| **non-root user** | uid=1000 (judge), no privileges |
| **network=none** | No network access |

---

## Output Comparison Algorithm

### Default Comparison

```python
def compare_outputs(expected: str, actual: str) -> str:
    """
    Compare expected and actual outputs.
    Returns: "ACCEPTED", "WRONG_ANSWER", or "PRESENTATION_ERROR"
    """
    # Normalize for strict comparison
    exp_lines = expected.strip().split('\n')
    act_lines = actual.strip().split('\n')
    
    # Strict comparison (trimmed lines)
    exp_trimmed = [line.rstrip() for line in exp_lines]
    act_trimmed = [line.rstrip() for line in act_lines]
    
    if exp_trimmed == act_trimmed:
        return "ACCEPTED"
    
    # Check for presentation error
    # Normalize: collapse whitespace, remove empty lines
    exp_normalized = normalize(expected)
    act_normalized = normalize(actual)
    
    if exp_normalized == act_normalized:
        return "PRESENTATION_ERROR"
    
    return "WRONG_ANSWER"


def normalize(text: str) -> str:
    """Normalize text for lenient comparison."""
    # Remove trailing whitespace from lines
    lines = [line.strip() for line in text.strip().split('\n')]
    # Remove empty lines
    lines = [line for line in lines if line]
    # Join with single newline
    return '\n'.join(lines)
```

### Checker Execution

```python
def run_checker(
    checker_path: str,
    input_file: str,
    expected_file: str,
    actual_file: str,
    timeout: int = 30
) -> str:
    """
    Run custom checker.
    Returns: "ACCEPTED", "WRONG_ANSWER", or "PRESENTATION_ERROR"
    """
    try:
        result = subprocess.run(
            [checker_path, input_file, expected_file, actual_file],
            timeout=timeout,
            capture_output=True
        )
        
        if result.returncode == 0:
            return "ACCEPTED"
        elif result.returncode == 2:
            return "PRESENTATION_ERROR"
        else:
            return "WRONG_ANSWER"
            
    except subprocess.TimeoutExpired:
        logger.error("Checker timeout")
        return "WRONG_ANSWER"  # Fallback
    except Exception as e:
        logger.error(f"Checker error: {e}")
        return "WRONG_ANSWER"  # Fallback
```

---

## Functional Requirements

### Message Processing

- **FR-001**: The worker MUST acknowledge messages before processing to prevent duplicates.
- **FR-002**: The worker MUST process messages in priority order.
- **FR-003**: The worker MUST update submission status to RUNNING immediately.
- **FR-004**: The worker MUST complete judging even if the message times out (re-ack).

### Compilation

- **FR-005**: The system MUST compile C++ with `g++ -std=c++20 -O2 -Wall`.
- **FR-006**: The system MUST compile Java with `javac -encoding UTF-8`.
- **FR-007**: The system MUST verify Python syntax with `pypy3 -m py_compile`.
- **FR-008**: Compilation MUST timeout after 30 seconds.
- **FR-009**: Compilation MUST be limited to 512MB memory.
- **FR-010**: Compilation logs MUST be truncated to 10KB max.

### Execution

- **FR-011**: Each test case MUST run in an isolated container.
- **FR-012**: Execution MUST respect problem's timeLimit (with language overrides).
- **FR-013**: Execution MUST respect problem's memoryLimit (with language overrides).
- **FR-014**: Container MUST have network disabled.
- **FR-015**: Container MUST have read-only filesystem (except /tmp).
- **FR-016**: Container MUST be limited to 1 process.
- **FR-017**: User output MUST be truncated at 64MB.

### Verdict Determination

- **FR-018**: If compilation fails → COMPILATION_ERROR.
- **FR-019**: If process crashes (signal) → RUNTIME_EXCEPTION.
- **FR-020**: If time limit exceeded → TIME_LIMIT_EXCEEDED.
- **FR-021**: If memory limit exceeded → MEMORY_LIMIT_EXCEEDED.
- **FR-022**: If output format wrong → PRESENTATION_ERROR.
- **FR-023**: If output content wrong → WRONG_ANSWER.
- **FR-024**: If all tests pass → ACCEPTED.
- **FR-025**: Execution MUST stop at first non-AC verdict.

### Result Storage

- **FR-026**: The system MUST store final verdict in submission.status.
- **FR-027**: The system MUST store max execution time in submission.executionTime.
- **FR-028**: The system MUST store max memory used in submission.memoryUsed.
- **FR-029**: The system MUST store judgedAt timestamp.
- **FR-030**: The system MUST store compilationLog only for CE.
- **FR-031**: The system MUST NOT store which test case failed.
- **FR-032**: The system MUST NOT store user's output.

### Standing Updates

- **FR-033**: Standing MUST be updated if contest is ACTIVE and submittedAt ≤ endTime.
- **FR-034**: Standing MUST NOT be updated for postcompetition submissions.
- **FR-035**: Standing MUST NOT be updated for practice submissions.
- **FR-036**: Standing updates MUST be atomic.

### Error Handling

- **FR-037**: Transient errors MUST be retried up to 3 times.
- **FR-038**: Permanent errors MUST result in SYSTEM_ERROR status.
- **FR-039**: SYSTEM_ERROR submissions can be manually rejudged.

---

## Non-Functional Requirements

- **NFR-001**: Average judging time MUST be < 30 seconds for typical submissions.
- **NFR-002**: Worker MUST handle container failures gracefully.
- **NFR-003**: Worker MUST clean up resources after each submission.
- **NFR-004**: Worker MUST log all judging steps for debugging.
- **NFR-005**: System MUST scale horizontally by adding workers.
- **NFR-006**: Time measurement precision MUST be at least 10ms.
- **NFR-007**: Memory measurement precision MUST be at least 1KB.

---

## Data Model

### Key Entities

- **Submission**: The solution being judged.
  Key attributes for judging:
  - `id` (UUID)
  - `status` (enum: PENDING → RUNNING → final verdict)
  - `language` (cpp20, java17, python310)
  - `filePath` (Cloud Storage path)
  - `executionTime` (milliseconds, nullable until judged)
  - `memoryUsed` (KB, nullable until judged)
  - `judgedAt` (timestamp, nullable until judged)
  - `compilationLog` (string, only for CE)
  - `problem_id` (FK)
  - `contest_id` (FK, nullable)
  - `submittedAt` (timestamp)

- **Problem**: Contains judging configuration.
  Key attributes:
  - `timeLimit` (milliseconds)
  - `memoryLimit` (MiB)
  - `languageOverrides` (array)
  - `testCasesFileKey` (Cloud Storage path to ZIP)
  - `checkerFileKey` (Cloud Storage path, nullable)

- **Contest**: For standing updates.
  Key attributes:
  - `status` (SCHEDULED, ACTIVE, FINISHED)
  - `endTime` (timestamp)

- **ContestParticipant** (NoSQL): Standing document.
  Key attributes:
  - `contestantId`
  - `problemsSolved`
  - `penalty`
  - `problems` (array of attempts)

---

## Security Considerations

- **SEC-001**: All code execution MUST be in isolated containers.
- **SEC-002**: Containers MUST run as non-root user.
- **SEC-003**: Network access MUST be completely disabled.
- **SEC-004**: Filesystem MUST be read-only except /tmp.
- **SEC-005**: Process count MUST be limited to 1 (prevent fork bombs).
- **SEC-006**: Use gVisor runtime for additional syscall filtering.
- **SEC-007**: Source code MUST NOT be exposed to other users.
- **SEC-008**: Test cases MUST NOT be exposed to users.

---

## Optional Notes

- **Worker Scaling**: Use Kubernetes HPA to scale workers based on queue depth.
- **Monitoring**: Export metrics to Cloud Monitoring (queue depth, judging time, error rate).
- **Logging**: Use structured logging with submission ID for tracing.
- **Test Case Caching**: Cache test cases in worker memory for frequently used problems.
- **Container Pooling**: Consider pre-warming containers for faster startup (advanced).
- **Partial Scoring**: Not implemented in MVP, but checker exit codes can support it later.

