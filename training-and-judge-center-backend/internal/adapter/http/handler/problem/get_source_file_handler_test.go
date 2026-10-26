package problem

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"context"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

// problemWithJudgingFiles is authored by u1 (the user the mock token logs in as).
func problemWithJudgingFiles(author string) *domainProblem.Problem {
	sol := domainProblem.RestoreJudgingFile("sol.cpp", "problems/test-problem/solution/sol.cpp", "cpp20", nil, nil)
	checker := domainProblem.RestoreJudgingFile("chk.py", "problems/test-problem/checker/Checker.py", "python310", nil, nil)
	return domainProblem.RestoreProblem(
		"p1", "test-problem", "Test Problem",
		nil, nil, nil, []string{},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID(author),
		[]shared.UserID{},
		[]domainProblem.LanguageOverride{},
		nil, []domainProblem.JudgingFile{sol},
		&checker, nil, nil,
		testNow, testNow,
	)
}

func sourceStorage() *mockFileStorageH {
	return &mockFileStorageH{downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
		switch path {
		case "problems/test-problem/solution/sol.cpp":
			return []byte("int main(){}"), nil
		case "problems/test-problem/checker/Checker.py":
			return []byte("# checker"), nil
		}
		return nil, nil
	}}
}

func serveSource(h *Handler, fileType, query string) *httptest.ResponseRecorder {
	r := authedRequest(http.MethodGet, "/problems/p/test-problem/files/"+fileType+query, nil)
	r.SetPathValue("slug", "test-problem")
	r.SetPathValue("fileType", fileType)
	w := httptest.NewRecorder()
	wrapAuth(http.HandlerFunc(h.GetSourceFile)).ServeHTTP(w, r)
	return w
}

func handlerForSource(p *domainProblem.Problem) *Handler {
	return &Handler{getSourceFile: appProblem.NewGetSourceFileUseCase(repoReturning(p), sourceStorage())}
}

func TestGetSourceFile_ReturnsTheSolutionAndTheChecker(t *testing.T) {
	h := handlerForSource(problemWithJudgingFiles("u1"))

	w := serveSource(h, "solution", "?fileName=sol.cpp")
	if w.Code != http.StatusOK {
		t.Fatalf("solution: expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var sol sourceFileResponse
	_ = json.NewDecoder(w.Body).Decode(&sol)
	if sol.Content != "int main(){}" || sol.Filename != "sol.cpp" || sol.Language != "cpp20" || sol.Truncated {
		t.Errorf("unexpected solution: %+v", sol)
	}

	w = serveSource(h, "checker", "")
	var chk sourceFileResponse
	_ = json.NewDecoder(w.Body).Decode(&chk)
	if w.Code != http.StatusOK || chk.Content != "# checker" || chk.Language != "python310" {
		t.Errorf("unexpected checker: %d %+v", w.Code, chk)
	}
}

func TestGetSourceFile_ErrorsMapToTheRightStatus(t *testing.T) {
	h := handlerForSource(problemWithJudgingFiles("u1"))
	cases := map[string]struct {
		w    *httptest.ResponseRecorder
		want int
	}{
		"solution without a file name": {serveSource(h, "solution", ""), http.StatusBadRequest},
		"unknown solution":             {serveSource(h, "solution", "?fileName=nope.cpp"), http.StatusNotFound},
		"no validator":                 {serveSource(h, "validator", ""), http.StatusNotFound},
		"bad type":                     {serveSource(h, "exe", ""), http.StatusBadRequest},
		"test cases are not source":    {serveSource(h, "testCases", ""), http.StatusBadRequest},
	}
	for name, tc := range cases {
		if tc.w.Code != tc.want {
			t.Errorf("%s: expected %d, got %d: %s", name, tc.want, tc.w.Code, tc.w.Body.String())
		}
	}
}

func TestGetSourceFile_StrangerGets403WithoutTheCode(t *testing.T) {
	h := handlerForSource(problemWithJudgingFiles("someone-else"))

	w := serveSource(h, "solution", "?fileName=sol.cpp")
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
	if w.Body.Len() > 0 && json.Valid(w.Body.Bytes()) {
		var raw map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &raw)
		if _, leaked := raw["content"]; leaked {
			t.Error("a forbidden response must not carry file content")
		}
	}
}

func TestGetSourceFile_Unauthenticated401(t *testing.T) {
	h := handlerForSource(problemWithJudgingFiles("u1"))
	w := httptest.NewRecorder()
	http.HandlerFunc(h.GetSourceFile).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/x", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestGetSourceFile_FullQueryParamReturnsTheWholeFile(t *testing.T) {
	long := make([]byte, appProblem.SourceFilePreviewMaxBytes+1000)
	for i := range long {
		long[i] = 'a'
	}
	storage := &mockFileStorageH{downloadFileFn: func(_ context.Context, _ string) ([]byte, error) { return long, nil }}
	h := &Handler{getSourceFile: appProblem.NewGetSourceFileUseCase(repoReturning(problemWithJudgingFiles("u1")), storage)}

	cut := serveSource(h, "solution", "?fileName=sol.cpp")
	var cutBody sourceFileResponse
	_ = json.NewDecoder(cut.Body).Decode(&cutBody)
	if !cutBody.Truncated || len(cutBody.Content) != appProblem.SourceFilePreviewMaxBytes {
		t.Errorf("default must stay a capped preview: truncated=%v len=%d", cutBody.Truncated, len(cutBody.Content))
	}

	full := serveSource(h, "solution", "?fileName=sol.cpp&full=true")
	var fullBody sourceFileResponse
	_ = json.NewDecoder(full.Body).Decode(&fullBody)
	if fullBody.Truncated || len(fullBody.Content) != len(long) {
		t.Errorf("full=true must return everything: truncated=%v len=%d want %d", fullBody.Truncated, len(fullBody.Content), len(long))
	}
}
