package problem_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/training-judge-center/backend/internal/adapter/problem"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// memoryStorage is the slice of ProblemFileRepository the export reads.
type memoryStorage struct{ files map[string][]byte }

func (m *memoryStorage) UploadFile(context.Context, string, []byte) error { return nil }
func (m *memoryStorage) DeleteFile(context.Context, string) error         { return nil }
func (m *memoryStorage) DeleteFilesWithPrefix(context.Context, string) error {
	return nil
}
func (m *memoryStorage) ListFiles(context.Context, string) ([]string, error) { return nil, nil }
func (m *memoryStorage) DownloadFile(_ context.Context, path string) ([]byte, error) {
	if c, ok := m.files[path]; ok {
		return c, nil
	}
	return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "file not found")
}

// The export is only worth having if the import reads it back: this feeds what
// BuildICPCPackage writes to the real ICPCParser and checks every part survives.
func TestExportedPackageImportsBackToTheSameProblem(t *testing.T) {
	statement := "Add two numbers: $a + b$.\n\nSegunda línea con tildes: áéíóú."
	timeLimit, memoryLimit := 1500, 256
	key := "problems/sum/testcases/abc"
	sol1 := domainProblem.RestoreJudgingFile("sol.cpp", "problems/sum/solution/sol.cpp", "cpp20", nil, nil)
	sol2 := domainProblem.RestoreJudgingFile("alt.py", "problems/sum/solution/alt.py", "python310", nil, nil)
	checker := domainProblem.RestoreJudgingFile("c.cpp", "problems/sum/checker/Checker.cpp", "cpp20", nil, nil)
	validator := domainProblem.RestoreJudgingFile("v.py", "problems/sum/validator/Validator.py", "python310", nil, nil)
	now := time.Now()

	p := domainProblem.RestoreProblem(
		"id-1", "sum", "Suma: dos números", &statement, &timeLimit, &memoryLimit, []string{},
		"PUBLISHED", "PUBLIC", shared.RestoreUserID("author"), []shared.UserID{},
		[]domainProblem.LanguageOverride{},
		&key, []domainProblem.JudgingFile{sol1, sol2}, &checker, &validator, nil, now, now,
	)

	original := createZipBuffer(t, map[string][]byte{
		"data/sample/1.in":  []byte("1 2\n"),
		"data/sample/1.ans": []byte("3\n"),
		"data/secret/1.in":  []byte("100 200\n"),
		"data/secret/1.ans": []byte("300\n"),
	})
	storage := &memoryStorage{files: map[string][]byte{
		key + "/testcases.zip":                original,
		"problems/sum/solution/sol.cpp":       []byte("int main(){}"),
		"problems/sum/solution/alt.py":        []byte("print(3)"),
		"problems/sum/checker/Checker.cpp":    []byte("// checker"),
		"problems/sum/validator/Validator.py": []byte("# validator"),
	}}

	exported, err := appProblem.BuildICPCPackage(context.Background(), p, storage)
	if err != nil {
		t.Fatalf("BuildICPCPackage: %v", err)
	}

	cfg := testParserConfig()
	cfg.LanguageExtensions = map[string]string{".cpp": "cpp20", ".py": "python310", ".java": "java17"}
	pkg, err := problem.NewICPCParser(cfg).ParsePackageZip(context.Background(), exported)
	if err != nil {
		t.Fatalf("the import rejects what the export wrote: %v", err)
	}

	if pkg.Title != "Suma: dos números" {
		t.Errorf("Title = %q", pkg.Title)
	}
	if pkg.TimeLimitMs == nil || *pkg.TimeLimitMs != 1500 {
		t.Errorf("TimeLimitMs = %v, want 1500", pkg.TimeLimitMs)
	}
	if pkg.MemoryLimit == nil || *pkg.MemoryLimit != 256 {
		t.Errorf("MemoryLimit = %v, want 256", pkg.MemoryLimit)
	}
	if pkg.Statement == nil || *pkg.Statement != statement {
		t.Errorf("Statement = %v, want %q", pkg.Statement, statement)
	}
	if len(pkg.SampleFiles) != 2 {
		t.Errorf("SampleFiles = %d, want the sample .in/.ans pair", len(pkg.SampleFiles))
	}
	if len(pkg.ZipData) == 0 {
		t.Error("the import found no test cases in the exported package")
	}

	var sols []string
	for _, s := range pkg.Solutions {
		sols = append(sols, s.Path)
	}
	sort.Strings(sols)
	if len(sols) != 2 || sols[0] != "solutions/alt.py" || sols[1] != "solutions/sol.cpp" {
		t.Errorf("Solutions = %v", sols)
	}
	if pkg.Checker == nil || string(pkg.Checker.Content) != "// checker" {
		t.Errorf("Checker = %+v", pkg.Checker)
	}
	if pkg.Validator == nil || string(pkg.Validator.Content) != "# validator" {
		t.Errorf("Validator = %+v", pkg.Validator)
	}
}
