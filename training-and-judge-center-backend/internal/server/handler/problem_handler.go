package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type ProblemHandler struct {
	createUC *appProblem.CreateProblemUseCase
	display  user.DisplayProvider
}

func NewProblemHandler(createUC *appProblem.CreateProblemUseCase, display user.DisplayProvider) *ProblemHandler {
	return &ProblemHandler{createUC: createUC, display: display}
}

type createProblemRequest struct {
	Slug          string                    `json:"slug"`
	Title         string                    `json:"title"`
	Statement     *string                   `json:"statement"`
	TimeLimit     *int                      `json:"timeLimit"`
	MemoryLimit   *int                      `json:"memoryLimit"`
	LangOverrides []langOverrideRequest     `json:"languageOverrides"`
	Tags          []string                  `json:"tags"`
}

type langOverrideRequest struct {
	Language    string `json:"language"`
	TimeLimit   *int   `json:"timeLimit"`
	MemoryLimit *int   `json:"memoryLimit"`
}

func (h *ProblemHandler) Create(w http.ResponseWriter, r *http.Request) {
	currentUser := middleware.GetCurrentUser(r.Context())
	if currentUser == nil {
		writeJSON(w, http.StatusUnauthorized, apperror.AppError{
			Code:    "UNAUTHORIZED",
			Message: "Invalid or missing authentication token",
		})
		return
	}

	var body createProblemRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, apperror.AppError{
			Code:    "VALIDATION_ERROR",
			Message: "Invalid request body",
		})
		return
	}

	langOverrides := make([]appProblem.LanguageOverrideInput, 0, len(body.LangOverrides))
	for _, lo := range body.LangOverrides {
		langOverrides = append(langOverrides, appProblem.LanguageOverrideInput{
			Language:    lo.Language,
			TimeLimit:   lo.TimeLimit,
			MemoryLimit: lo.MemoryLimit,
		})
	}

	result, ucErr := h.createUC.Execute(r.Context(), appProblem.CreateProblemInput{
		Slug:          body.Slug,
		Title:         body.Title,
		Statement:     body.Statement,
		TimeLimit:     body.TimeLimit,
		MemoryLimit:   body.MemoryLimit,
		LangOverrides: langOverrides,
		Tags:          body.Tags,
		CurrentUser:   *currentUser,
	})

	if ucErr != nil {
		var appErr *apperror.AppError
		if errors.As(ucErr, &appErr) {
			writeError(w, appErr)
		} else {
			writeError(w, apperror.NewInternal())
		}
		return
	}

	p := result.Problem
	authorDisplay, _ := h.display.GetDisplay(r.Context(), p.AuthorID)

	writeJSON(w, http.StatusCreated, h.buildResponse(p, authorDisplay))
}

type problemResponse struct {
	Slug          string                `json:"slug"`
	Title         string                `json:"title"`
	Statement     *string               `json:"statement"`
	TimeLimit     *int                  `json:"timeLimit"`
	MemoryLimit   *int                  `json:"memoryLimit"`
	LangOverrides []langOverrideResp    `json:"languageOverrides"`
	Tags          []string              `json:"tags"`
	Status        string                `json:"status"`
	Accessibility string                `json:"accessibility"`
	Author        authorResp            `json:"author"`
	Modifiers     []interface{}         `json:"modifiers"`
	Files         filesResp             `json:"files"`
	CreatedAt     string                `json:"createdAt"`
	UpdatedAt     string                `json:"updatedAt"`
}

type langOverrideResp struct {
	Language    string `json:"language"`
	TimeLimit   *int   `json:"timeLimit,omitempty"`
	MemoryLimit *int   `json:"memoryLimit,omitempty"`
}

type authorResp struct {
	Nickname string `json:"nickname"`
	Name     string `json:"name"`
}

type filesResp struct {
	TestCases bool          `json:"testCases"`
	Solutions []interface{} `json:"solutions"`
	Checker   bool          `json:"checker"`
	Validator bool          `json:"validator"`
}

func (h *ProblemHandler) buildResponse(p *problem.Problem, display *user.Display) problemResponse {
	author := authorResp{Nickname: "unknown", Name: ""}
	if display != nil {
		author = authorResp{Nickname: display.Nickname, Name: display.Name}
	}

	overrides := make([]langOverrideResp, 0, len(p.LangOverrides))
	for _, lo := range p.LangOverrides {
		overrides = append(overrides, langOverrideResp{
			Language:    lo.Language(),
			TimeLimit:   lo.TimeLimit(),
			MemoryLimit: lo.MemoryLimit(),
		})
	}

	tags := p.Tags.Values()

	return problemResponse{
		Slug:          p.Slug.String(),
		Title:         p.Title.String(),
		Statement:     p.Statement,
		TimeLimit:     p.TimeLimit,
		MemoryLimit:   p.MemoryLimit,
		LangOverrides: overrides,
		Tags:          tags,
		Status:        p.Status.String(),
		Accessibility: p.Accessibility.String(),
		Author:        author,
		Modifiers:     []interface{}{},
		Files: filesResp{
			TestCases: false,
			Solutions: []interface{}{},
			Checker:   false,
			Validator: false,
		},
		CreatedAt: p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
