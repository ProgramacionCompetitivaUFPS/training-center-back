# Feature Specification: Manage Judge Pool (Admin)

**Created**: 2026-02-07

## User Scenarios & Testing *(mandatory)*

### User Story 1 - View Pool Status (Priority: P1)

As an Admin, I want to view the current status of the judge container pool so that I can monitor system capacity and performance.

**Acceptance Scenarios**:

1. **Scenario**: Admin views pool status
   * **Given** requesting user has Admin role
   * **When** Admin requests GET `/admin/judge/pool-status`
   * **Then** system returns current container counts per language
   * **And** response includes active, idle, and queued counts

2. **Scenario**: Non-admin attempts to view pool status
   * **Given** requesting user is not Admin
   * **When** they try to view pool status
   * **Then** system rejects with 403 (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 2 - View Pool Configuration (Priority: P1)

As an Admin, I want to view the current pool configuration so that I can understand the current scaling parameters.

**Acceptance Scenarios**:

1. **Scenario**: Admin views pool config
   * **Given** requesting user has Admin role
   * **When** Admin requests GET `/admin/judge/pool-config`
   * **Then** system returns current container pool settings
   * **And** response includes min/max per language and scaling parameters

---

### User Story 3 - Update Pool Configuration (Priority: P1)

As an Admin, I want to update pool configuration at runtime so that I can adjust capacity without redeploying the application.

**Acceptance Scenarios**:

1. **Scenario**: Admin updates container limits
   * **Given** requesting user has Admin role
   * **When** Admin requests PUT `/admin/judge/pool-config` with new limits
   * **Then** system validates the configuration
   * **And** applies changes gradually (no service interruption)
   * **And** returns updated configuration

2. **Scenario**: Scale up triggered by config change
   * **Given** current pool has 1 cpp20 container
   * **When** Admin updates min to 3
   * **Then** system creates 2 additional containers
   * **And** new containers are available within 60 seconds

3. **Scenario**: Scale down triggered by config change
   * **Given** current pool has 5 idle cpp20 containers
   * **When** Admin updates max to 2
   * **Then** system marks 3 containers as "draining"
   * **And** draining containers complete current work before shutdown
   * **And** no submissions are lost

4. **Scenario**: Invalid configuration rejected
   * **Given** Admin provides invalid config (min > max)
   * **When** update is attempted
   * **Then** system rejects with 400 (`VALIDATION_ERROR`)

---

## Requirements *(mandatory)*

### Functional Requirements

* **FR-001**: System MUST allow Admins to view pool status via API.
* **FR-002**: System MUST allow Admins to view pool configuration via API.
* **FR-003**: System MUST allow Admins to update pool configuration at runtime.
* **FR-004**: System MUST validate configuration changes before applying.
* **FR-005**: System MUST apply configuration changes gradually without service interruption.
* **FR-006**: System MUST restrict all pool management to Admin role only.
* **FR-007**: System MUST maintain minimum containers per language at all times.
* **FR-008**: System MUST not exceed maximum containers per language.
* **FR-009**: System MUST drain busy containers before shutdown (no submission loss).

---

## API Contract

### GET /admin/judge/pool-status

View current pool status.

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Authorization | string | Yes | Bearer token for Admin authentication |

**Response 200 OK**:

```json
{
  "pools": {
    "cpp20": {
      "active": 2,
      "idle": 1,
      "queued": 5,
      "draining": 0
    },
    "java17": {
      "active": 0,
      "idle": 1,
      "queued": 0,
      "draining": 0
    },
    "python310": {
      "active": 1,
      "idle": 0,
      "queued": 3,
      "draining": 0
    }
  },
  "lastUpdated": "2026-02-07T15:30:00Z"
}
```

---

### GET /admin/judge/pool-config

View current pool configuration.

**Response 200 OK**:

```json
{
  "containerPool": {
    "cpp20": { "min": 1, "max": 10 },
    "java17": { "min": 1, "max": 5 },
    "python310": { "min": 1, "max": 5 }
  },
  "scaling": {
    "scaleUpThreshold": 3,
    "scaleDownDelayMinutes": 5,
    "cooldownSeconds": 30
  }
}
```

---

### PUT /admin/judge/pool-config

Update pool configuration.

**Request Body**:

```json
{
  "containerPool": {
    "cpp20": { "min": 2, "max": 15 },
    "java17": { "min": 1, "max": 5 },
    "python310": { "min": 1, "max": 5 }
  },
  "scaling": {
    "scaleUpThreshold": 5,
    "scaleDownDelayMinutes": 10,
    "cooldownSeconds": 60
  }
}
```

| Field | Type | Required | Constraints |
|-------|------|----------|-------------|
| containerPool | object | Yes | At least one language must be configured |
| containerPool.<lang>.min | integer | Yes | 1 ≤ min ≤ max |
| containerPool.<lang>.max | integer | Yes | min ≤ max ≤ 50 |
| scaling.scaleUpThreshold | integer | Yes | 1 ≤ value ≤ 100 |
| scaling.scaleDownDelayMinutes | integer | Yes | 1 ≤ value ≤ 60 |
| scaling.cooldownSeconds | integer | Yes | 10 ≤ value ≤ 300 |

**Response 200 OK**:

```json
{
  "containerPool": { ... },
  "scaling": { ... },
  "appliedAt": "2026-02-07T15:30:00Z"
}
```

**Response 400 Bad Request**:

```json
{
  "error": "VALIDATION_ERROR",
  "message": "min cannot be greater than max for cpp20",
  "details": {
    "field": "containerPool.cpp20.min",
    "value": 10,
    "constraint": "min ≤ max"
  }
}
```

---

## Data Model

### JudgePoolConfig (Database)

| Field | Type | Description |
|-------|------|-------------|
| id | string | Always "default" (singleton) |
| containerPool | json | Min/max per language |
| scaling | json | Scaling parameters |
| updatedAt | timestamp | Last update time |
| updatedBy | string | Admin user ID |

---

## Notes / Implementation hints

* Store configuration in database for persistence across restarts
* Use database change notification or polling (60s) to detect config changes
* Apply changes gradually: drain → destroy → create
* Log all configuration changes for audit
* Consider using Kubernetes HPA for production scaling
* Default configuration should be loaded from VObject if database is empty

