package problem

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

const handlerCasesZipKey = "problems/test-problem/testcases/xyz/testcases.zip"

func casesZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range [][2]string{
		{"data/sample/1.in", "1 2\n"}, {"data/sample/1.ans", "3\n"},
		{"data/secret/1.in", "5 6\n"}, {"data/secret/1.ans", "11\n"},
	} {
		w, err := zw.Create(e[0])
		if err != nil {
			t.Fatal(err)
		}
		_, _ = w.Write([]byte(e[1]))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func storageWithCases(data []byte) *mockFileStorageH {
	return &mockFileStorageH{downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
		if path == handlerCasesZipKey {
			return data, nil
		}
		return []byte("file"), nil
	}}
}

// someoneElsesProblem has test cases but isn't authored by the user the mock token logs in (u1).
func someoneElsesProblem() *domainProblem.Problem {
	key := "problems/test-problem/testcases/xyz"
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID("someone-else"),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		&key, []domainProblem.JudgingFile{},
		nil, nil, nil,
		testNow, testNow,
	)
}

func handlerFor(p *domainProblem.Problem, storage appProblem.ProblemFileRepository) *Handler {
	repo := repoReturning(p)
	return &Handler{
		listTestCases:     appProblem.NewListTestCasesUseCase(repo, storage),
		getTestCase:       appProblem.NewGetTestCaseUseCase(repo, storage),
		downloadTestCases: appProblem.NewDownloadTestCasesUseCase(repo, storage),
		exportProblem:     appProblem.NewExportProblemUseCase(repo, storage),
		getSourceFile:     appProblem.NewGetSourceFileUseCase(repo, storage),
	}
}

func serve(h http.HandlerFunc, pathValues map[string]string) *httptest.ResponseRecorder {
	r := authedRequest(http.MethodGet, "/x", nil)
	for k, v := range pathValues {
		r.SetPathValue(k, v)
	}
	w := httptest.NewRecorder()
	wrapAuth(h).ServeHTTP(w, r)
	return w
}

var slugOnly = map[string]string{"slug": "test-problem"}

func caseParams(kind, name string) map[string]string {
	return map[string]string{"slug": "test-problem", "kind": kind, "name": name}
}

func TestListTestCases_ReturnsCases(t *testing.T) {
	h := handlerFor(publishedProblemWithTestCases(), storageWithCases(casesZip(t)))

	w := serve(h.ListTestCases, slugOnly)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp listTestCasesResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.HasTestCases || resp.SampleCount != 1 || resp.SecretCount != 1 || len(resp.Cases) != 2 {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if resp.Cases[0].Kind != "sample" || resp.Cases[1].Kind != "secret" {
		t.Errorf("samples must come first: %+v", resp.Cases)
	}
	if resp.Cases[0].InputBytes == nil || *resp.Cases[0].InputBytes != 4 {
		t.Errorf("InputBytes = %v, want 4", resp.Cases[0].InputBytes)
	}
}

func TestListTestCases_WithoutTestCasesIsAnEmptyList(t *testing.T) {
	h := handlerFor(publishedProblem(), &mockFileStorageH{})

	w := serve(h.ListTestCases, slugOnly)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var raw map[string]json.RawMessage
	_ = json.NewDecoder(w.Body).Decode(&raw)
	if string(raw["cases"]) != "[]" {
		t.Errorf(`"cases" must be [] (not null) so clients can map over it, got %s`, raw["cases"])
	}
}

func TestTestCaseEndpoints_StrangerGets403WithoutLeakingData(t *testing.T) {
	h := handlerFor(someoneElsesProblem(), storageWithCases(casesZip(t)))

	responses := map[string]*httptest.ResponseRecorder{
		"list":     serve(h.ListTestCases, slugOnly),
		"preview":  serve(h.GetTestCase, caseParams("secret", "1")),
		"download": serve(h.DownloadTestCases, slugOnly),
		"export":   serve(h.Export, slugOnly),
	}
	for name, w := range responses {
		if w.Code != http.StatusForbidden {
			t.Errorf("%s: expected 403, got %d: %s", name, w.Code, w.Body.String())
		}
		if bytes.Contains(w.Body.Bytes(), []byte("5 6")) {
			t.Errorf("%s: a forbidden response leaked secret test data", name)
		}
	}
}

func TestGetTestCase_ReturnsInputAndAnswer(t *testing.T) {
	h := handlerFor(publishedProblemWithTestCases(), storageWithCases(casesZip(t)))

	w := serve(h.GetTestCase, caseParams("secret", "1"))

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp testCaseResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Input != "5 6\n" || resp.Answer != "11\n" || resp.InputTruncated || resp.AnswerTruncated {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetTestCase_BadKindIs400AndUnknownCaseIs404(t *testing.T) {
	h := handlerFor(publishedProblemWithTestCases(), storageWithCases(casesZip(t)))

	if w := serve(h.GetTestCase, caseParams("hidden", "1")); w.Code != http.StatusBadRequest {
		t.Errorf("bad kind: expected 400, got %d", w.Code)
	}
	if w := serve(h.GetTestCase, caseParams("secret", "99")); w.Code != http.StatusNotFound {
		t.Errorf("unknown case: expected 404, got %d", w.Code)
	}
}

func TestDownloads_AreZipAttachmentsThatNobodyMayCache(t *testing.T) {
	data := casesZip(t)
	h := handlerFor(publishedProblemWithTestCases(), storageWithCases(data))

	cases := []struct {
		name     string
		w        *httptest.ResponseRecorder
		filename string
	}{
		{"testcases", serve(h.DownloadTestCases, slugOnly), "test-problem-testcases.zip"},
		{"export", serve(h.Export, slugOnly), "test-problem.zip"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.w.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", tc.w.Code, tc.w.Body.String())
			}
			if got := tc.w.Header().Get("Content-Type"); got != "application/zip" {
				t.Errorf("Content-Type = %q", got)
			}
			if got := tc.w.Header().Get("Content-Disposition"); got != `attachment; filename="`+tc.filename+`"` {
				t.Errorf("Content-Disposition = %q", got)
			}
			if got := tc.w.Header().Get("Cache-Control"); got != "no-store" {
				t.Errorf("Cache-Control = %q: secret test data must not be cached", got)
			}
			if _, err := zip.NewReader(bytes.NewReader(tc.w.Body.Bytes()), int64(tc.w.Body.Len())); err != nil {
				t.Errorf("body is not a valid ZIP: %v", err)
			}
		})
	}
	if !bytes.Equal(cases[0].w.Body.Bytes(), data) {
		t.Error("the test cases download must be the uploaded ZIP byte for byte")
	}
}

func TestDownloadTestCases_NothingUploadedIs404(t *testing.T) {
	h := handlerFor(publishedProblem(), &mockFileStorageH{})
	if w := serve(h.DownloadTestCases, slugOnly); w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestTestCaseEndpoints_Unauthenticated401(t *testing.T) {
	h := handlerFor(publishedProblemWithTestCases(), storageWithCases(casesZip(t)))
	endpoints := map[string]http.HandlerFunc{
		"list": h.ListTestCases, "preview": h.GetTestCase, "download": h.DownloadTestCases, "export": h.Export,
	}
	for name, fn := range endpoints {
		w := httptest.NewRecorder()
		fn.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
		if w.Code != http.StatusUnauthorized {
			t.Errorf("%s: expected 401, got %d", name, w.Code)
		}
	}
}
