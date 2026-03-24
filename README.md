# Training & Judge Center - Platform Configuration

This document defines the global configuration for the Training & Judge Center platform.

---

## Virtual Object Configuration

The Virtual Object contains platform-wide settings that constrain problem limits, sandbox configuration, and other global parameters. These settings are **static** and require a deployment to change (for security reasons).

### Time and Memory Limits

```json
{
  "maxTimeLimitGlobal": 300000,
  "maxMemoryLimitGlobal": 2048,
  "languageOverrides": [
    { "language": "cpp20", "maxTimeLimit": 300000, "maxMemoryLimit": 2048 },
    { "language": "java17", "maxTimeLimit": 400000, "maxMemoryLimit": 2048 },
    { "language": "python310", "maxTimeLimit": 600000, "maxMemoryLimit": 2048 }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| maxTimeLimitGlobal | integer | Maximum time limit in milliseconds that can be set as the default for a problem (applies to all languages unless overridden) |
| maxMemoryLimitGlobal | integer | Maximum memory limit in MiB that can be set as the default for a problem (applies to all languages unless overridden) |
| languageOverrides | array | Language-specific maximum limits. Allows certain languages (e.g., interpreted languages) to have higher maximum limits. |

### Language-Specific Overrides

Each entry in the `languageOverrides` array can specify:

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| language | string | Yes | Language identifier (e.g., `cpp20`, `java17`, `python310`) |
| maxTimeLimit | integer | No | Maximum time limit in milliseconds for this specific language |
| maxMemoryLimit | integer | No | Maximum memory limit in MiB for this specific language |

### How Limits Work

1. **Problem Default Limits**: When creating/updating a problem, `timeLimit` and `memoryLimit` define the default limits for all languages. These must not exceed `maxTimeLimitGlobal` and `maxMemoryLimitGlobal`.

2. **Problem Language Overrides**: Problems can specify `languageOverrides` to set different limits for specific languages. Each override must not exceed the corresponding maximum from the Virtual Object's `languageOverrides` for that language.

3. **Validation Flow**:
   - `problem.timeLimit` ≤ `virtualObject.maxTimeLimitGlobal`
   - `problem.memoryLimit` ≤ `virtualObject.maxMemoryLimitGlobal`
   - For each entry in `problem.languageOverrides`:
     - `entry.timeLimit` ≤ `virtualObject.languageOverrides[entry.language].maxTimeLimit`
     - `entry.memoryLimit` ≤ `virtualObject.languageOverrides[entry.language].maxMemoryLimit`

### Example: Creating a Problem with Limits

```json
{
  "title": "Sum of Two Numbers",
  "statement": "Given two integers a and b, return their sum.",
  "timeLimit": 2000,
  "memoryLimit": 256,
  "languageOverrides": [
    { "language": "python310", "timeLimit": 4000 },
    { "language": "java17", "memoryLimit": 512 }
  ]
}
```

In this example:
- C++ submissions use: 2000ms, 256 MiB (defaults)
- Python submissions use: 4000ms, 256 MiB (time overridden)
- Java submissions use: 2000ms, 512 MiB (memory overridden)

---

## Sandbox Configuration (Static)

The sandbox configuration defines security-critical settings for code execution. These settings are **static** and require a deployment to change to prevent accidental or malicious misconfiguration.

```json
{
  "sandbox": {
    "path": "/sandbox",
    "sizeMB": 64,
    "useTmpfs": true,
    "maxOutputSizeMB": 64,
    "maxProcesses": 1,
    "networkEnabled": false,
    "readOnlyFilesystem": true
  },
  "compilation": {
    "timeoutSeconds": 30,
    "memoryLimitMB": 512,
    "outputLimitMB": 10
  },
  "checker": {
    "timeoutSeconds": 30
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| sandbox.path | string | Directory where user code executes (only writable location) |
| sandbox.sizeMB | integer | Maximum size of sandbox directory (tmpfs limit) |
| sandbox.useTmpfs | boolean | If true, sandbox uses RAM-based tmpfs for faster I/O and instant cleanup |
| sandbox.maxOutputSizeMB | integer | Maximum size of user program output |
| sandbox.maxProcesses | integer | Maximum number of processes (1 = no fork bombs) |
| sandbox.networkEnabled | boolean | Whether network access is allowed (always false for security) |
| sandbox.readOnlyFilesystem | boolean | Whether filesystem is read-only except /sandbox |
| compilation.timeoutSeconds | integer | Maximum time for compilation phase |
| compilation.memoryLimitMB | integer | Maximum memory for compilation |
| compilation.outputLimitMB | integer | Maximum compilation log size |
| checker.timeoutSeconds | integer | Maximum time for custom checker execution |

---

## Supported Languages

Each language has a dedicated container image that may include multiple compiler versions.

### C++ Family (image: `judge-runner:cpp`)

| Language ID | Compiler | Compile Command | Run Command |
|-------------|----------|-----------------|-------------|
| cpp17 | g++ 12 | `g++ -std=c++17 -O2 -o solution solution.cpp` | `./solution` |
| cpp20 | g++ 12 | `g++ -std=c++20 -O2 -o solution solution.cpp` | `./solution` |

### Java Family (image: `judge-runner:java`)

| Language ID | Compiler | Compile Command | Run Command |
|-------------|----------|-----------------|-------------|
| java17 | javac 17 | `javac -encoding UTF-8 Solution.java` | `java Solution` |

### Python Family (image: `judge-runner:python`)

| Language ID | Interpreter | Syntax Check | Run Command |
|-------------|-------------|--------------|-------------|
| python310 | pypy3 | `pypy3 -m py_compile solution.py` | `pypy3 solution.py` |

> **Note**: Additional languages and versions may be added in the future. Each language family shares a container image for efficient resource usage. The platform validates that `languageOverrides` entries have valid `language` identifiers.


---

## Related Specifications

- [Create Problem](./specs/Problem%20management/Create%20problem/spec.md)
- [Update Problem](./specs/Problem%20management/Update%20problem/spec.md)
- [Judge Submission](./specs/Judge%20System/Judge%20submission/spec.md)

