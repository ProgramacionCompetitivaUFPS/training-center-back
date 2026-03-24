# Judge System - Business Logic & Design

**Created**: 2026-01-24

This document centralizes the complete business logic of the Judge System and design considerations being applied across all related specs.

---

## 🔹 General Concept

* **Judge System** is a microservice responsible for evaluating submissions.
* Receives submissions from a **message queue** (Pub/Sub).
* Executes code in isolated **Docker containers** with resource limits.
* Returns verdicts and updates submission status in the database.
* Updates **contest standings** when applicable.

---

## 🔹 Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                      JUDGE SYSTEM                               │
│                                                                 │
│   ┌──────────────┐     ┌──────────────┐     ┌──────────────┐    │
│   │   Backend    │────▶│  Message     │────▶│   Worker     │    │
│   │   API        │     │  Queue       │     │   Pool       │    │
│   │              │     │  (Pub/Sub)   │     │              │    │
│   └──────────────┘     └──────────────┘     └──────────────┘    │
│                                                    │            │
│                                              ┌─────▼─────┐      │
│                                              │  Docker   │      │
│                                              │ Container │      │
│                                              │ (gVisor)  │      │
│                                              └───────────┘      │
└─────────────────────────────────────────────────────────────────┘
```

### Components

| Component | Responsibility |
|-----------|----------------|
| **Backend API** | Creates submissions, enqueues to message queue |
| **Message Queue** | Priority queue for submissions (Pub/Sub) |
| **Worker Pool** | Multiple workers consuming from queue |
| **Docker Container** | Isolated execution environment per submission |

---

## 🔹 Judging Flow

```
1. RECEIVE submission from queue
      │
2. FETCH submission data from DB
      │
3. DOWNLOAD source code from Cloud Storage
      │
4. COMPILE (if needed)
      │ ├── Success → Continue
      │ └── Failure → COMPILATION_ERROR (done)
      │
5. FOR EACH test case:
      │ ├── Execute in Docker container
      │ ├── Capture stdout, stderr, exit code
      │ ├── Check time limit
      │ ├── Check memory limit
      │ └── Compare output (or run checker)
      │
6. DETERMINE final verdict
      │
7. UPDATE submission in DB
      │
8. UPDATE standing (if contest + submittedAt ≤ endTime)
      │
9. DONE
```

---

## 🔹 Submission States

| State | Description |
|-------|-------------|
| `PENDING` | Queued for judging, not yet started |
| `RUNNING` | Currently being judged |
| `ACCEPTED` | All test cases passed |
| `WRONG_ANSWER` | Output does not match expected |
| `PRESENTATION_ERROR` | Output format incorrect (whitespace, newlines) |
| `TIME_LIMIT_EXCEEDED` | Execution exceeded time limit |
| `MEMORY_LIMIT_EXCEEDED` | Execution exceeded memory limit |
| `RUNTIME_EXCEPTION` | Program crashed during execution |
| `COMPILATION_ERROR` | Code failed to compile |

### State Transitions

```
PENDING ──(worker picks up)──> RUNNING
                                  │
                    ┌─────────────┴─────────────┐
                    ▼                           ▼
              COMPILATION_ERROR            (execution)
                                               │
                         ┌──────────┬──────────┼──────────┬──────────┐
                         ▼          ▼          ▼          ▼          ▼
                    ACCEPTED    WRONG_ANSWER   TLE       MLE         RE
                                     │
                                     ▼
                              PRESENTATION_ERROR
