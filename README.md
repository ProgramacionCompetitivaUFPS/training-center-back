# Training & Judge Center - Platform Configuration

This document defines the global configuration for the Training & Judge Center platform.

---

## Virtual Object Configuration

The Virtual Object contains platform-wide settings that constrain problem limits and other global parameters.

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

### Supported Languages

| Language ID | Description |
|-------------|-------------|
| cpp20 | C++ 20 |
| java17 | Java 17 |
| python310 | Python 3.10 |

> **Note**: Additional languages may be added in the future. The platform will validate that `languageOverrides` entries have valid `language` identifiers.

---

## Related Specifications

- [Create Problem](./Problem%20management/Create%20problem/spec.md)
- [Update Problem](./Problem%20management/Update%20problem/spec.md)
- [Submit Solution](./Submission%20management/Submit%20solution/spec.md)

