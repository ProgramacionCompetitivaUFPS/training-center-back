# Feature Specification: Update Material

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Update Material Content (Priority: P1)

As the author of a material, I want to update its content so that I can correct mistakes, add new information, or improve the presentation.

**Why this priority**: Materials need to be editable to keep content current and accurate.

**Independent Test**: Authenticated request by the material author to update content. Verify DB changes are persisted, `updatedAt` is updated, and other fields remain unchanged. Verify non-authors cannot update.

**Acceptance Scenarios**:

1. **Scenario**: Successful update by author

* **Given** a user is authenticated as the author of a material
* **When** they submit an UpdateMaterial request with new `title` and `content`
* **Then** the system returns 200 OK with the updated material
* **And** `updatedAt` is set to current timestamp
* **And** other fields remain unchanged (status, pinned, authorId, groupId)

2. **Scenario**: Partial update (only title)

* **Given** a user is authenticated as the author of a material
* **When** they submit an UpdateMaterial request with only `title`
* **Then** the system updates only `title`
* **And** `content` and `tags` remain unchanged
* **And** `updatedAt` is updated

3. **Scenario**: Update Markdown content with embedded media

* **Given** a user is authenticated as the author of a material
* **When** they submit an UpdateMaterial request with new Markdown content containing images and videos
* **Then** the system stores the updated content as-is
* **And** embedded media will render correctly

4. **Scenario**: Update tags

* **Given** a user is authenticated as the author of a material
* **When** they submit an UpdateMaterial request with new `tags`
* **Then** the system replaces all existing tags with the new ones
* **And** tags are validated for format (lowercase + hyphens)

5. **Scenario**: Attempt to update by non-author Lead

* **Given** a Lead of the group (but not the author) is authenticated
* **When** they attempt to update another Lead's material
* **Then** the system rejects with 403 Forbidden (`NOT_MATERIAL_AUTHOR`)

6. **Scenario**: Update by Admin (any material)

* **Given** an Admin is authenticated
* **When** they update any material (regardless of authorship)
* **Then** the system updates the material successfully

7. **Scenario**: Update DRAFT material

* **Given** a material is in DRAFT status
* **When** the author updates it
* **Then** the update succeeds
* **And** `status` remains DRAFT

8. **Scenario**: Update PUBLISHED material

