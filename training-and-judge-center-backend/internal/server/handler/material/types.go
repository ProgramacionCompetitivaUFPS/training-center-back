package material

import (
	"time"

	domainMaterial "github.com/training-judge-center/backend/internal/domain/material"
)

type createMaterialRequest struct {
	Title   string   `json:"title"`
	Content *string  `json:"content"`
	Tags    []string `json:"tags"`
}

type updateMaterialRequest struct {
	Title   *string   `json:"title"`
	Content *string   `json:"content"`
	Tags    *[]string `json:"tags"`
}

type materialResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Tags        []string   `json:"tags"`
	Status      string     `json:"status"`
	Pinned      bool       `json:"pinned"`
	PinnedAt    *string    `json:"pinnedAt"`
	GroupID     string     `json:"groupId"`
	AuthorID    string     `json:"authorId"`
	CreatedAt   string     `json:"createdAt"`
	UpdatedAt   string     `json:"updatedAt"`
	PublishedAt *string    `json:"publishedAt"`
}

func buildResponse(m *domainMaterial.Material) materialResponse {
	var pinnedAt *string
	if m.PinnedAt() != nil {
		s := m.PinnedAt().Format(time.RFC3339)
		pinnedAt = &s
	}

	var publishedAt *string
	if m.PublishedAt() != nil {
		s := m.PublishedAt().Format(time.RFC3339)
		publishedAt = &s
	}

	tags := m.Tags().Values()
	if tags == nil {
		tags = []string{}
	}

	return materialResponse{
		ID:          m.ID(),
		Title:       m.Title().String(),
		Content:     m.Content().String(),
		Tags:        tags,
		Status:      m.Status().String(),
		Pinned:      m.Pinned(),
		PinnedAt:    pinnedAt,
		GroupID:     m.GroupID(),
		AuthorID:    m.AuthorID().Value(),
		CreatedAt:   m.CreatedAt().Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt().Format(time.RFC3339),
		PublishedAt: publishedAt,
	}
}
