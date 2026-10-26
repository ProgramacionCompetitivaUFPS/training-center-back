package problem

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type testCaseSummaryResponse struct {
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	InputBytes  *int64 `json:"inputBytes"`
	AnswerBytes *int64 `json:"answerBytes"`
}

type listTestCasesResponse struct {
	HasTestCases bool                      `json:"hasTestCases"`
	SampleCount  int                       `json:"sampleCount"`
	SecretCount  int                       `json:"secretCount"`
	Cases        []testCaseSummaryResponse `json:"cases"`
}

type testCaseResponse struct {
	Kind            string `json:"kind"`
	Name            string `json:"name"`
	Input           string `json:"input"`
	Answer          string `json:"answer"`
	InputTruncated  bool   `json:"inputTruncated"`
	AnswerTruncated bool   `json:"answerTruncated"`
}

// writeZipDownload sends a ZIP as a download. These ZIPs hold secret test data and solutions,
// so they must never be cached by a browser or a proxy.
func writeZipDownload(w http.ResponseWriter, fileName string, data []byte) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

// @Summary      List the test cases of a problem
// @Description  Names and sizes of the cases in the uploaded ZIP. Only the author, modifiers and Admin.
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {object} listTestCasesResponse
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/testcases [get]
func (h *Handler) ListTestCases(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.listTestCases.Execute(r.Context(), appProblem.ListTestCasesInput{
		Slug:        r.PathValue("slug"),
		CurrentUser: *cu,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	cases := make([]testCaseSummaryResponse, 0, len(out.Cases))
	for _, c := range out.Cases {
		cases = append(cases, testCaseSummaryResponse{Kind: c.Kind, Name: c.Name, InputBytes: c.InputBytes, AnswerBytes: c.AnswerBytes})
	}
	handler.WriteJSON(r.Context(), w, http.StatusOK, listTestCasesResponse{
		HasTestCases: out.HasTestCases,
		SampleCount:  out.SampleCount,
		SecretCount:  out.SecretCount,
		Cases:        cases,
	})
}

// @Summary      Preview one test case
// @Description  The start of the input and answer (64 KiB each; the response says if it was cut). Only the author, modifiers and Admin.
// @Tags         problems
// @Produce      json
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        kind path string true "sample or secret"
// @Param        name path string true "Case name, without extension"
// @Success      200 {object} testCaseResponse
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/testcases/{kind}/{name} [get]
func (h *Handler) GetTestCase(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.getTestCase.Execute(r.Context(), appProblem.GetTestCaseInput{
		Slug:        r.PathValue("slug"),
		Kind:        r.PathValue("kind"),
		Name:        r.PathValue("name"),
		CurrentUser: *cu,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	handler.WriteJSON(r.Context(), w, http.StatusOK, testCaseResponse{
		Kind:            out.Kind,
		Name:            out.Name,
		Input:           out.Input,
		Answer:          out.Answer,
		InputTruncated:  out.InputTruncated,
		AnswerTruncated: out.AnswerTruncated,
	})
}

// @Summary      Download the uploaded test cases ZIP
// @Description  The ZIP exactly as it was uploaded. Only the author, modifiers and Admin.
// @Tags         problems
// @Produce      application/zip
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Success      200 {file} binary
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/files/testcases [get]
func (h *Handler) DownloadTestCases(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	out, err := h.downloadTestCases.Execute(r.Context(), appProblem.DownloadTestCasesInput{
		Slug:        r.PathValue("slug"),
		CurrentUser: *cu,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}
	writeZipDownload(w, out.FileName, out.Data)
}

// @Summary      Read one file of a test case in full
// @Description  The whole input or answer of a case as plain text (the preview endpoint cuts each at 64 KiB). Only the author, modifiers and Admin.
// @Tags         problems
// @Produce      text/plain
// @Security     BearerAuth
// @Param        slug path string true "Problem slug"
// @Param        kind path string true "sample or secret"
// @Param        name path string true "Case name, without extension"
// @Param        part path string true "input or answer"
// @Success      200 {string} string
// @Failure      400 {object} apperror.AppError
// @Failure      401 {object} apperror.AppError
// @Failure      403 {object} apperror.AppError
// @Failure      404 {object} apperror.AppError
// @Router       /problems/p/{slug}/testcases/{kind}/{name}/{part} [get]
func (h *Handler) GetTestCasePart(w http.ResponseWriter, r *http.Request) {
	cu, ok := handler.RequireCurrentUser(w, r)
	if !ok {
		return
	}

	content, err := h.getTestCasePart.Execute(r.Context(), appProblem.GetTestCasePartInput{
		Slug:        r.PathValue("slug"),
		Kind:        r.PathValue("kind"),
		Name:        r.PathValue("name"),
		Part:        r.PathValue("part"),
		CurrentUser: *cu,
	})
	if err != nil {
		handler.WriteError(r.Context(), w, err)
		return
	}

	// Secret test data: never cached, and never sniffed into something executable.
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Length", strconv.Itoa(len(content)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
