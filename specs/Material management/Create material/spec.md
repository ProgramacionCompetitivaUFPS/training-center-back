# Feature Specification: Create Material

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Create a new Material (Priority: P1)

As a Lead of a group, I want to create a material (announcement/post) so that I can share information, resources, and announcements with group members.

**Why this priority**: Materials are the primary communication mechanism between Leads and group members for sharing announcements, resources, and educational content.

**Independent Test**: Authenticated request to the Material creation endpoint by a Lead of the group. Verify DB persisted Material record with correct author_id, group_id, status=DRAFT, and all provided fields. Verify non-Lead requests are rejected.

**Acceptance Scenarios**:

1. **Scenario**: Successful creation by a Lead with minimal data

* **Given** a Lead of a group is authenticated
* **When** they submit a CreateMaterial request with only `title`
* **Then** the system returns 201 Created with the created Material data
* **And** `status` is set to `DRAFT`
* **And** `author_id` is set to the creator's ID
* **And** `group_id` is set correctly
* **And** `content` defaults to empty string
* **And** `pinned` defaults to false
* **And** `createdAt` and `updatedAt` are set

2. **Scenario**: Successful creation with full data

* **Given** a Lead of a group is authenticated
* **When** they submit a CreateMaterial request with `title`, `content` (Markdown with embedded images/links), and `tags`
* **Then** the system returns 201 Created with all provided data
* **And** `status` is set to `DRAFT`
* **And** content is stored as-is (Markdown)
* **And** Tags are validated for format (lowercase + hyphens)

3. **Scenario**: Creation by Admin in any group

* **Given** an Admin is authenticated
* **When** they create a material in any group (even without explicit membership)
* **Then** the system creates the material successfully
* **And** `author_id` is set to the Admin's ID

4. **Scenario**: Attempt to create by a Member (not Lead)

* **Given** a Member of a group (not Lead) is authenticated
* **When** they attempt to create a material in that group
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

5. **Scenario**: Attempt to create by non-member

* **Given** a user who is not a member of the group is authenticated
* **When** they attempt to create a material in that group
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

6. **Scenario**: Attempt to create with invalid tag format

* **Given** a Lead is authenticated
* **When** they submit a CreateMaterial request with an invalid tag (uppercase, numbers, etc.)
* **Then** the system rejects with 400 Bad Request (`INVALID_TAG_FORMAT`)
* **And** the response includes the list of invalid tags and format rules

---

### User Story 2 - Create Material with Markdown Content (Priority: P1)

As a Lead, I want to write rich content with embedded images, links, and videos using Markdown so that I can create engaging blog-style posts.

**Why this priority**: Markdown with embedded media is essential for creating rich, well-formatted content.

**Independent Test**: Create material with Markdown content including images, links, and video embeds. Verify content is stored correctly and can be rendered.

**Acceptance Scenarios**:

1. **Scenario**: Create material with embedded images

* **Given** a Lead is authenticated
* **When** they create a material with Markdown containing `![alt](image-url)`
* **Then** the system stores the content as-is
* **And** images will be rendered inline when displayed

2. **Scenario**: Create material with embedded video links

* **Given** a Lead is authenticated
* **When** they create a material with YouTube/Vimeo links in Markdown
* **Then** the system stores the content
* **And** video links will be converted to embedded players when rendered

3. **Scenario**: Create material with mixed content (blog-style)

* **Given** a Lead is authenticated
* **When** they create a material with text → image → text → video → text
* **Then** all content is stored in order
* **And** the layout is preserved when rendered

4. **Scenario**: Create material with code blocks