```

---

## 🔹 Verdict Priority

When a submission has multiple issues, the verdict is determined by this priority (highest first):

| Priority | Verdict | Condition |
|----------|---------|-----------|
| 1 | `COMPILATION_ERROR` | Code does not compile |
| 2 | `RUNTIME_EXCEPTION` | Program crashed (SIGSEGV, SIGFPE, etc.) |
| 3 | `TIME_LIMIT_EXCEEDED` | Execution time > limit |
| 4 | `MEMORY_LIMIT_EXCEEDED` | Memory usage > limit |
| 5 | `PRESENTATION_ERROR` | Output format incorrect |
| 6 | `WRONG_ANSWER` | Output content incorrect |
| 7 | `ACCEPTED` | All test cases passed |

### PRESENTATION_ERROR vs WRONG_ANSWER

| Issue | Verdict |
|-------|---------|
| Extra spaces at end of line | PRESENTATION_ERROR |
| Extra blank line at end | PRESENTATION_ERROR |
| Missing newline at end | PRESENTATION_ERROR |
| Wrong numeric format (1.0 vs 1) | PRESENTATION_ERROR |
| Completely wrong output | WRONG_ANSWER |
| Partial correct output | WRONG_ANSWER |

---

## 🔹 Supported Languages

### C++20

| Property | Value |
|----------|-------|
| Language ID | `cpp20` |
| Compiler | g++ 12+ |
| Compile Command | `g++ -std=c++20 -O2 -Wall -o solution solution.cpp` |
| Run Command | `./solution` |
| File Extension | `.cpp` |
| Compile Timeout | 30 seconds |
| Compile Memory | 512 MB |

### Java 17

| Property | Value |
|----------|-------|
| Language ID | `java17` |
| Compiler | OpenJDK 17 (javac) |
| Compile Command | `javac -encoding UTF-8 Solution.java` |
| Run Command | `java -Xmx{memoryLimit}m -Xss64m Solution` |
| File Extension | `.java` |
| Compile Timeout | 30 seconds |
| Compile Memory | 512 MB |
| Note | Class must be named `Solution` |

### Python 3.10

| Property | Value |
|----------|-------|
| Language ID | `python310` |
| Interpreter | PyPy 3.10 |
| Compile Command | (none - interpreted) |
| Run Command | `pypy3 solution.py` |
| File Extension | `.py` |
| Syntax Check | 5 seconds |

---

## 🔹 Resource Limits

### Execution Limits

Limits are defined per problem with optional language-specific overrides.

```json
{
  "timeLimit": 2000,        // Default: 2 seconds (milliseconds)
  "memoryLimit": 256,       // Default: 256 MiB
  "languageOverrides": [
    { "language": "python310", "timeLimit": 4000 },
    { "language": "java17", "memoryLimit": 512 }
  ]
}
```

### Compilation Limits (Fixed)

| Limit | Value |
|-------|-------|
| Time | 30 seconds |
| Memory | 512 MB |
| Output | 10 MB |

### Execution Constraints (Fixed)

| Constraint | Value |
|------------|-------|
| Max processes | 1 |
| Network | Disabled |
| Filesystem | Read-only (except /tmp) |
| Max output size | 64 MB |

---

## 🔹 Docker Container Configuration

### Base Image

```dockerfile
FROM ubuntu:22.04

# Install compilers and runtimes
RUN apt-get update && apt-get install -y \
    g++-12 \
    openjdk-17-jdk \
    pypy3 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN useradd -m -s /bin/bash judge
USER judge
WORKDIR /home/judge
```

### Container Limits (cgroups)

```yaml
resources:
  limits:
    cpu: "1"                    # 1 CPU core
    memory: "{memoryLimit}Mi"   # From problem config
  requests:
    cpu: "0.5"
    memory: "128Mi"
```

### Security Configuration

```yaml
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop:
      - ALL
```

---

## 🔹 Test Case Execution

### Input/Output Handling

```
1. Write input to stdin of process
2. Capture stdout to file
3. Capture stderr to file (for debugging)
4. Wait for process to complete or timeout
5. Read output file
6. Compare with expected output (or run checker)
```

### Time Measurement

- **Wall clock time**: Total elapsed time (used for TLE detection)
- **CPU time**: Actual CPU usage (logged but not primary metric)

### Memory Measurement

- **Peak RSS**: Maximum resident set size during execution
- Measured via cgroups memory.max_usage_in_bytes

---

## 🔹 Output Comparison

### Default Comparison (No Checker)

Line-by-line comparison with these rules:

1. Split output into lines
2. Trim trailing whitespace from each line
3. Remove trailing empty lines
4. Compare line by line

```python
def compare(expected, actual):
    exp_lines = [line.rstrip() for line in expected.strip().split('\n')]
    act_lines = [line.rstrip() for line in actual.strip().split('\n')]
    
    if exp_lines == act_lines:
        return "ACCEPTED"
    
    # Check for presentation errors
    if normalize(expected) == normalize(actual):
        return "PRESENTATION_ERROR"
    
    return "WRONG_ANSWER"
