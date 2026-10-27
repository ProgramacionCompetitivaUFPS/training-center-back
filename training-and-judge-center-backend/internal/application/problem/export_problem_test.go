package problem

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

func readZipEntries(t *testing.T, data []byte) map[string]string {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)
	out := make(map[string]string, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		content, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		out[f.Name] = string(content)
	}
	return out
}

func entryNames(entries map[string]string) []string {
	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// completeProblem has every part the package carries; the test cases ZIP sits under a root folder.
func completeProblem(t *testing.T, title string) (*domainProblem.Problem, *mockFileStorage) {
	statement := "Add two numbers: $a + b$."
	timeLimit, memoryLimit := 1500, 256
	key := "problems/test-problem/testcases/xyz"
	sol1 := domainProblem.RestoreJudgingFile("sol.cpp", "problems/test-problem/solution/sol.cpp", "cpp20", nil, nil)
	sol2 := domainProblem.RestoreJudgingFile("alt.py", "problems/test-problem/solution/alt.py", "python310", nil, nil)
	checker := domainProblem.RestoreJudgingFile("my_checker.cpp", "problems/test-problem/checker/Checker.cpp", "cpp20", nil, nil)
	validator := domainProblem.RestoreJudgingFile("val.py", "problems/test-problem/validator/Validator.py", "python310", nil, nil)

	p := domainProblem.RestoreProblem(
		testProbID, testSlug, title,
		&statement, &timeLimit, &memoryLimit, []string{"dp"},
		"PUBLISHED", "PUBLIC",
		shared.RestoreUserID(authorID), []shared.UserID{},
		[]domainProblem.LanguageOverride{},
		&key, []domainProblem.JudgingFile{sol1, sol2}, &checker, &validator,
		nil, testNow, testNow,
	)

	cases := buildZip(t, [2]string{"pkg/data/sample/1.in", "1 2\n"}, [2]string{"pkg/data/sample/1.ans", "3\n"},
		[2]string{"pkg/data/secret/a.in", "5 6\n"}, [2]string{"pkg/data/secret/a.ans", "11\n"})
	files := map[string]string{
		casesZipKey:                                    string(cases),
		"problems/test-problem/solution/sol.cpp":       "int main(){}",
		"problems/test-problem/solution/alt.py":        "print(1)",
		"problems/test-problem/checker/Checker.cpp":    "// checker",
		"problems/test-problem/validator/Validator.py": "# validator",
	}
	return p, &mockFileStorage{downloadFileFn: func(_ context.Context, path string) ([]byte, error) {
		if c, ok := files[path]; ok {
			return []byte(c), nil
		}
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "file not found")
	}}
}

func TestExportProblem_WritesTheLayoutTheImportReads(t *testing.T) {
	p, storage := completeProblem(t, "Suma de Dos")
	out, err := NewExportProblemUseCase(repoWith(p), storage).
		Execute(context.Background(), ExportProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)
	assert.Equal(t, "test-problem.zip", out.FileName)

	entries := readZipEntries(t, out.Data)
	assert.Equal(t, []string{
		"checker.cpp",
		"data/sample/1.ans", "data/sample/1.in",
		"data/secret/a.ans", "data/secret/a.in",
		"problem.yaml",
		"problem_statement/problem.en.tex",
		"solutions/alt.py", "solutions/sol.cpp",
		"validator.py",
	}, entryNames(entries), "root folder of the stored cases ZIP is dropped; checker/validator get the names the import looks for")

	var meta struct {
		Name        string  `yaml:"name"`
		TimeLimit   float64 `yaml:"time_limit"`
		MemoryLimit int     `yaml:"memory_limit"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(entries["problem.yaml"]), &meta))
	assert.Equal(t, "Suma de Dos", meta.Name)
	assert.Equal(t, 1.5, meta.TimeLimit, "milliseconds on the platform, seconds in ICPC")
	assert.Equal(t, 256, meta.MemoryLimit)

	assert.Equal(t, "Add two numbers: $a + b$.", entries["problem_statement/problem.en.tex"])
	assert.Equal(t, "5 6\n", entries["data/secret/a.in"])
	assert.Equal(t, "int main(){}", entries["solutions/sol.cpp"])
	assert.Equal(t, "// checker", entries["checker.cpp"])
	assert.Equal(t, "# validator", entries["validator.py"])
}

func TestExportProblem_TitleWithYAMLSpecialCharactersSurvives(t *testing.T) {
	for _, title := range []string{`Sumas: "rápidas" # 1`, "A: B", "- guion", "yes", "123", "línea\ndos"} {
		t.Run(title, func(t *testing.T) {
			p, storage := completeProblem(t, title)
			out, err := NewExportProblemUseCase(repoWith(p), storage).
				Execute(context.Background(), ExportProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
			require.NoError(t, err)

			var meta struct {
				Name string `yaml:"name"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(readZipEntries(t, out.Data)["problem.yaml"]), &meta))
			assert.Equal(t, title, meta.Name)
		})
	}
}

func TestExportProblem_BareDraftIsJustTheMetadata(t *testing.T) {
	out, err := NewExportProblemUseCase(repoWith(newDraftProblem()), &mockFileStorage{}).
		Execute(context.Background(), ExportProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	require.NoError(t, err)

	entries := readZipEntries(t, out.Data)
	assert.Equal(t, []string{"problem.yaml"}, entryNames(entries))
	assert.NotContains(t, entries["problem.yaml"], "time_limit", "an unset limit stays unset instead of being exported as 0")
	assert.NotContains(t, entries["problem.yaml"], "memory_limit")
}

func TestExportProblem_OnlyAuthorModifiersAndAdmin(t *testing.T) {
	p, storage := completeProblem(t, "X")
	uc := NewExportProblemUseCase(repoWith(p), storage)

	_, err := uc.Execute(context.Background(), ExportProblemInput{Slug: testSlug, CurrentUser: asContestant("stranger")})
	ae := appErr(t, err)
	assert.Equal(t, apperror.KindForbidden, ae.Kind)
	assert.Equal(t, ErrCodeInsufficientPermissions, ae.Code)

	_, err = uc.Execute(context.Background(), ExportProblemInput{Slug: testSlug, CurrentUser: asCoach(authorID)})
	assert.NoError(t, err)
}

func TestExportProblem_AFileTheProblemPointsAtButStorageLostFails(t *testing.T) {
	p, storage := completeProblem(t, "X")
	good := storage.downloadFileFn
	storage.downloadFileFn = func(ctx context.Context, path string) ([]byte, error) {
		if path == "problems/test-problem/solution/alt.py" {
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "file not found")
		}
		return good(ctx, path)
	}

	_, err := NewExportProblemUseCase(repoWith(p), storage).
		Execute(context.Background(), ExportProblemInput{Slug: testSlug, CurrentUser: asAdmin(authorID)})
	ae := appErr(t, err)
	assert.Equal(t, ErrCodeProblemFileNotFound, ae.Code)
	assert.Contains(t, ae.Message, "alt.py")
}
