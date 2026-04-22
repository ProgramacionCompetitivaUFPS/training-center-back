package material

import (
	"testing"
	"time"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func newTestMaterial() *Material {
	fixed := fixedClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	m, err := NewMaterial("id-1", "group-1", "author-1", RestoreTitle("Test"), NewEmptyContent(), RestoreTags(nil), fixed)
	if err != nil {
		panic(err)
	}
	return m
}

func TestPublish(t *testing.T) {
	m := newTestMaterial()

	if err := m.Publish(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Status().IsPublished() {
		t.Error("expected PUBLISHED status")
	}
	if m.PublishedAt() == nil {
		t.Error("expected publishedAt to be set")
	}

	firstPublishedAt := *m.PublishedAt()

	// idempotente — publishedAt no cambia
	if err := m.Publish(); err != nil {
		t.Fatalf("unexpected error on second publish: %v", err)
	}
	if *m.PublishedAt() != firstPublishedAt {
		t.Error("publishedAt should not change on second publish")
	}
}

func TestUnpublish(t *testing.T) {
	m := newTestMaterial()
	_ = m.Publish()

	m.Unpublish()
	if !m.Status().IsDraft() {
		t.Error("expected DRAFT status after unpublish")
	}

	// idempotente
	m.Unpublish()
	if !m.Status().IsDraft() {
		t.Error("expected DRAFT status after second unpublish")
	}
}

func TestUnpublish_AutoUnpin(t *testing.T) {
	m := newTestMaterial()
	_ = m.Publish()
	_ = m.Pin()

	m.Unpublish()
	if m.Pinned() {
		t.Error("expected pinned=false after unpublish")
	}
	if m.PinnedAt() != nil {
		t.Error("expected pinnedAt=nil after unpublish")
	}
}

func TestUnpublish_PreservesPublishedAt(t *testing.T) {
	m := newTestMaterial()
	_ = m.Publish()
	publishedAt := *m.PublishedAt()

	m.Unpublish()
	if m.PublishedAt() == nil || *m.PublishedAt() != publishedAt {
		t.Error("publishedAt should be preserved after unpublish")
	}
}

func TestPin(t *testing.T) {
	m := newTestMaterial()
	_ = m.Publish()

	if err := m.Pin(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Pinned() {
		t.Error("expected pinned=true")
	}
	if m.PinnedAt() == nil {
		t.Error("expected pinnedAt to be set")
	}

	firstPinnedAt := *m.PinnedAt()

	// idempotente — pinnedAt no cambia
	if err := m.Pin(); err != nil {
		t.Fatalf("unexpected error on second pin: %v", err)
	}
	if *m.PinnedAt() != firstPinnedAt {
		t.Error("pinnedAt should not change on second pin")
	}
}

func TestPin_DraftReturnsError(t *testing.T) {
	m := newTestMaterial()

	if err := m.Pin(); err == nil {
		t.Error("expected error when pinning a DRAFT material")
	}
}

func TestUnpin(t *testing.T) {
	m := newTestMaterial()
	_ = m.Publish()
	_ = m.Pin()

	m.Unpin()
	if m.Pinned() {
		t.Error("expected pinned=false after unpin")
	}
	if m.PinnedAt() != nil {
		t.Error("expected pinnedAt=nil after unpin")
	}

	// idempotente
	m.Unpin()
	if m.Pinned() {
		t.Error("expected pinned=false after second unpin")
	}
}

func TestCanBeEditedBy(t *testing.T) {
	m := newTestMaterial()

	if !m.CanBeEditedBy("author-1", false) {
		t.Error("author should be able to edit")
	}
	if !m.CanBeEditedBy("other", true) {
		t.Error("admin should be able to edit")
	}
	if m.CanBeEditedBy("other", false) {
		t.Error("non-author non-admin should not be able to edit")
	}
}

func TestCanBePinnedBy(t *testing.T) {
	m := newTestMaterial()

	if !m.CanBePinnedBy("author-1", false, false) {
		t.Error("author should be able to pin")
	}
	if !m.CanBePinnedBy("other", true, false) {
		t.Error("admin should be able to pin")
	}
	if !m.CanBePinnedBy("other", false, true) {
		t.Error("group lead should be able to pin")
	}
	if m.CanBePinnedBy("other", false, false) {
		t.Error("regular member should not be able to pin")
	}
}
