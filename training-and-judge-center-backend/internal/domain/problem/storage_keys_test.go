package problem_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/shared"
)

func problemWithFiles(testCasesKey *string, solutions []problem.JudgingFile, checker, validator *problem.JudgingFile) *problem.Problem {
	now := time.Now()
	return problem.RestoreProblem(
		"id", "slug", "Title",
		nil, nil, nil, []string{},
		"DRAFT", "PUBLIC",
		shared.RestoreUserID("author"), []shared.UserID{},
		[]problem.LanguageOverride{},
		testCasesKey, solutions, checker, validator,
		nil, now, now,
	)
}

func judgingFile(fileKey string, compiledKey *string) problem.JudgingFile {
	return problem.RestoreJudgingFile("name", fileKey, "cpp20", compiledKey, nil)
}

func strPtr(s string) *string { return &s }

func TestStorageKeys_EmptyProblemOwnsNothing(t *testing.T) {
	keys := problemWithFiles(nil, nil, nil, nil).StorageKeys()

	assert.Empty(t, keys.TestCasesPrefix)
	assert.Empty(t, keys.Files)
}

func TestStorageKeys_ListsEverythingTheProblemOwns(t *testing.T) {
	checker := judgingFile("p/checker/Checker.cpp", strPtr("p/checker/compiled"))
	validator := judgingFile("p/validator/Validator.cpp", strPtr("p/validator/compiled"))
	solutions := []problem.JudgingFile{judgingFile("p/solution/a.cpp", nil), judgingFile("p/solution/b.py", nil)}

	keys := problemWithFiles(strPtr("p/testcases/1"), solutions, &checker, &validator).StorageKeys()

	assert.Equal(t, "p/testcases/1", keys.TestCasesPrefix)
	assert.Equal(t, []string{
		"p/solution/a.cpp", "p/solution/b.py",
		"p/checker/Checker.cpp", "p/checker/compiled",
		"p/validator/Validator.cpp", "p/validator/compiled",
	}, keys.Files)
}

func TestStorageKeys_FileNeverCompiledHasNoBinaryKey(t *testing.T) {
	checker := judgingFile("p/checker/Checker.cpp", nil)

	keys := problemWithFiles(nil, nil, &checker, nil).StorageKeys()

	assert.Equal(t, []string{"p/checker/Checker.cpp"}, keys.Files)
}

func TestStorageKeys_NoLongerIn(t *testing.T) {
	tests := []struct {
		name   string
		before problem.StorageKeys
		after  problem.StorageKeys
		want   problem.StorageKeys
	}{
		{
			name:   "nothing changed",
			before: problem.StorageKeys{TestCasesPrefix: "tc/1", Files: []string{"a", "b"}},
			after:  problem.StorageKeys{TestCasesPrefix: "tc/1", Files: []string{"a", "b"}},
			want:   problem.StorageKeys{},
		},
		{
			name:   "test cases replaced",
			before: problem.StorageKeys{TestCasesPrefix: "tc/1"},
			after:  problem.StorageKeys{TestCasesPrefix: "tc/2"},
			want:   problem.StorageKeys{TestCasesPrefix: "tc/1"},
		},
		{
			name:   "test cases removed",
			before: problem.StorageKeys{TestCasesPrefix: "tc/1"},
			after:  problem.StorageKeys{},
			want:   problem.StorageKeys{TestCasesPrefix: "tc/1"},
		},
		{
			name:   "test cases added leave nothing behind",
			before: problem.StorageKeys{},
			after:  problem.StorageKeys{TestCasesPrefix: "tc/1"},
			want:   problem.StorageKeys{},
		},
		{
			name:   "checker replaced keeps the new source, drops the old binary",
			before: problem.StorageKeys{Files: []string{"checker/Checker.cpp", "checker/compiled"}},
			after:  problem.StorageKeys{Files: []string{"checker/Checker.cpp"}},
			want:   problem.StorageKeys{Files: []string{"checker/compiled"}},
		},
		{
			name:   "file removed",
			before: problem.StorageKeys{Files: []string{"a", "b", "c"}},
			after:  problem.StorageKeys{Files: []string{"a", "c"}},
			want:   problem.StorageKeys{Files: []string{"b"}},
		},
		{
			name:   "everything removed",
			before: problem.StorageKeys{TestCasesPrefix: "tc/1", Files: []string{"a", "b"}},
			after:  problem.StorageKeys{},
			want:   problem.StorageKeys{TestCasesPrefix: "tc/1", Files: []string{"a", "b"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.before.NoLongerIn(tt.after))
		})
	}
}
