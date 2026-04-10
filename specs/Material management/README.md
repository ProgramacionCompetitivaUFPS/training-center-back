# Material Management - Business Logic & Design

**Created**: 2026-01-24

This document centralizes the complete business logic of the Material Management system and design considerations being applied across all related specs.

---

## 🔹 General Concept

* **`Material`** is a post/announcement that belongs to a Group.
* Materials allow Leads to share announcements, resources, and information with group members.
* Materials support **Markdown content** with embedded images, links, and videos.
* Materials have a **lifecycle** (DRAFT → PUBLISHED) for content preparation.
* Materials can be **pinned** to appear at the top of the list.
* **No file uploads**: All media (images, PDFs, videos) are embedded via URLs directly in the Markdown content.

---

## 🔹 Material States

### Status (Publication State)

| Status | Description | Visible To |
|--------|-------------|------------|
| `DRAFT` | Material is being prepared | Author only (and Admin) |
| `PUBLISHED` | Material is visible to group | All group members |

### State Transitions

```
DRAFT ──(publish)──> PUBLISHED ──(unpublish)──> DRAFT
```

### Publication Requirements

To publish a material, all of the following are required:
- `title` (always present, required at creation)
- `content` (Markdown content, can be empty but must exist)

---

## 🔹 Roles and Permissions

### Material Creation

| Role | Regular Group | Global Group |
|------|--------------|--------------|
| **Admin** | ✅ Any group (implicit Lead) | ✅ Yes |
| **Lead** | ✅ Only in their groups | ✅ Only if Lead of global group |
| **Member** | ❌ | ❌ |

### Material Management (Update, Delete, Publish, Pin)

| Action | Author | Other Leads | Admin |
|--------|--------|-------------|-------|
| View DRAFT | ✅ | ❌ | ✅ |
| View PUBLISHED | ✅ | ✅ | ✅ |
| Update | ✅ | ❌ | ✅ |
| Delete | ✅ | ❌ | ✅ |
| Publish/Unpublish | ✅ | ❌ | ✅ |
| Pin/Unpin | ✅ | ✅ | ✅ |

> **Note**: Only the author (or Admin) can edit/delete/publish materials. However, **any Lead** of the group can pin/unpin published materials to highlight important announcements.

---

## 🔹 Material Attributes

### Basic Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `id` | UUID | Auto | Unique identifier |
| `title` | string | Yes | Material title (max 200 chars) |
| `content` | string | Yes | Markdown content with embedded media (max 50000 chars) |
| `tags` | string[] | No | User-defined tags for categorization (lowercase + hyphens only) |
| `status` | enum | Auto | `DRAFT` or `PUBLISHED` (default: DRAFT) |
| `pinned` | boolean | Auto | Whether material is pinned (default: false) |
| `pinnedAt` | timestamp | Auto | When material was pinned (null if not pinned) |
| `group_id` | UUID | Yes | Group the material belongs to (immutable) |
| `author_id` | UUID | Yes | Material creator (immutable) |
| `createdAt` | timestamp | Auto | Creation timestamp (immutable) |
| `updatedAt` | timestamp | Auto | Last modification timestamp |
| `publishedAt` | timestamp | Auto | When first published (null if never published) |

---

## 🔹 Markdown Content with Embedded Media

Materials use **pure Markdown** with URLs embedded directly in the content. This allows natural flow of text and media like a blog.

### Supported Markdown Syntax

```markdown
# Main Title

This is introductory text explaining the topic.

## Section with Image

Here's a diagram showing the algorithm:

![Algorithm Diagram](https://example.com/diagram.png)

Now let's continue with the explanation...

## Video Tutorial

Watch this tutorial to understand better:

[![Video Tutorial](https://img.youtube.com/vi/VIDEO_ID/0.jpg)](https://youtube.com/watch?v=VIDEO_ID)

## Resources

- [📄 Download Syllabus (PDF)](https://example.com/syllabus.pdf)
- [🔗 External Documentation](https://docs.example.com)

## Code Example

```python
def hello():
    print("Hello, World!")
```
```

### Rendering Behavior

| Content Type | Markdown Syntax | Rendering |
|--------------|-----------------|-----------|
| **Images** | `![alt](url)` | Displayed inline |
| **Links** | `[text](url)` | Clickable link |
| **PDFs** | `[text](url.pdf)` | Link with PDF icon, opens in viewer |
| **YouTube** | `[![img](thumbnail)](youtube-url)` | Embedded video player |
| **Vimeo** | `[![img](thumbnail)](vimeo-url)` | Embedded video player |

### Auto-Detection

The renderer automatically detects and enhances:
- **YouTube/Vimeo URLs**: Converted to embedded players
- **PDF links**: Displayed with PDF viewer option
- **Image URLs**: Rendered as images with lazy loading

---

## 🔹 Tags (User-Defined)

### Tag Rules

* Tags are **created by users** when adding them to materials.
* Tags must follow these format rules:
  - Only **lowercase letters** (a-z), **numbers** (0-9), **hyphens** (-), and **underscores** (_)
  - No spaces or special characters
  - Length: 2-50 characters
  - Cannot start or end with hyphen or underscore
  - No consecutive hyphens or underscores

### Valid Tag Examples

```
✅ announcement
✅ algorithms
✅ data-structures
✅ data_structures
✅ week1
✅ week-1
✅ important_notice
✅ dp2024
```

### Invalid Tag Examples

```
❌ Announcement (uppercase)
❌ week 1 (space)
❌ -start (starts with hyphen)
❌ end_ (ends with underscore)
❌ too--many (consecutive hyphens)
❌ bad__tag (consecutive underscores)
```

