# Feature Specification: Reveal Contest Results

**Created**: 2026-11-19

## User Scenarios & Testing *(mandatory)*

### User Story 1 – Lead reveals the final results (Priority: P1)

As a Lead of a group (or an Admin), I want to unfreeze the standings of a finished contest when I decide, so that participants see the final results only after the awards ceremony.

**Independent Test**: Finish a contest with `freezeMinutes > 0`, check that a participant still gets frozen standings, call `POST /groups/{groupId}/contests/{contestId}/reveal-results` as a Lead, and check that standings are unfrozen for everyone.

**Acceptance Scenarios**:

1. **Scenario**: Lead reveals the results
   * **Given** a contest is FINISHED and has `freezeMinutes > 0`
   * **And** the authenticated user is a Lead of the group (or an Admin)
   * **When** they send the reveal request
   * **Then** the system stores `resultsRevealedAt` with the current time and returns 204 No Content
   * **And** from then on standings are no longer frozen, postcompetition rows are available and the submissions list shows full details to everyone

2. **Scenario**: Results already revealed
   * **Given** the results were already revealed
   * **When** a Lead or Admin sends the request again
   * **Then** the system returns 204 and keeps the first `resultsRevealedAt`

3. **Scenario**: Not a Lead
   * **Given** the authenticated user is neither a Lead of the group nor an Admin
   * **When** they send the request
   * **Then** the system rejects with 403 `INSUFFICIENT_PERMISSIONS`

4. **Scenario**: Contest has not finished
   * **Given** the contest is SCHEDULED or ACTIVE
   * **When** a Lead sends the request
   * **Then** the system rejects with 409 `CONTEST_NOT_FINISHED`

5. **Scenario**: Contest without freeze
   * **Given** the contest has `freezeMinutes = 0`
   * **When** a Lead sends the request
   * **Then** the system rejects with 409 `CONTEST_HAS_NO_FREEZE` (such a contest is never frozen)

6. **Scenario**: Contest not found in the group
   * **Then** the system returns 404 `CONTEST_NOT_FOUND`

### Edge Cases

* A contest can also open its results by itself: `autoRevealMinutes` (see [Create contest](../Create%20contest/spec.md)). The button keeps working in that mode to open them earlier; once the automatic time has passed, the request answers 204 and records nothing.
* Revealing is one-way: there is no way to freeze the results again.
* Leads and Admin can read the real standings of a finished contest at any time with `?realtime=true`, revealed or not.
* Contests that had already finished when this feature was deployed are marked as revealed (`resultsRevealedAt = endTime`).
* Locked contests can be revealed: revealing does not change the contest configuration.

## Requirements *(mandatory)*

### Endpoint

`POST /groups/{groupId}/contests/{contestId}/reveal-results` — authenticated, no body.

| Status | Code | When |
|--------|------|------|
| 204 | | Revealed, or already revealed |
| 403 | `INSUFFICIENT_PERMISSIONS` | Caller is not a Lead of the group nor an Admin |
| 404 | `CONTEST_NOT_FOUND` | Contest does not exist or belongs to another group |
| 409 | `CONTEST_NOT_FINISHED` | Contest has not ended |
| 409 | `CONTEST_HAS_NO_FREEZE` | `freezeMinutes = 0` |

### Functional Requirements

- **FR-001**: A contest with `freezeMinutes > 0` MUST stay frozen after `endTime` until `resultsRevealedAt` is set.
- **FR-002**: Only a Lead of the contest's group or an Admin MAY reveal the results.
- **FR-003**: Revealing MUST be idempotent and irreversible.
- **FR-004**: `GET .../standings` MUST expose `contest.resultsRevealedAt`, so the client knows when to offer the reveal button (`status = FINISHED`, `isFrozen = true`).
- **FR-004.1**: With `autoRevealMinutes` set, the results MUST open by themselves at `endTime + autoRevealMinutes` (`contest.autoRevealAt`).

- **FR-005**: The standings cache needs no invalidation: it stores raw data and the freeze is applied when reading.