```

### Custom Checker

When problem has a checker:

```bash
./checker input.txt expected.txt actual.txt
# Exit code 0 = ACCEPTED
# Exit code 1 = WRONG_ANSWER
# Exit code 2 = PRESENTATION_ERROR (optional)
```

Checker receives:
- `input.txt`: Test case input
- `expected.txt`: Expected output (may be ignored)
- `actual.txt`: User's output

---

## 🔹 Queue Priority

| Priority | Type | Value |
|----------|------|-------|
| 🔴 High | Contest ACTIVE (normal submission) | 1 |
| 🔴 High | Contest ACTIVE (rejudge) | 1 |
| 🟡 Medium | Postcompetition submission | 2 |
| 🟢 Normal | Practice submission | 3 |
| 🔵 Low | Bulk rejudge (entire problem) | 4 |

### Queue Implementation (Pub/Sub)

```
Topic: judge-submissions
  ├── Subscription: high-priority (filter: priority <= 1)
  ├── Subscription: medium-priority (filter: priority == 2)
  ├── Subscription: normal-priority (filter: priority == 3)
  └── Subscription: low-priority (filter: priority >= 4)
```

Workers consume from higher priority subscriptions first.

---

## 🔹 Standing Updates

### When to Update Standing

| Condition | Update Standing |
|-----------|-----------------|
| Contest ACTIVE + submittedAt ≤ endTime | ✅ Yes |
| Contest FINISHED | ❌ No (frozen) |
| Postcompetition (submittedAt > endTime) | ❌ No |
| Practice (no contest) | ❌ N/A |

### Update Logic

```python
if submission.contest_id and submission.submittedAt <= contest.endTime:
    if contest.status == "ACTIVE":
        update_standing(submission)
```

---

## 🔹 Error Handling

### Transient Errors (Retry)

| Error | Action |
|-------|--------|
| Container failed to start | Retry up to 3 times |
| Network error fetching test cases | Retry with backoff |
| Database connection lost | Retry with backoff |

### Permanent Errors (Fail)

| Error | Action |
|-------|--------|
| Source code not found | Mark as SYSTEM_ERROR |
| Problem data corrupted | Mark as SYSTEM_ERROR |
| Max retries exceeded | Mark as SYSTEM_ERROR |

### SYSTEM_ERROR

A special verdict for internal errors:
- Not shown to users as a normal verdict
- Logged for investigation
- Submission can be manually rejudged

---

## 🔹 Submission Result Data

Data stored after judging:

| Field | Type | Description |
|-------|------|-------------|
| `status` | enum | Final verdict |
| `executionTime` | integer | Max execution time across all cases (ms) |
| `memoryUsed` | integer | Max memory used across all cases (KB) |
| `judgedAt` | timestamp | When judging completed |
| `compilationLog` | string | Only for COMPILATION_ERROR |

**Note**: We do NOT store:
- Which test case failed
- User's output
- Per-case execution times

---

## 🔹 Related Specs

### Implemented Specs

1. **[Judge submission](Judge%20submission/spec.md)** - Core judging logic and worker behavior

### Related Specs (Other Modules)

* **[Rejudge submissions](../Problem%20management/Rejudge%20submissions/spec.md)** - Re-evaluate submissions (individual, bulk, global) - in Problem Management module

### Future Specs (Planned)

* **Judge metrics** - Monitoring and performance metrics

### Implementation Dependencies

```
Submission Management
    ↓
Submit Solution ✅ ← (creates submission, enqueues)
    ↓
Judge System
    ↓
Judge Submission ✅ ← (evaluates, updates status)
    ↓
Contest Management
    ↓
Standing Updates ← (when applicable)
```

---

## 🔹 Key Design Decisions

### Why Docker for MVP?

* **Simplicity**: Easy to set up and maintain
* **Portability**: Works anywhere Docker runs
* **Security**: Good isolation with gVisor
* **GCP Integration**: Native support in Cloud Run/GKE

### Why Pub/Sub for queue?

* **Managed**: No infrastructure to maintain
* **Priority**: Multiple subscriptions for priority
* **Scalability**: Handles high throughput
* **GCP Native**: Integrates with Cloud Run

### Why not show which test case failed?

* **Fairness**: Prevents reverse-engineering test cases
* **Simplicity**: Less data to store and transmit
* **Standard**: Most competitive platforms do this

### Why PyPy instead of CPython?

* **Performance**: 5-10x faster for competitive programming
* **Compatibility**: 99% compatible with CPython
* **Fairness**: Reduces language disadvantage

### Why fixed compilation limits?

* **Security**: Prevents compilation bombs
* **Simplicity**: Same limits for all problems
* **Predictability**: Consistent behavior

---

*This document should be updated when new design decisions are made or additional specs are implemented.*

