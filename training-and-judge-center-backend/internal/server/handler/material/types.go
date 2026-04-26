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

// materialResponse is used by Create and Update — returns groupId/authorId, no author object.
type materialResponse struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Content     string   `json:"content"`
	Tags        []string `json:"tags"`
	Status      string   `json:"status"`
	Pinned      bool     `json:"pinned"`
	PinnedAt    *string  `json:"pinnedAt"`
	GroupID     string   `json:"groupId"`
	AuthorID    string   `json:"authorId"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
	PublishedAt *string  `json:"publishedAt"`
}

type authorResp struct {
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

// materialDetailResponse is used by Get and List — returns author object, no groupId/authorId.
type materialDetailResponse struct {
	ID          string      `json:"id"`
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
	TotalCount   int `json:"totalCount"`
	CurrentPage  int `json:"currentPage"`
	TotalPages   int `json:"totalPages"`
	ItemsPerPage int `json:"itemsPerPage"`
}

type listMaterialsResponse struct {
	Materials  []materialDetailResponse `json:"materials"`
	Pagination paginationResp           `json:"pagination"`
}

func buildResponse(m appMaterial.MaterialData) materialResponse {
	return materialResponse{
		ID:          m.ID,
		Title:       m.Title,
		Content:     m.Content,
		Tags:        m.Tags,
		Status:      m.Status,
		Pinned:      m.Pinned,
		PinnedAt:    formatTimePtr(m.PinnedAt),
		GroupID:     m.GroupID,
		AuthorID:    m.AuthorID,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339Nano),
		PublishedAt: formatTimePtr(m.PublishedAt),
	}
}

func buildDetailResponse(m appMaterial.MaterialData) materialDetailResponse {
	var author *authorResp
	if m.Author != nil {
		author = &authorResp{Nickname: m.Author.Nickname, Name: m.Author.Name}
	}
	return materialDetailResponse{
		ID:          m.ID,
		Title:       m.Title,
		Content:     m.Content,
		Tags:        m.Tags,
		Status:      m.Status,
		Pinned:      m.Pinned,
		PinnedAt:    formatTimePtr(m.PinnedAt),
		Author:      author,
		CreatedAt:   m.CreatedAt.Format(time.RFC3339Nano),
		UpdatedAt:   m.UpdatedAt.Format(time.RFC3339Nano),
		PublishedAt: formatTimePtr(m.PublishedAt),
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(time.RFC3339Nano)
	return &s
}
