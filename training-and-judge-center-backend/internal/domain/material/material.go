package material

import (
	"time"

	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Material struct {
	id          string
	title       Title
	content     Content
	tags        Tags
	status      Status
	pinned      bool
	pinnedAt    *time.Time
	groupID     string
	authorID    shared.UserID
	createdAt   time.Time
	updatedAt   time.Time
	// publishedAt is the timestamp of the first time this material was published.
	// It is never cleared once set, even if the material is unpublished.
	// A non-nil publishedAt means "has ever been published", not "is currently published".
	publishedAt *time.Time
	clock       func() time.Time
}

func (m *Material) now() time.Time { return m.clock().UTC() }

func (m *Material) ID() string              { return m.id }
func (m *Material) Title() Title            { return m.title }
func (m *Material) Content() Content        { return m.content }
func (m *Material) Tags() Tags              { return m.tags }
func (m *Material) Status() Status          { return m.status }
func (m *Material) Pinned() bool            { return m.pinned }
func (m *Material) PinnedAt() *time.Time    { return m.pinnedAt }
func (m *Material) GroupID() string         { return m.groupID }
func (m *Material) AuthorID() shared.UserID { return m.authorID }
func (m *Material) CreatedAt() time.Time    { return m.createdAt }
func (m *Material) UpdatedAt() time.Time    { return m.updatedAt }
func (m *Material) PublishedAt() *time.Time { return m.publishedAt }

// NewMaterial constructs a validated Material. Pass a non-nil clock for deterministic
// timestamps in tests; nil defaults to time.Now.
func NewMaterial(
	id, groupID string,
	authorID shared.UserID,
	title Title,
	content Content,
	tags Tags,
	clock func() time.Time,
) (*Material, error) {
	if id == "" {
		return nil, apperror.NewBadRequest("INVALID_MATERIAL_ID", "material id cannot be empty")
	}
	if groupID == "" {
		return nil, apperror.NewBadRequest("INVALID_GROUP_ID", "group id cannot be empty")
	}
	if authorID.Value() == "" {
		return nil, apperror.NewBadRequest("INVALID_AUTHOR_ID", "material author id cannot be empty")
	}
	if clock == nil {
		clock = time.Now
	}
	now := clock().UTC()
	return &Material{
		clock:     clock,
		id:        id,
		title:     title,
		content:   content,
		tags:      tags,
		status:    NewStatusDraft(),
		pinned:    false,
		groupID:   groupID,
		authorID:  authorID,
		createdAt: now,
		updatedAt: now,
	}, nil
}

func (m *Material) WithClock(fn func() time.Time) *Material {
	m.clock = fn
	return m
}

func RestoreMaterial(
	id, groupID string,
	authorID shared.UserID,
	title, content string,
	tags []string,
	status string,
	pinned bool,
	pinnedAt *time.Time,
	createdAt, updatedAt time.Time,
	publishedAt *time.Time,
) *Material {
	return &Material{
		clock:       time.Now,
		id:          id,
		title:       RestoreTitle(title),
		content:     RestoreContent(content),
		tags:        RestoreTags(tags),
		status:      RestoreStatus(status),
		pinned:      pinned,
		pinnedAt:    pinnedAt,
		groupID:     groupID,
		authorID:    authorID,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		publishedAt: publishedAt,
	}
}

func (m *Material) Publish() error {
	if m.status.IsPublished() {
		return apperror.NewConflict(ErrCodeAlreadyPublished, "material is already published")
	}
	now := m.now()
	m.status = NewStatusPublished()
	m.publishedAt = &now
	m.updatedAt = now
	return nil
}

// Unpublish reverts the material to DRAFT. publishedAt is preserved as a
// first-publish timestamp — (status=DRAFT, publishedAt!=nil) is a valid state
// meaning "was published at least once". List queries must filter by status,
// not by publishedAt, to determine visibility.
func (m *Material) Unpublish() error {
	if m.status.IsDraft() {
		return apperror.NewConflict(ErrCodeAlreadyDraft, "material is already unpublished")
	}
	m.status = NewStatusDraft()
	m.pinned = false
	m.pinnedAt = nil
	m.updatedAt = m.now()
	return nil
}

func (m *Material) Pin() error {
	if m.status.IsDraft() {
		return apperror.NewBadRequest(ErrCodeCannotPinDraft, "cannot pin a draft material, publish it first")
	}
	if m.pinned {
		return nil
	}
	now := m.now()
	m.pinned = true
	m.pinnedAt = &now
	m.updatedAt = now
	return nil
}

func (m *Material) Unpin() {
	if !m.pinned {
		return
	}
	m.pinned = false
	m.pinnedAt = nil
	m.updatedAt = m.now()
}

func (m *Material) UpdateMetadata(title *Title, content *Content, tags *Tags) {
	if title != nil {
		m.title = *title
	}
	if content != nil {
		m.content = *content
	}
	if tags != nil {
		m.tags = *tags
	}
	m.updatedAt = m.now()
}

// Edit rights are tied to authorship, not current group membership — a user
// who created a material retains edit rights even if later demoted or removed.
func (m *Material) CanBeEditedBy(userID shared.UserID, isAdmin bool) bool {
	return isAdmin || m.authorID == userID
}

func (m *Material) CanModifyPinStateBy(userID shared.UserID, isAdmin, isGroupLead bool) bool {
	return isAdmin || isGroupLead || m.authorID == userID
}
