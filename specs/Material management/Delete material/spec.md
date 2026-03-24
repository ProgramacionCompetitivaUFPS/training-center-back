# Feature Specification: Delete Material

**Created**: 2026-01-24

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Delete Material (Priority: P1)

As the author of a material, I want to delete it permanently so that I can remove outdated or incorrect content from the group.

**Why this priority**: Authors need the ability to remove content they no longer want visible.

**Independent Test**: Authenticated request by the material author to delete. Verify material and associated data (URLs, tags) are hard deleted from DB. Verify non-authors cannot delete.

**Acceptance Scenarios**:

1. **Scenario**: Successful deletion by author

* **Given** a user is authenticated as the author of a material
* **When** they submit a DeleteMaterial request
* **Then** the system returns 204 No Content
* **And** the material is permanently deleted from the database
* **And** all associated URLs are deleted
* **And** all associated tags are removed

2. **Scenario**: Delete DRAFT material

* **Given** a material is in DRAFT status
* **When** the author deletes it
* **Then** the deletion succeeds

3. **Scenario**: Delete PUBLISHED material

* **Given** a material is in PUBLISHED status
* **When** the author deletes it
* **Then** the deletion succeeds
* **And** the material is no longer visible to group members

4. **Scenario**: Delete pinned material

* **Given** a material is pinned
* **When** the author deletes it
* **Then** the deletion succeeds
* **And** the material is removed from the pinned list

5. **Scenario**: Attempt to delete by non-author Lead

* **Given** a Lead of the group (but not the author) is authenticated
* **When** they attempt to delete another Lead's material
* **Then** the system rejects with 403 Forbidden (`NOT_MATERIAL_AUTHOR`)

6. **Scenario**: Deletion by Admin (any material)

* **Given** an Admin is authenticated
* **When** they delete any material (regardless of authorship)
* **Then** the system deletes the material successfully

7. **Scenario**: Attempt to delete non-existent material

* **Given** a material ID that doesn't exist
* **When** the author attempts to delete it
* **Then** the system returns 404 Not Found (`MATERIAL_NOT_FOUND`)

---

### Edge Cases

- Delete material that was just created (immediate deletion).
- Concurrent delete requests (idempotent behavior).
- Delete last material in a group.
- Delete material in global group.

---

## API Contract

### Endpoint

```
DELETE /groups/{groupId}/materials/{materialId}
```

### Request Headers

| Header | Required | Description |
|--------|----------|-------------|
| Authorization | Yes | Bearer token for authentication |

### Path Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| groupId | UUID | Yes | The group the material belongs to |
| materialId | UUID | Yes | The material to delete |

### Success Response (204 No Content)

No response body.

### Error Responses

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
  "message": "Only the material author can delete this material",
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
- **FR-004.1**: Other Leads of the group CANNOT delete materials they didn't create.

### Deletion Behavior

- **FR-005**: The system MUST perform a hard delete (permanent removal).
- **FR-006**: The system MUST handle deletion regardless of material status (DRAFT or PUBLISHED).
- **FR-007**: The system MUST handle deletion regardless of pinned state.

### Response

- **FR-008**: The system MUST return 204 No Content on successful deletion.
- **FR-009**: The system MUST return 404 if material doesn't exist.

---

## Non-Functional Requirements

- **NFR-001**: Material deletion MUST complete within 500ms under normal load.
- **NFR-002**: Deletion MUST be atomic (all or nothing).
- **NFR-003**: The system MUST handle concurrent deletion requests gracefully.

---

## Data Model

### Deletion

When a material is deleted:

| Entity | Action |
|--------|--------|
| Material | Hard delete (including embedded tags) |

---

## Security Considerations

- **SEC-001**: Only authenticated users can delete materials.
- **SEC-002**: Only the author or Admin can delete a material.
- **SEC-003**: Other Leads cannot delete materials they didn't create.
- **SEC-004**: Deletion is permanent and cannot be undone.

---

## Optional Notes

- **Soft Delete**: Not implemented - deletion is permanent as per requirements.
- **Audit Log**: Consider logging deletions for audit purposes (who deleted what, when).
- **Confirmation**: UI should implement confirmation dialog before deletion.

