package material

import (
	"time"

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
	authorID    string
	createdAt   time.Time
	updatedAt   time.Time
	publishedAt *time.Time
	clock       func() time.Time
}

func (m *Material) now() time.Time { return m.clock().UTC() }

func (m *Material) WithClock(fn func() time.Time) *Material {
	m.clock = fn
	return m
}

func (m *Material) ID() string           { return m.id }
func (m *Material) Title() Title         { return m.title }
func (m *Material) Content() Content     { return m.content }
func (m *Material) Tags() Tags           { return m.tags }
func (m *Material) Status() Status       { return m.status }
func (m *Material) Pinned() bool         { return m.pinned }
func (m *Material) PinnedAt() *time.Time { return m.pinnedAt }
func (m *Material) GroupID() string      { return m.groupID }
func (m *Material) AuthorID() string     { return m.authorID }
func (m *Material) CreatedAt() time.Time { return m.createdAt }
func (m *Material) UpdatedAt() time.Time { return m.updatedAt }
func (m *Material) PublishedAt() *time.Time { return m.publishedAt }

func NewMaterial(id, groupID, authorID string, title Title, content Content, tags Tags) *Material {
	now := time.Now().UTC()
	return &Material{
		clock:     time.Now,
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
	}
}

func RestoreMaterial(
	id, groupID, authorID string,
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
		return nil
	}
	m.status = NewStatusPublished()
	if m.publishedAt == nil {
		now := m.now()
		m.publishedAt = &now
	}
	m.updatedAt = m.now()
	return nil
}

func (m *Material) Unpublish() {
	if m.status.IsDraft() {
		return
	}
	m.status = NewStatusDraft()
	m.pinned = false
	m.pinnedAt = nil
	m.updatedAt = m.now()
}

func (m *Material) Pin() error {
	if m.status.IsDraft() {
		return apperror.NewBadRequest(ErrCodeCannotPinDraft, "Cannot pin a draft material. Publish it first.")
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

func (m *Material) UpdateContent(title *Title, content *Content, tags *Tags) {
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

func (m *Material) CanBeEditedBy(userID string, isAdmin bool) bool {
	return isAdmin || m.authorID == userID
}

func (m *Material) CanBePinnedBy(userID string, isAdmin, isGroupLead bool) bool {
	return isAdmin || isGroupLead || m.authorID == userID
}
