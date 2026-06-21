package submission

import (
	"net/http"
	"strconv"
	"time"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	appsubmission "github.com/training-judge-center/backend/internal/application/submission"
)

// ── sub-components ────────────────────────────────────────────────────────────

type problemSummary struct {
	Slug  string `json:"slug"`
	Title string `json:"title"`
}

type contestSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type userSummary struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
}

// ── list response ─────────────────────────────────────────────────────────────

type listSubmissionsResponse struct {
	Submissions []submissionSummaryResponse `json:"submissions"`
	Pagination  pagination                  `json:"pagination"`
}

type submissionSummaryResponse struct {
	ID            string          `json:"id"`
	Status        string          `json:"status"`
	Visibility    string          `json:"visibility"`
	SubmittedAt   string          `json:"submittedAt"`
	JudgedAt      *string         `json:"judgedAt"`
	Problem       problemSummary  `json:"problem"`
	Contest       *contestSummary `json:"contest"`
	SubmittedBy   userSummary     `json:"submittedBy"`
	Language      string          `json:"language"`
	ExecutionTime *int            `json:"executionTime"`
	MemoryUsed    *int            `json:"memoryUsed"`
}

type pagination struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"totalPages"`
	HasNext    bool `json:"hasNextPage"`
	HasPrev    bool `json:"hasPrevPage"`
}

// ── shared helpers ────────────────────────────────────────────────────────────

func parsePagination(r *http.Request) (page, limit int) {
	page, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if page <= 0 {
		page = 1
	}
	limit, _ = strconv.Atoi(r.URL.Query().Get("limit"))
	switch {
	case limit <= 0:
		limit = 20
	case limit > 100:
		limit = 100
	}
	return
}

func queryParamPtr(r *http.Request, key string) *string {
	v := r.URL.Query().Get(key)
	if v == "" {
		return nil
	}
	return &v
}

func toSummaryResponse(s appsubmission.SubmissionSummary) submissionSummaryResponse {
	resp := submissionSummaryResponse{
		ID:            s.ID,
		Status:        s.Status,
		Visibility:    s.Visibility,
		SubmittedAt:   s.SubmittedAt.UTC().Format(time.RFC3339),
		Problem:       problemSummary{Slug: s.Problem.Slug, Title: s.Problem.Title},
		SubmittedBy:   userSummary{ID: s.SubmittedBy.ID, Nickname: s.SubmittedBy.Nickname},
		Language:      s.Language,
		ExecutionTime: s.ExecutionTime,
		MemoryUsed:    s.MemoryUsed,
	}
	if s.JudgedAt != nil {
		t := s.JudgedAt.UTC().Format(time.RFC3339)
		resp.JudgedAt = &t
	}
	if s.Contest != nil {
		resp.Contest = &contestSummary{ID: s.Contest.ID, Name: s.Contest.Name}
	}
	return resp
}

func toListResponse(subs []appsubmission.SubmissionSummary, total, page, limit int) listSubmissionsResponse {
	items := make([]submissionSummaryResponse, 0, len(subs))
	for _, s := range subs {
		items = append(items, toSummaryResponse(s))
	}

	totalPages := appshared.CalcTotalPages(total, limit)

	return listSubmissionsResponse{
		Submissions: items,
		Pagination: pagination{
			Page:       page,
			Limit:      limit,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		},
	}
}
