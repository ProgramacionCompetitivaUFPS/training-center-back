# Feature Specification: Change Material Visibility

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Publish Material (Priority: P1)

As the author of a material, I want to publish it so that all group members can see the content.

**Why this priority**: Publishing is essential for making content visible to group members.

**Independent Test**: Author publishes a DRAFT material. Verify status changes to PUBLISHED, publishedAt is set, and material becomes visible to group members.

**Acceptance Scenarios**:

1. **Scenario**: Successful publish by author

* **Given** a user is authenticated as the author of a DRAFT material
* **When** they submit a publish request
* **Then** the system returns 200 OK with updated material
* **And** `status` is changed to `PUBLISHED`
* **And** `publishedAt` is set to current timestamp (if first publish)
* **And** `updatedAt` is updated
* **And** material becomes visible to all group members

2. **Scenario**: Publish already published material (idempotent)

* **Given** a material is already PUBLISHED
* **When** the author publishes it again
* **Then** the system returns 200 OK
* **And** `status` remains `PUBLISHED`
* **And** `publishedAt` remains unchanged (original publish time)

3. **Scenario**: Publish by Admin

* **Given** an Admin is authenticated
* **When** they publish any material (regardless of authorship)
* **Then** the system publishes the material successfully

4. **Scenario**: Attempt to publish by non-author Lead

* **Given** a Lead (not the author) is authenticated
* **When** they attempt to publish another Lead's material
* **Then** the system rejects with 403 Forbidden (`NOT_MATERIAL_AUTHOR`)

---

### User Story 2 - Unpublish Material (Priority: P1)

As the author of a material, I want to unpublish it so that I can make edits without the content being visible to members.

**Why this priority**: Unpublishing allows authors to hide content temporarily for revisions.

**Independent Test**: Author unpublishes a PUBLISHED material. Verify status changes to DRAFT and material is no longer visible to non-author members.

**Acceptance Scenarios**:

1. **Scenario**: Successful unpublish by author

* **Given** a user is authenticated as the author of a PUBLISHED material
* **When** they submit an unpublish request
* **Then** the system returns 200 OK with updated material
* **And** `status` is changed to `DRAFT`
* **And** `updatedAt` is updated
* **And** material is only visible to author (and Admin)
* **And** `publishedAt` remains unchanged (historical record)

2. **Scenario**: Unpublish already draft material (idempotent)

* **Given** a material is already DRAFT
* **When** the author unpublishes it
* **Then** the system returns 200 OK
* **And** `status` remains `DRAFT`

3. **Scenario**: Unpublish pinned material

* **Given** a material is PUBLISHED and pinned
* **When** the author unpublishes it
* **Then** the material becomes DRAFT
* **And** material is removed from pinned list (automatically unpins)
* **And** `pinned` is set to `false`
* **And** `pinnedAt` is set to `null`

---

### User Story 3 - Pin Material (Priority: P2)

As a Lead of a group, I want to pin a published material so that it appears at the top of the material list for group members.

**Why this priority**: Pinning highlights important announcements for the group.

**Independent Test**: Any Lead pins a PUBLISHED material. Verify pinned=true, pinnedAt is set, and material appears at top of listing.

**Acceptance Scenarios**:

1. **Scenario**: Successful pin by author

* **Given** a user is authenticated as the author of a PUBLISHED material
* **When** they submit a pin request
* **Then** the system returns 200 OK with updated material
* **And** `pinned` is set to `true`
* **And** `pinnedAt` is set to current timestamp
* **And** `updatedAt` is updated
* **And** material appears at top of group material list

2. **Scenario**: Successful pin by another Lead (not author)

* **Given** a Lead (not the author) is authenticated
* **And** the material is PUBLISHED
* **When** they submit a pin request
* **Then** the system pins the material successfully
* **And** `pinned` is set to `true`
* **And** `pinnedAt` is set to current timestamp

3. **Scenario**: Attempt to pin DRAFT material

