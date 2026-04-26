package material

import (
	"time"

	appMaterial "github.com/training-judge-center/backend/internal/application/material"
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
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	Tags        []string `json:"tags"`
	Status      string  `json:"status"`
	Pinned      bool    `json:"pinned"`
	PinnedAt    *string `json:"pinnedAt"`
	GroupID     string  `json:"groupId"`
	AuthorID    string  `json:"authorId"`
	CreatedAt   string  `json:"createdAt"`
	UpdatedAt   string  `json:"updatedAt"`
	PublishedAt *string `json:"publishedAt"`
}

func buildResponse(m appMaterial.MaterialData) materialResponse {
	var pinnedAt *string
	if m.PinnedAt != nil {
		s := m.PinnedAt.Format(time.RFC3339Nano)
		pinnedAt = &s
	}

	var publishedAt *string
	if m.PublishedAt != nil {
		s := m.PublishedAt.Format(time.RFC3339Nano)
		publishedAt = &s
	}

	return materialResponse{
		ID:          m.ID,
		Title:       m.Title,
		Content:     m.Content,
		Tags:        m.Tags,
		Status:      m.Status,
		Pinned:      m.Pinned,
		PinnedAt:    pinnedAt,
		GroupID:     m.GroupID,
		AuthorID:    m.AuthorID,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339Nano),
		PublishedAt: publishedAt,
	}
}
