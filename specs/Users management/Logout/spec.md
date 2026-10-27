# Feature Specification: Logout

**Created**: 2026-08-11

## User Scenarios & Testing *(mandatory)*

### User Story 1 – End the current session on demand (Priority: P1)

As a logged-in user, I want to click "Log out" and have my session actually end on the server, so that the access token and refresh token I was using stop working immediately instead of quietly staying valid until they expire on their own.

**Why this priority**: Before this feature, no endpoint revoked the `refresh_token` server-side — the cookie stayed alive and a silent [`POST /auth/refresh`](../Refresh%20session/spec.md) on page reload could revive the "logged out" session. This is the gap that made "Log out" a purely client-side illusion.

**Independent Test**: Can be tested independently by logging in to obtain a refresh token cookie and an access token, calling `POST /auth/logout`, and then validating that (a) the previously-issued access token is rejected on its next authenticated request and (b) `POST /auth/refresh` with the old cookie no longer works.

**Acceptance Scenarios**:

1. **Scenario**: Successful logout with an active session
   - **Given** a user has a valid refresh token cookie from a previous login
   - **When** the client calls `POST /auth/logout`
   - **Then** the system returns 204 No Content
   - **And** the refresh token's entire family is revoked in storage (see [Refresh Session](../Refresh%20session/spec.md))
   - **And** the session's `sid` is invalidated for access-token purposes (see `docs/plan_PR7a_session_scoped_revocation.md`, `SessionInvalidator.InvalidateSession`), so any access token already issued for that session is rejected on its very next use — it does not have to wait out its remaining `exp`
   - **And** the response clears the `refresh_token` cookie (`Set-Cookie` with `Max-Age=-1`)

2. **Scenario**: Logout with no cookie present
   - **Given** the request carries no `refresh_token` cookie (already logged out, or never logged in)
   - **When** `POST /auth/logout` is called
   - **Then** the system returns 204 No Content
   - **And** the response still clears the `refresh_token` cookie
   - **And** no storage lookup is attempted

3. **Scenario**: Logout with an already-revoked or unrecognized token
   - **Given** the cookie's token does not resolve to any active row in storage (already logged out from this device, or a stale/garbage cookie value)
   - **When** `POST /auth/logout` is called
   - **Then** the system returns 204 No Content
   - **And** the response clears the cookie
   - **And** no error is surfaced — logout must never fail a user for "you were already logged out"

4. **Scenario**: Underlying revocation fails
   - **Given** the token is resolved and active, but persisting the revocation fails (Postgres or Redis unavailable)
   - **When** `POST /auth/logout` is called
   - **Then** the system returns 500 Internal Server Error (both the Postgres and Redis adapters translate failures to a generic internal error, never a distinguishable 503)
   - **And** the cookie is NOT cleared — the client keeps presenting the same (still-valid) token and the frontend may retry
   - **And** retrying the same request completes the logout without leaving a partially-revoked session, regardless of how far the previous attempt got (see FR-004)

---

### Edge Cases

- Tampered or forged cookie value that fails signature verification — treated identically to "no cookie" (Scenario 2): 204, cookie cleared, no storage lookup, no information leaked about why it failed.
- Logout called twice in a row (double-click, or a retry after a slow-but-successful first request) — the second call resolves the token as already revoked and behaves like Scenario 3.
- Logout for a session that a different feature already invalidated in the meantime (password change, admin deactivation — see [Login](../Login/spec.md) Session Invalidation Context) — behaves like Scenario 3, not an error.
- Logout does not affect the user's other sessions/devices — only the family tied to the presented cookie is revoked; "log out everywhere" is a distinct, out-of-scope action (see below).

## API Contract

### POST /auth/logout

End the current session: revoke its refresh token family and invalidate its access-token session ID.

> **Important**: Public endpoint in the sense that it requires no `Authorization` header — it acts on whatever `refresh_token` cookie is present, mirroring [`POST /auth/refresh`](../Refresh%20session/spec.md). Not rate-limited: revoking one's own session is not attacker-valuable surface (at worst, an attacker who already controls the cookie can only end the session they already control).

**Headers**:
| Header | Type | Required | Description |
|--------|------|----------|-------------|
| Cookie | string | No | `refresh_token`, if present, set automatically by the browser (not read or set manually by client code) |