* **Given** a material is in DRAFT status
* **When** any Lead attempts to pin it
* **Then** the system rejects with 400 Bad Request (`CANNOT_PIN_DRAFT`)
* **And** message explains material must be published first

4. **Scenario**: Pin already pinned material (idempotent)

* **Given** a material is already pinned
* **When** a Lead pins it again
* **Then** the system returns 200 OK
* **And** `pinned` remains `true`
* **And** `pinnedAt` remains unchanged (original pin time)

5. **Scenario**: Pin by Admin

* **Given** an Admin is authenticated
* **When** they pin any PUBLISHED material
* **Then** the system pins the material successfully

6. **Scenario**: Multiple pinned materials

* **Given** multiple materials are already pinned in the group
* **When** a Lead pins another material
* **Then** the new pinned material appears first (most recent pinnedAt)
* **And** other pinned materials follow by pinnedAt descending

7. **Scenario**: Attempt to pin by Member (not Lead)

* **Given** a Member (not Lead) is authenticated
* **When** they attempt to pin a material
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

---

### User Story 4 - Unpin Material (Priority: P2)

As a Lead of a group, I want to unpin a material so that it returns to the normal chronological order.

**Why this priority**: Unpinning allows removing outdated highlights.

**Independent Test**: Any Lead unpins a pinned material. Verify pinned=false, pinnedAt is null, and material returns to normal ordering.

**Acceptance Scenarios**:

1. **Scenario**: Successful unpin by author

* **Given** a user is authenticated as the author of a pinned material
* **When** they submit an unpin request
* **Then** the system returns 200 OK with updated material
* **And** `pinned` is set to `false`
* **And** `pinnedAt` is set to `null`
* **And** `updatedAt` is updated
* **And** material returns to chronological position

2. **Scenario**: Successful unpin by another Lead (not author)

* **Given** a Lead (not the author) is authenticated
* **And** the material is pinned
* **When** they submit an unpin request
* **Then** the system unpins the material successfully
* **And** `pinned` is set to `false`
* **And** `pinnedAt` is set to `null`

3. **Scenario**: Unpin already unpinned material (idempotent)

* **Given** a material is not pinned
* **When** a Lead unpins it
* **Then** the system returns 200 OK
* **And** `pinned` remains `false`

4. **Scenario**: Attempt to unpin by Member (not Lead)

* **Given** a Member (not Lead) is authenticated
* **When** they attempt to unpin a material
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

---

### Edge Cases

- Publish material with empty content (allowed).
- Pin multiple materials in rapid succession.
- Concurrent publish/pin requests.
- Unpublish material viewed by members (they lose access immediately).
- Lead pins material then author unpublishes (auto-unpin).

---

## API Contract

### Publish Material

#### Endpoint

```
POST /groups/{groupId}/materials/{materialId}/publish
```

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group the material belongs to |
| materialId | UUID | Yes | The material to publish |

#### Success Response (200 OK)

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Course Introduction",
  "content": "# Introduction\n\nWelcome...",
  "tags": ["announcement"],
  "status": "PUBLISHED",
  "pinned": false,
  "pinnedAt": null,
  "groupId": "g1h2i3j4-k5l6-7890-mnop-qr1234567890",
  "authorId": "u1v2w3x4-y5z6-7890-abcd-ef1234567890",
  "createdAt": "2026-01-24T10:30:00Z",
  "updatedAt": "2026-01-24T15:00:00Z",
  "publishedAt": "2026-01-24T15:00:00Z"
}
```

---

### Unpublish Material

#### Endpoint

```
POST /groups/{groupId}/materials/{materialId}/unpublish
```

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group the material belongs to |
| materialId | UUID | Yes | The material to unpublish |

#### Success Response (200 OK)

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Course Introduction",
  "content": "# Introduction\n\nWelcome...",
  "tags": ["announcement"],
  "status": "DRAFT",
  "pinned": false,
  "pinnedAt": null,
  "groupId": "g1h2i3j4-k5l6-7890-mnop-qr1234567890",
  "authorId": "u1v2w3x4-y5z6-7890-abcd-ef1234567890",
  "createdAt": "2026-01-24T10:30:00Z",
  "updatedAt": "2026-01-24T16:00:00Z",
  "publishedAt": "2026-01-24T15:00:00Z"
}
```

