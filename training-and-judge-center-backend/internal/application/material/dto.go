package material

import (
	"time"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
)

type MaterialData struct {
	ID          string
	GroupID     string
	AuthorID    string
	Title       string
	Content     string
	Tags        []string
	Status      string
	Pinned      bool
	PinnedAt    *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}

func toMaterialData(m *domainMaterial.Material) MaterialData {
	tags := m.Tags().Values()
	if tags == nil {
		tags = []string{}
	}
	return MaterialData{
		ID:          m.ID(),
		GroupID:     m.GroupID(),
		AuthorID:    m.AuthorID().Value(),
		Title:       m.Title().String(),
		Content:     m.Content().String(),
		Tags:        tags,
		Status:      m.Status().String(),
		Pinned:      m.Pinned(),
		PinnedAt:    m.PinnedAt(),
		CreatedAt:   m.CreatedAt(),
		UpdatedAt:   m.UpdatedAt(),
		PublishedAt: m.PublishedAt(),
	}
}
