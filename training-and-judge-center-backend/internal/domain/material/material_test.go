package material_test

import (
	"errors"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/domain/material"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

var testNow = time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)

func newTestMaterial() *material.Material {
	fixed := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	m, err := material.NewMaterial("id-1", "group-1", shared.RestoreUserID("author-1"), material.RestoreTitle("Test"), material.NewEmptyContent(), material.RestoreTags(nil), fixed)
	if err != nil {
		panic(err)
	}
	return m
}

func TestPublish(t *testing.T) {
	m := newTestMaterial()

	if err := m.Publish(testNow); err != nil {
		t.Fatalf("unexpected error on first publish: %v", err)
	}
	if !m.Status().IsPublished() {
		t.Error("expected PUBLISHED status")
	}
	if m.PublishedAt() == nil {
		t.Error("expected publishedAt to be set")
	}

	firstPublishedAt := *m.PublishedAt()

	// publishedAt must not change on subsequent publishes
	if err := m.Publish(testNow); err == nil {
		t.Fatal("expected error when publishing an already published material")
	} else {
		var appErr *apperror.AppError
		if !errors.As(err, &appErr) || appErr.Code != material.ErrCodeAlreadyPublished {
			t.Errorf("expected error code %s, got %v", material.ErrCodeAlreadyPublished, err)
		}
	}
	if *m.PublishedAt() != firstPublishedAt {
		t.Error("publishedAt should not change on second publish")
	}
}

func TestUnpublish(t *testing.T) {
	m := newTestMaterial()
	if err := m.Publish(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := m.Unpublish(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Status().IsDraft() {
		t.Error("expected DRAFT status after unpublish")
	}
}

func TestUnpublish_AlreadyDraftReturnsError(t *testing.T) {
	m := newTestMaterial()

	err := m.Unpublish(testNow)
	if err == nil {
		t.Fatal("expected error when unpublishing a DRAFT material")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != material.ErrCodeAlreadyDraft {
		t.Errorf("expected error code %s, got %v", material.ErrCodeAlreadyDraft, err)
	}
}

func TestUnpublish_AutoUnpin(t *testing.T) {
	m := newTestMaterial()
	if err := m.Publish(testNow); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
	if err := m.Pin(testNow); err != nil {
		t.Fatalf("unexpected pin error: %v", err)
	}

	if err := m.Unpublish(testNow); err != nil {
		t.Fatalf("unexpected unpublish error: %v", err)
	}
	if m.Pinned() {
		t.Error("expected pinned=false after unpublish")
	}
	if m.PinnedAt() != nil {
		t.Error("expected pinnedAt=nil after unpublish")
	}
}

func TestUnpublish_PreservesPublishedAt(t *testing.T) {
	m := newTestMaterial()
	if err := m.Publish(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	publishedAt := *m.PublishedAt()

	if err := m.Unpublish(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m.PublishedAt() == nil || *m.PublishedAt() != publishedAt {
		t.Error("publishedAt should be preserved after unpublish")
	}
}

func TestPin(t *testing.T) {
	m := newTestMaterial()
	if err := m.Publish(testNow); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}

	if err := m.Pin(testNow); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !m.Pinned() {
		t.Error("expected pinned=true")
	}
	if m.PinnedAt() == nil {
		t.Error("expected pinnedAt to be set")
	}

	firstPinnedAt := *m.PinnedAt()

	// pinnedAt must not change when already pinned
	if err := m.Pin(testNow); err != nil {
		t.Fatalf("unexpected error on second pin: %v", err)
	}
	if *m.PinnedAt() != firstPinnedAt {
		t.Error("pinnedAt should not change on second pin")
	}
}

func TestPin_DraftReturnsError(t *testing.T) {
	m := newTestMaterial()

	err := m.Pin(testNow)
	if err == nil {
		t.Fatal("expected error when pinning a DRAFT material")
	}
	var appErr *apperror.AppError
	if !errors.As(err, &appErr) || appErr.Code != material.ErrCodeCannotPinDraft {
		t.Errorf("expected error code %s, got %v", material.ErrCodeCannotPinDraft, err)
	}
}

func TestUnpin(t *testing.T) {
	m := newTestMaterial()
	if err := m.Publish(testNow); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
	if err := m.Pin(testNow); err != nil {
		t.Fatalf("unexpected pin error: %v", err)
	}

	m.Unpin(testNow)
	if m.Pinned() {
		t.Error("expected pinned=false after unpin")
	}
	if m.PinnedAt() != nil {
		t.Error("expected pinnedAt=nil after unpin")
	}

	// second unpin must be a no-op
	m.Unpin(testNow)
	if m.Pinned() {
		t.Error("expected pinned=false after second unpin")
	}
}

func TestUpdateMetadata(t *testing.T) {
	m := newTestMaterial()

	newTitle, _ := material.NewTitle("Updated Title")
	newContent, _ := material.NewContent("Updated content")
	newTags, _ := material.NewTags([]string{"tag1", "tag2"})

	m.UpdateMetadata(&newTitle, &newContent, &newTags, testNow)

	if m.Title().String() != "Updated Title" {
		t.Errorf("expected updated title, got %q", m.Title().String())
	}
	if m.Content().String() != "Updated content" {
		t.Errorf("expected updated content, got %q", m.Content().String())
	}
	if len(m.Tags().Values()) != 2 {
		t.Errorf("expected 2 tags, got %d", len(m.Tags().Values()))
	}
}

func TestUpdateMetadata_PartialUpdate(t *testing.T) {
	m := newTestMaterial()
	originalContent := m.Content().String()

	newTitle, _ := material.NewTitle("Only Title Updated")
	m.UpdateMetadata(&newTitle, nil, nil, testNow)

	if m.Title().String() != "Only Title Updated" {
		t.Errorf("expected updated title, got %q", m.Title().String())
	}
	if m.Content().String() != originalContent {
		t.Error("content should remain unchanged")
	}
}

func TestCanBeEditedBy(t *testing.T) {
	m := newTestMaterial()

	if !m.CanBeEditedBy(shared.RestoreUserID("author-1"), false) {
		t.Error("author should be able to edit")
	}
	if !m.CanBeEditedBy(shared.RestoreUserID("other"), true) {
		t.Error("admin should be able to edit")
	}
	if m.CanBeEditedBy(shared.RestoreUserID("other"), false) {
		t.Error("non-author non-admin should not be able to edit")
	}
}

func TestCanModifyPinStateBy(t *testing.T) {
	m := newTestMaterial()

	if !m.CanModifyPinStateBy(shared.RestoreUserID("author-1"), false, false) {
		t.Error("author should be able to pin")
	}
	if !m.CanModifyPinStateBy(shared.RestoreUserID("other"), true, false) {
		t.Error("admin should be able to pin")
	}
	if !m.CanModifyPinStateBy(shared.RestoreUserID("other"), false, true) {
		t.Error("group lead should be able to pin")
	}
	if m.CanModifyPinStateBy(shared.RestoreUserID("other"), false, false) {
		t.Error("regular member should not be able to pin")
	}
}