**Request Body**: none.

**Responses**:

#### 204 No Content
Always returned when the request is well-formed, regardless of whether there was an active session to revoke — idempotent, and does not leak whether the cookie corresponded to a real session.

**Set-Cookie**:
| Cookie | Attributes | Description |
|--------|-----------|--------------|
| `refresh_token` | `HttpOnly; Secure; SameSite=Strict; Path=/auth; Max-Age=-1` | Instructs the browser to delete the cookie immediately. |

#### 500 Internal Server Error
The revocation could not be completed because a dependency (Postgres or Redis) failed. The cookie is left untouched so the client can retry with the same token.

```json
{
  "error": "INTERNAL_ERROR",
  "message": "An unexpected error occurred"
}
```

---

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose `POST /auth/logout`, requiring no `Authorization` header, acting on the `refresh_token` cookie if present.
- **FR-002**: The system MUST return 204 No Content whenever the request is well-formed, whether or not a session was actually revoked (no cookie, unrecognized token, already-revoked token, or a genuinely active session all resolve to 204).
- **FR-003**: On a genuinely active session, the system MUST both (a) invalidate the session's `sid` for access-token purposes (see `docs/plan_PR7a_session_scoped_revocation.md`, `SessionInvalidator.InvalidateSession`) and (b) revoke every token in the refresh token family (see [Refresh Session](../Refresh%20session/spec.md)) — both are required; neither alone stops both token types.
- **FR-004**: The two revocation steps in FR-003 MUST be ordered so a failure partway through is recoverable by a client retry without a silently-incomplete session: the access-token-side invalidation MUST happen before the refresh-token-family revocation, so that if the second step fails, the token presented on retry is still found active and the full sequence runs again — reversing the order would let a retry short-circuit on "already revoked" while the access-token side was never invalidated.
- **FR-005**: The system MUST clear the `refresh_token` cookie (`Max-Age=-1`) only after both revocation steps succeed (or there was nothing to revoke) — never on a failed revocation, so the client's retry still presents a usable token.
- **FR-006**: The system MUST NOT distinguish, in its response, between "no cookie," "unrecognized token," and "already revoked" — all three are silent no-ops, consistent with [Refresh Session](../Refresh%20session/spec.md)'s policy of not leaking token validity information.
- **FR-007**: The system MUST NOT rate-limit this endpoint — revoking a session one already controls is not an abuse vector.
- **FR-008**: Logout MUST NOT affect any other session (family) belonging to the same user — only the one tied to the presented cookie. "Log out on all devices" is a distinct action, already possible via the existing all-sessions invalidation path (see [Login](../Login/spec.md) Session Invalidation Context), but is not exposed by this endpoint.

### Key Entities

- **RefreshToken**: as defined in [Refresh Session](../Refresh%20session/spec.md).
- **SessionInvalidator**: introduced by `docs/plan_PR7a_session_scoped_revocation.md`; this feature only calls its existing `InvalidateSession` operation.

This feature triggers existing revocation operations (`RevokeByFamilyID`, `InvalidateSession`) — it introduces no new persisted state.

> **Out of scope for this spec**: a "log out on all devices" endpoint (would call `InvalidateAllUserSessions` + revoke-all-by-user instead of the single-family operations used here) — the underlying mechanism already exists for other features (password change, deactivation) but is not wired to a user-facing logout action in this spec.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: After calling `POST /auth/logout`, an access token issued for that session is rejected on its very next use, instead of remaining valid for up to its remaining 1-hour lifetime.
- **SC-002**: After calling `POST /auth/logout`, `POST /auth/refresh` with the same (now-revoked) cookie fails with 401, identical to any other revoked-family case.
- **SC-003**: Calling `POST /auth/logout` with no cookie, a garbage cookie, or an already-used cookie never returns an error status — always 204.
- **SC-004**: A logout request that fails partway through (one of the two revocation steps succeeds, the other doesn't) never leaves the session in a state where a client retry can't complete it — every retry either finishes the logout or confirms it was already complete.
- **SC-005**: Logging out from one device never revokes the refresh token family of a session belonging to a different device/login.