---

### Pin Material

#### Endpoint

```
POST /groups/{groupId}/materials/{materialId}/pin
```

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group the material belongs to |
| materialId | UUID | Yes | The material to pin |

#### Success Response (200 OK)

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Important Announcement",
  "content": "# Important\n\nPlease read...",
  "tags": ["announcement", "important"],
  "status": "PUBLISHED",
  "pinned": true,
  "pinnedAt": "2026-01-24T16:30:00Z",
  "groupId": "g1h2i3j4-k5l6-7890-mnop-qr1234567890",
  "authorId": "u1v2w3x4-y5z6-7890-abcd-ef1234567890",
  "createdAt": "2026-01-24T10:30:00Z",
  "updatedAt": "2026-01-24T16:30:00Z",
  "publishedAt": "2026-01-24T15:00:00Z"
}
```

#### Error Response - Cannot Pin Draft

```json
{
  "error": "CANNOT_PIN_DRAFT",
  "message": "Cannot pin a draft material. Publish it first.",
  "currentStatus": "DRAFT"
}
```

---

### Unpin Material

#### Endpoint

```
POST /groups/{groupId}/materials/{materialId}/unpin
```

#### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |

#### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group the material belongs to |
| materialId | UUID | Yes | The material to unpin |

#### Success Response (200 OK)

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Important Announcement",
  "content": "# Important\n\nPlease read...",
  "tags": ["announcement", "important"],
  "status": "PUBLISHED",
  "pinned": false,
  "pinnedAt": null,
  "groupId": "g1h2i3j4-k5l6-7890-mnop-qr1234567890",
  "authorId": "u1v2w3x4-y5z6-7890-abcd-ef1234567890",
  "createdAt": "2026-01-24T10:30:00Z",
  "updatedAt": "2026-01-24T17:00:00Z",
  "publishedAt": "2026-01-24T15:00:00Z"
}
```

---

### Common Error Responses

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 403 Forbidden - Not Author (for publish/unpublish only)

```json
{
  "error": "NOT_MATERIAL_AUTHOR",
  "message": "Only the material author can publish/unpublish this material",
  "authorId": "original-author-id"
}
```