* **Given** a Lead is authenticated
* **When** they create a material with fenced code blocks (```)
* **Then** code blocks are stored and will render with syntax highlighting

---

### User Story 3 - Create Material with Tags (Priority: P2)

As a Lead, I want to add custom tags to my material so that group members can filter and find relevant content.

**Why this priority**: Tags help organize content and improve discoverability.

**Independent Test**: Create material with various tags. Verify tags are validated for format and stored correctly.

**Acceptance Scenarios**:

1. **Scenario**: Create material with valid tags

* **Given** a Lead is authenticated
* **When** they create a material with tags `["announcement", "week1", "data_structures"]`
* **Then** the system stores the tags with the material
* **And** the material can be filtered by these tags

2. **Scenario**: Attempt to create with uppercase tag

* **Given** a Lead is authenticated
* **When** they submit a CreateMaterial request with tag `"Announcement"`
* **Then** the system rejects with 400 Bad Request (`INVALID_TAG_FORMAT`)
* **And** message explains tags must be lowercase

3. **Scenario**: Attempt to create with tag containing spaces

* **Given** a Lead is authenticated
* **When** they submit a CreateMaterial request with tag `"week one"`
* **Then** the system rejects with 400 Bad Request (`INVALID_TAG_FORMAT`)
* **And** message explains tags can only contain lowercase letters, numbers, hyphens, and underscores

4. **Scenario**: Create material with tags containing numbers

* **Given** a Lead is authenticated
* **When** they create a material with tags `["week1", "dp2024", "round-2"]`
* **Then** the system stores the tags successfully

5. **Scenario**: Create with duplicate tags (deduplicated)

* **Given** a Lead is authenticated
* **When** they create a material with tags `["important", "important", "news"]`
* **Then** duplicates are silently removed
* **And** material is stored with tags `["important", "news"]`

---

### User Story 4 - Create Material in Global Group (Priority: P2)

As an Admin or Lead of the global group, I want to create materials in the global group so that I can share platform-wide announcements with all users.

**Why this priority**: Global group materials serve as platform announcements visible to all users.

**Independent Test**: Admin or Lead of global group creates material. Verify material is created and visible to all platform users.

**Acceptance Scenarios**:

1. **Scenario**: Admin creates material in global group

* **Given** an Admin is authenticated
* **When** they create a material in the global group
* **Then** the system creates the material successfully
* **And** when published, all platform users can see it

2. **Scenario**: Lead of global group creates material

* **Given** a Coach who is Lead of the global group is authenticated
* **When** they create a material in the global group
* **Then** the system creates the material successfully

3. **Scenario**: Non-Lead attempts to create in global group

* **Given** a regular user (not Lead of global group) is authenticated
* **When** they attempt to create a material in the global group
* **Then** the system rejects with 403 Forbidden (`INSUFFICIENT_PERMISSIONS`)

---

### Edge Cases

- Title with Unicode characters requiring normalization (NFKC).
- Content with very long Markdown (up to 50000 chars).
- Content with many embedded images (no limit, but content size limited).
- Markdown with potentially malicious HTML (sanitized on render).
- Empty content with only title (valid).
- Material creation in non-existent group.
- Concurrent creation requests by same author.
- Tags with edge cases: exactly 2 chars, exactly 50 chars.

---

## API Contract

### Endpoint

```
POST /groups/{groupId}/materials
```

### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |
| Content-Type | Yes | application/json |

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group to create material in |

### Request Body

```json
{
  "title": "Welcome to the Course",
  "content": "# Introduction\n\nWelcome to our programming course!\n\n![Course Overview](https://example.com/course-overview.png)\n\nIn this course, we'll cover:\n\n- Algorithm basics\n- Data structures\n- Problem solving\n\n## Video Introduction\n\n[![Watch the intro](https://img.youtube.com/vi/abc123/0.jpg)](https://youtube.com/watch?v=abc123)\n\n## Resources\n\n- [📄 Syllabus (PDF)](https://example.com/syllabus.pdf)\n- [🔗 Documentation](https://docs.example.com)\n\n**Important**: Please read all materials carefully.",
  "tags": ["announcement", "beginner", "welcome"]
}
```

### Request Body Fields

| Field | Type | Required | Constraints | Description |
|-------|------|----------|-------------|-------------|
| title | string | Yes | 1-200 chars | Material title |
| content | string | No | 0-50000 chars | Markdown content with embedded media (default: "") |
| tags | string[] | No | 2-50 chars each, lowercase + numbers + hyphens + underscores | Tags for categorization |

### Tag Format Rules

| Rule | Description |
|------|-------------|
| Characters | Lowercase letters (a-z), numbers (0-9), hyphens (-), underscores (_) |
| Length | 2-50 characters |
| Start/End | Cannot start or end with hyphen or underscore |
| Consecutive | No consecutive hyphens (--) or underscores (__) |

### Success Response (201 Created)

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "title": "Welcome to the Course",
  "content": "# Introduction\n\nWelcome to our programming course!...",
  "tags": ["announcement", "beginner", "welcome"],
  "status": "DRAFT",
  "pinned": false,
  "pinnedAt": null,
  "groupId": "g1h2i3j4-k5l6-7890-mnop-qr1234567890",
  "authorId": "u1v2w3x4-y5z6-7890-abcd-ef1234567890",
  "createdAt": "2026-01-24T10:30:00Z",
  "updatedAt": "2026-01-24T10:30:00Z",
  "publishedAt": null
}
```

### Error Responses

#### 400 Bad Request - Validation Error

```json
{
  "error": "VALIDATION_ERROR",
  "message": "Request validation failed",
  "details": [
    {
      "field": "title",
      "code": "REQUIRED",
      "message": "Title is required"
    }
  ]
}
```

#### 400 Bad Request - Title Too Long

```json
{
  "error": "TITLE_TOO_LONG",
  "message": "Title must be at most 200 characters",
  "maxLength": 200,
  "providedLength": 250
}
```

#### 400 Bad Request - Content Too Long

```json
{
  "error": "CONTENT_TOO_LONG",
  "message": "Content must be at most 50000 characters",
  "maxLength": 50000,
  "providedLength": 55000
}
```

#### 400 Bad Request - Invalid Tag Format

```json
{
  "error": "INVALID_TAG_FORMAT",
  "message": "Tags must contain only lowercase letters, numbers, hyphens, and underscores",
  "invalidTags": ["Week1", "Important!", "my tag"],
  "rules": {
    "characters": "lowercase letters (a-z), numbers (0-9), hyphens (-), underscores (_)",
    "length": "2-50 characters",
    "format": "cannot start/end with hyphen or underscore, no consecutive hyphens or underscores"
  }
}
```

#### 400 Bad Request - Tag Too Short

```json
{
  "error": "TAG_TOO_SHORT",
  "message": "Tags must be at least 2 characters",
  "invalidTags": ["a"],
  "minLength": 2
}
```

#### 400 Bad Request - Tag Too Long

```json
{
  "error": "TAG_TOO_LONG",
  "message": "Tags must be at most 50 characters",
  "invalidTags": ["this-is-a-very-long-tag-that-exceeds-the-maximum-allowed-length"],
  "maxLength": 50
}
```

#### 401 Unauthorized

```json
{
  "error": "UNAUTHORIZED",
  "message": "Authentication required"
}
```

#### 403 Forbidden - Insufficient Permissions

```json
{
  "error": "INSUFFICIENT_PERMISSIONS",
  "message": "Only group Leads can create materials",
  "requiredRole": "LEAD",
  "currentRole": "MEMBER"
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

- **FR-001**: The system MUST verify the user is authenticated before processing the request.
- **FR-002**: The system MUST verify the group exists before creating material.
- **FR-003**: The system MUST verify the user is a Lead of the group OR has Admin role.
- **FR-003.1**: Admin users have implicit Lead permissions on all groups.

### Input Validation

- **FR-004**: The system MUST require `title` field.
- **FR-005**: The system MUST validate `title` length is between 1 and 200 characters.
- **FR-006**: The system MUST normalize `title` using Unicode NFKC normalization.
- **FR-007**: The system MUST validate `content` length does not exceed 50000 characters.
- **FR-008**: The system MUST default `content` to empty string if not provided.

### Tag Validation

- **FR-009**: The system MUST validate all tags contain only lowercase letters (a-z), numbers (0-9), hyphens (-), and underscores (_).
- **FR-010**: The system MUST validate tag length is between 2 and 50 characters.
- **FR-011**: The system MUST reject tags that start or end with a hyphen or underscore.
- **FR-012**: The system MUST reject tags with consecutive hyphens (--) or underscores (__).
- **FR-013**: The system MUST silently deduplicate tags if duplicates are provided.
- **FR-014**: The system MUST return descriptive error with invalid tags listed.

### Material Creation

- **FR-015**: The system MUST generate a unique UUID for the material.
- **FR-016**: The system MUST set `author_id` to the authenticated user's ID.
- **FR-017**: The system MUST set `group_id` to the target group's ID.
- **FR-018**: The system MUST set `status` to `DRAFT` on creation.
- **FR-019**: The system MUST set `pinned` to `false` on creation.
- **FR-020**: The system MUST set `createdAt` and `updatedAt` to current timestamp.
- **FR-021**: The system MUST set `publishedAt` to `null` on creation.
- **FR-022**: The system MUST set `pinnedAt` to `null` on creation.

### Response

- **FR-023**: The system MUST return 201 Created with the complete material data on success.
- **FR-024**: The system MUST return appropriate error codes for validation failures.

---

## Non-Functional Requirements

- **NFR-001**: Material creation MUST complete within 500ms under normal load.
- **NFR-002**: The system MUST handle concurrent material creation requests gracefully.
- **NFR-003**: The system MUST sanitize Markdown content on rendering to prevent XSS attacks.
- **NFR-004**: The system SHOULD validate URLs within Markdown content for basic format.

---

## Data Model

### Key Entities

- **Material**: The content/announcement being created.
  Key attributes:
  - `id` (UUID, primary key, auto-generated)
  - `title` (string, 1-200 chars, normalized NFKC)
  - `content` (string, 0-50000 chars, Markdown format with embedded media)
  - `tags` (string[], each 2-50 chars, lowercase + numbers + hyphens + underscores)
  - `status` (enum: `DRAFT` | `PUBLISHED`, default: DRAFT)
  - `pinned` (boolean, default: false)
  - `pinnedAt` (timestamp, nullable)
  - `group_id` (UUID, FK to Group, immutable)
  - `author_id` (UUID, FK to User, immutable)
  - `createdAt` (timestamp, immutable)
  - `updatedAt` (timestamp)
  - `publishedAt` (timestamp, nullable)

- **Group**: The group that owns the material.
  Key attributes:
  - `id` (UUID)
  - `name` (string)
  - `visibility` (enum: `VISIBLE` | `NOT_VISIBLE`)

- **User**: The author of the material.
  Key attributes:
  - `id` (UUID)
  - `role` (enum: `ADMIN` | `COACH` | `CONTESTANT`)

- **GroupMember**: Membership relationship.
  Key attributes:
  - `user_id` (UUID, FK to User)
  - `group_id` (UUID, FK to Group)
  - `memberRole` (enum: `LEAD` | `MEMBER`)

### Entity Relationships

```
Material >------ Group (material belongs to one group)
Material >------ User (material has one author)
User >-----< Group (many-to-many through GroupMember)
```

---

## Security Considerations

- **SEC-001**: Only authenticated users can create materials.
- **SEC-002**: Only Leads or Admin can create materials in a group.
- **SEC-003**: Markdown content must be sanitized on rendering to prevent XSS.
- **SEC-004**: URLs in Markdown should be validated for basic format.
- **SEC-005**: Rate limiting should be applied to prevent spam material creation.

---

## Optional Notes

- **Markdown Rendering**: Use sanitized renderer (marked.js + DOMPurify).
- **Video Detection**: Auto-detect YouTube/Vimeo URLs and convert to embeds.
- **PDF Links**: Detect `.pdf` links and offer embedded viewer option.
- **Image Lazy Loading**: Implement lazy loading for images in content.
- **Draft Auto-save**: Consider implementing auto-save for drafts to prevent data loss.
- **Notification**: Consider notifying group members when a material is published (future feature).
