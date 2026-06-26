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

type authorResp struct {
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type materialResponse struct {
	ID          string      `json:"id"`
	GroupID     string      `json:"groupId"`
	Title       string      `json:"title"`
	Content     string      `json:"content"`
	Tags        []string    `json:"tags"`
	Status      string      `json:"status"`
	Pinned      bool        `json:"pinned"`
	PinnedAt    *string     `json:"pinnedAt"`
	Author      *authorResp `json:"author"`
	CreatedAt   string      `json:"createdAt"`
	UpdatedAt   string      `json:"updatedAt"`
	PublishedAt *string     `json:"publishedAt"`
}

type paginationResp struct {
	Page        int  `json:"page"`
	Limit       int  `json:"limit"`
	Total       int  `json:"total"`
	TotalPages  int  `json:"totalPages"`
	HasNextPage bool `json:"hasNextPage"`
	HasPrevPage bool `json:"hasPrevPage"`
}

type listMaterialsResponse struct {
	Materials  []materialResponse `json:"materials"`
	Pagination paginationResp     `json:"pagination"`
}

func buildResponse(m appMaterial.MaterialData) materialResponse {
	var author *authorResp
	if m.Author != nil {
		author = &authorResp{Nickname: m.Author.Nickname, Name: m.Author.Name}
	}
	return materialResponse{
		ID:          m.ID,
		GroupID:     m.GroupID,
		Title:       m.Title,
		Content:     m.Content,
		Tags:        m.Tags,
		Status:      m.Status,
		Pinned:      m.Pinned,
		PinnedAt:    formatTimePtr(m.PinnedAt),
		Author:      author,
		CreatedAt:   m.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:   m.UpdatedAt.UTC().Format(time.RFC3339),
		PublishedAt: formatTimePtr(m.PublishedAt),
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339)
	return &s
}