### Tag Behavior

* Tags allow filtering materials within a group.
* Tags are **always optional**.
* New tags are automatically created when first used.
* Tags are stored per material (no global tag registry).
* Duplicate tags in same material are silently deduplicated.

---

## 🔹 Pinning

### Concept

* **Pinned materials** appear at the top of the material list, above non-pinned materials.
* Multiple materials can be pinned simultaneously.
* **Any Lead** of the group can pin/unpin published materials (not just the author).
* Pinned materials are sorted by `pinnedAt` timestamp (most recent pin first).
* Non-pinned materials are sorted by `publishedAt` or `createdAt` (most recent first).

### Sorting Order

```
1. Pinned materials (sorted by pinnedAt DESC)
2. Non-pinned materials (sorted by publishedAt DESC, then createdAt DESC)
```

### Pin Permissions

| User | Can Pin/Unpin |
|------|---------------|
| Author | ✅ Own published materials |
| Other Leads | ✅ Any published material in their group |
| Admin | ✅ Any published material |
| Members | ❌ |

> **Note**: Only PUBLISHED materials can be pinned. DRAFT materials cannot be pinned.

---

## 🔹 Visibility Rules

### For Group Members

| User Role | DRAFT Materials | PUBLISHED Materials | Pinned |
|-----------|-----------------|---------------------|--------|
| Author | ✅ Own only | ✅ All | ✅ All |
| Other Lead | ❌ | ✅ All | ✅ All |
| Member | ❌ | ✅ All | ✅ All |

### For Non-Members

| Group Visibility | Can See Materials |
|------------------|-------------------|
| VISIBLE | ✅ PUBLISHED only (read-only) |
| NOT_VISIBLE | ❌ Cannot see anything |

### Admin Override

* Admin can view ALL materials (DRAFT and PUBLISHED) in ANY group.
* Admin can manage ANY material regardless of authorship.

---

## 🔹 Content Deletion

### When a Material is Deleted

| Entity | Action |
|--------|--------|
| Material | Hard delete |
| Material Tags | Hard delete (tags stored with material) |

### When the Parent Group is Deleted

* All materials in the group are deleted (hard delete cascade)
* This is permanent and cannot be undone

---

## 🔹 Technical Considerations

### Validation

* Title: 1-200 characters, normalized NFKC
* Content: 0-50000 characters (Markdown)
* Tags: 2-50 characters each, lowercase letters + numbers + hyphens + underscores

### Security

* Content sanitization to prevent XSS in Markdown rendering
* URL validation within Markdown to prevent malicious links
* Rate limiting on material creation/updates

### Performance

* Index on `group_id` for material queries
* Index on `author_id` for author's materials
* Index on `status` for visibility filtering
* Index on `pinned` and `pinnedAt` for sorting
* Pagination for material listings

### Markdown Rendering

* Use sanitized Markdown renderer (e.g., marked.js with DOMPurify)
* Auto-detect video URLs and convert to embeds
* Lazy load images for performance
* Support syntax highlighting for code blocks

---

## 🔹 Related Specs

### Implemented Specs

1. **[Create material](Create%20material/spec.md)** - Material creation with Markdown content
2. **[Update material](Update%20material/spec.md)** - Modify material content and tags
3. **[Delete material](Delete%20material/spec.md)** - Remove material permanently
4. **[Change material visibility](Change%20material%20visibility/spec.md)** - Publish/unpublish and pin/unpin
5. **[View material](View%20material/spec.md)** - List materials in a group with filters, sorting, and pagination
6. **[Search materials](Search%20materials/spec.md)** - Full-text search within group materials with advanced filtering

### Implementation Dependencies

```
Create Material (base)
    ↓
Update Material (P1) ✅ ← (modify content, tags)
    ↓
Delete Material (P1) ✅ ← (permanent removal)
    ↓
Change Material Visibility (P1) ✅ ← (publish/unpublish, pin/unpin)
    ↓
View Materials (P1) ✅ ← (listing with filters, sorting, pagination)
    ↓
Search Materials (P2) ✅ ← (full-text search with advanced filtering)
```

---

## 🔹 Key Design Decisions

### Why pure Markdown with embedded URLs?

* **Natural flow**: Text and media can be interleaved like a blog
* **Industry standard**: Same approach as GitHub, Medium, Notion
* **Simplicity**: Single content field instead of separate URL table
* **Author control**: Full control over content layout and order

### Why no file uploads?

* **Simplicity**: Avoids storage management complexity
* **Cost**: No storage costs for platform
* **Flexibility**: Users can use any hosting service (Imgur, Google Drive, etc.)
* **Performance**: No bandwidth costs for serving files

### Why DRAFT/PUBLISHED status?

* **Quality control**: Leads can prepare content before publishing
* **Preview**: Authors can review before making visible
* **Flexibility**: Content can be unpublished for edits

### Why only author can edit but any Lead can pin?

* **Accountability**: Clear ownership of content for edits
* **Collaboration**: Allows other Leads to highlight important materials
* **Flexibility**: Team can manage pinned items without editing content

### Why user-defined tags instead of predefined?

* **Flexibility**: Groups can use tags relevant to their context
* **Simplicity**: No need to maintain global tag registry
* **User freedom**: Authors know best how to categorize their content

### Why lowercase + hyphens only for tags?

* **Consistency**: All tags look the same (no case variations)
* **URL-friendly**: Tags can be used in filter URLs
* **Simplicity**: Easy validation rules

---

*This document should be updated when new design decisions are made or additional specs are implemented.*