#### 403 Forbidden - Insufficient Permissions (for pin/unpin by non-Lead)

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only group Leads can pin/unpin materials",
  "requiredRole": "LEAD",
  "currentRole": "MEMBER"
}
```

#### 404 Not Found - Material Not Found

```json
{
  "error": "MATERIAL_NOT_FOUND",
  "message": "Material not found",
  "materialId": "nonexistent-material-id"
}
```

#### 404 Not Found - Group Not Found

```json
{
  "error": "GROUP_NOT_FOUND",
  "message": "Group not found",
  "groupId": "nonexistent-group-id"
}
```

---

## Functional Requirements

### Permission Validation (Publish/Unpublish)

- **FR-001**: The system MUST verify the user is authenticated.
- **FR-002**: The system MUST verify the group exists.
- **FR-003**: The system MUST verify the material exists and belongs to the group.
- **FR-004**: For publish/unpublish, the system MUST verify the user is the author OR has Admin role.
- **FR-004.1**: Other Leads CANNOT publish/unpublish materials they didn't create.

### Permission Validation (Pin/Unpin)

- **FR-005**: For pin/unpin, the system MUST verify the user is a Lead of the group OR has Admin role.
- **FR-005.1**: Any Lead of the group can pin/unpin any PUBLISHED material (not just their own).
- **FR-005.2**: Members (non-Leads) CANNOT pin/unpin materials.

### Publish Behavior

- **FR-006**: The system MUST change `status` from `DRAFT` to `PUBLISHED` on publish.
- **FR-007**: The system MUST set `publishedAt` to current timestamp on first publish only.
- **FR-008**: The system MUST preserve `publishedAt` if material was previously published.
- **FR-009**: The system MUST update `updatedAt` on publish.
- **FR-010**: Publishing an already published material MUST be idempotent.

### Unpublish Behavior

- **FR-011**: The system MUST change `status` from `PUBLISHED` to `DRAFT` on unpublish.
- **FR-012**: The system MUST automatically unpin the material on unpublish.
- **FR-013**: The system MUST set `pinned` to `false` and `pinnedAt` to `null` on unpublish.
- **FR-014**: The system MUST preserve `publishedAt` (historical record).
- **FR-015**: The system MUST update `updatedAt` on unpublish.
- **FR-016**: Unpublishing an already draft material MUST be idempotent.

### Pin Behavior

- **FR-017**: The system MUST only allow pinning PUBLISHED materials.
- **FR-018**: The system MUST reject pinning DRAFT materials with `CANNOT_PIN_DRAFT` error.
- **FR-019**: The system MUST set `pinned` to `true` on pin.
- **FR-020**: The system MUST set `pinnedAt` to current timestamp on first pin only.
- **FR-021**: The system MUST preserve `pinnedAt` if material was previously pinned.
- **FR-022**: The system MUST update `updatedAt` on pin.
- **FR-023**: Pinning an already pinned material MUST be idempotent.

### Unpin Behavior

- **FR-024**: The system MUST set `pinned` to `false` on unpin.
- **FR-025**: The system MUST set `pinnedAt` to `null` on unpin.
- **FR-026**: The system MUST update `updatedAt` on unpin.
- **FR-027**: Unpinning an already unpinned material MUST be idempotent.

### Response

- **FR-028**: All operations MUST return 200 OK with complete updated material.
- **FR-029**: All operations MUST return appropriate error codes for failures.

---

## Non-Functional Requirements

- **NFR-001**: All visibility operations MUST complete within 300ms under normal load.
- **NFR-002**: Operations MUST be atomic.
- **NFR-003**: The system MUST handle concurrent visibility change requests gracefully.

---

## Data Model

### Key Entities

- **Material**: The content being managed.
  Key attributes for visibility:
  - `status` (enum: `DRAFT` | `PUBLISHED`)
  - `pinned` (boolean)
  - `pinnedAt` (timestamp, nullable)
  - `publishedAt` (timestamp, nullable)
  - `updatedAt` (timestamp)

### State Transitions

```
DRAFT ─────(publish)─────> PUBLISHED
  ↑                           │
  └───────(unpublish)─────────┘
                              │
                          (can pin - any Lead)
                              │
                              v
                      PUBLISHED + PINNED
                              │
                          (can unpin - any Lead)
                              │
                              v
                         PUBLISHED
```

### Sorting Logic

Material listing order:
1. **Pinned materials first**: Sorted by `pinnedAt` descending (most recently pinned first)
2. **Non-pinned materials**: Sorted by `publishedAt` descending, then `createdAt` descending

### Permission Summary

| Action | Author | Other Leads | Admin | Members |
|--------|--------|-------------|-------|---------|
| Publish | ✅ | ❌ | ✅ | ❌ |
| Unpublish | ✅ | ❌ | ✅ | ❌ |
| Pin | ✅ | ✅ | ✅ | ❌ |
| Unpin | ✅ | ✅ | ✅ | ❌ |

---

## Security Considerations

- **SEC-001**: Only authenticated users can change material visibility.
- **SEC-002**: Only the author or Admin can publish/unpublish a material.
- **SEC-003**: Any Lead of the group can pin/unpin published materials.
- **SEC-004**: Members cannot pin/unpin materials.
- **SEC-005**: DRAFT materials are only visible to author and Admin.

---

## Optional Notes

- **Notifications**: Consider notifying group members when a material is published (future feature).
- **Pin Limit**: Consider limiting number of pinned materials per group (e.g., max 5).
- **Scheduled Publishing**: Consider adding scheduled publish/unpublish times (future feature).
- **Pin Duration**: Consider auto-unpin after certain time period (future feature).