* **Given** a material is in PUBLISHED status
* **When** the author updates it
* **Then** the update succeeds
* **And** `status` remains PUBLISHED (update doesn't unpublish)

---

### User Story 2 - Clear Material Content (Priority: P2)

As the author of a material, I want to clear certain fields so that I can remove unwanted content.

**Why this priority**: Authors may need to remove tags without deleting the entire material.

**Independent Test**: Update material with empty arrays for tags. Verify fields are cleared.

**Acceptance Scenarios**:

1. **Scenario**: Clear all tags

* **Given** a material has multiple tags
* **When** the author updates with `tags: []`
* **Then** all tags are removed from the material

2. **Scenario**: Set content to empty

* **Given** a material has content
* **When** the author updates with `content: ""`
* **Then** content is cleared (set to empty string)
* **And** the material remains valid

---

### Edge Cases

- Updating with same values (idempotent operation).
- Concurrent update requests by same author.
- Update with Unicode characters requiring normalization.
- Update immediately after publish (verify status preserved).
- Very long content update (50000 chars).
- Update tags with invalid format.

---

## API Contract

### Endpoint

```
PUT /groups/{groupId}/materials/{materialId}
```

### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |
| Content-Type | Yes | application/json |

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group the material belongs to |
| materialId | UUID | Yes | The material to update |

### Request Body

```json
{
  "title": "Updated Course Introduction",
  "content": "# Updated Introduction\n\nThis is the updated content with new information...\n\n![New Diagram](https://example.com/new-diagram.png)\n\nMore text here...",
  "tags": ["announcement", "updated", "important"]
}
```

### Request Body Fields

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| title | string | No | 1-200 chars | New title (if updating) |
| content | string | No | 0-50000 chars | New Markdown content |
| tags | string[] | No | 2-50 chars each, lowercase + numbers + hyphens + underscores | New tags (replaces existing) |

> **Note**: All fields are optional. Only provided fields are updated. To clear a field, provide empty value (`""` for content, `[]` for tags).

### Success Response (200 OK)

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Updated Course Introduction",
  "content": "# Updated Introduction\n\nThis is the updated content...",
  "tags": ["announcement", "updated", "important"],
  "status": "DRAFT",
  "pinned": false,
  "pinnedAt": null,
  "groupId": "g1h2i3j4-k5l6-7890-mnop-qr1234567890",
  "authorId": "u1v2w3x4-y5z6-7890-abcd-ef1234567890",
  "createdAt": "2026-01-24T10:30:00Z",
  "updatedAt": "2026-01-24T14:45:00Z",
  "publishedAt": null
}
```

### Error Responses

#### 400 Bad Request - Title Too Long

```json
{
  "error": "TITLE_TOO_LONG",
  "message": "Title must be at most 200 characters",
  "maxLength": 200,
  "providedLength": 250
}
```

#### 400 Bad Request - Empty Title

```json
{
  "error": "TITLE_REQUIRED",
  "message": "Title cannot be empty if provided"
}
```

#### 400 Bad Request - Invalid Tag Format

```json
{
  "error": "INVALID_TAG_FORMAT",
  "message": "Tags must contain only lowercase letters, numbers, hyphens, and underscores",
  "invalidTags": ["Invalid-Tag", "tag with space"],
  "rules": {
    "characters": "lowercase letters (a-z), numbers (0-9), hyphens (-), underscores (_)",
    "length": "2-50 characters",
    "format": "cannot start/end with hyphen or underscore, no consecutive hyphens or underscores"
  }
}
```

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 403 Forbidden - Not Author

```json
{
  "error": "NOT_MATERIAL_AUTHOR",
  "message": "Only the material author can update this material",
  "authorId": "original-author-id"
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

### Permission Validation

- **FR-001**: The system MUST verify the user is authenticated.
- **FR-002**: The system MUST verify the group exists.
- **FR-003**: The system MUST verify the material exists and belongs to the group.
- **FR-004**: The system MUST verify the user is the author of the material OR has Admin role.
- **FR-004.1**: Other Leads of the group CANNOT update materials they didn't create.

### Input Validation

- **FR-005**: The system MUST validate `title` (if provided) is 1-200 characters.
- **FR-006**: The system MUST normalize `title` using Unicode NFKC normalization.
- **FR-007**: The system MUST validate `content` (if provided) does not exceed 50000 characters.
- **FR-008**: The system MUST allow empty string for `content` to clear it.

### Tag Validation

- **FR-009**: The system MUST validate all tags (if provided) contain only lowercase letters, numbers, hyphens, and underscores.
- **FR-010**: The system MUST validate tag length is between 2 and 50 characters.
- **FR-011**: The system MUST reject tags that start or end with a hyphen or underscore.
- **FR-012**: The system MUST reject tags with consecutive hyphens or underscores.
- **FR-013**: The system MUST replace all existing tags when `tags` is provided.
- **FR-014**: The system MUST allow empty array to clear all tags.
- **FR-015**: The system MUST silently deduplicate tags.

### Update Behavior

- **FR-016**: The system MUST only update provided fields (partial update).
- **FR-017**: The system MUST NOT change `status`, `pinned`, `authorId`, `groupId`, `createdAt`.
- **FR-018**: The system MUST update `updatedAt` to current timestamp.
- **FR-019**: The system MUST preserve `publishedAt` value.

### Response

- **FR-020**: The system MUST return 200 OK with complete updated material.
- **FR-021**: The system MUST return appropriate error codes for validation failures.

---

## Non-Functional Requirements

- **NFR-001**: Material update MUST complete within 500ms under normal load.
- **NFR-002**: The system MUST handle concurrent update requests gracefully.
- **NFR-003**: The system MUST use optimistic locking to prevent lost updates.

---

## Data Model

### Key Entities

- **Material**: The content being updated.
  Key attributes:
  - `id` (UUID, primary key)
  - `title` (string, 1-200 chars, normalized NFKC)
  - `content` (string, 0-50000 chars, Markdown with embedded media)
  - `tags` (string[], each 2-50 chars, lowercase + numbers + hyphens + underscores)
  - `status` (enum: `DRAFT` | `PUBLISHED`, immutable via this endpoint)
  - `pinned` (boolean, immutable via this endpoint)
  - `group_id` (UUID, immutable)
  - `author_id` (UUID, immutable)
  - `createdAt` (timestamp, immutable)
  - `updatedAt` (timestamp, updated on each modification)
  - `publishedAt` (timestamp, immutable via this endpoint)

---

## Security Considerations

- **SEC-001**: Only authenticated users can update materials.
- **SEC-002**: Only the author or Admin can update a material.
- **SEC-003**: Other Leads cannot update materials they didn't create.
- **SEC-004**: Content sanitization on rendering to prevent XSS.

---

## Optional Notes

- **Optimistic Locking**: Consider using version field or ETag for concurrent update handling.
- **Update History**: Consider tracking update history for audit purposes (future feature).
- **Diff View**: Consider showing diff between versions (future feature).
