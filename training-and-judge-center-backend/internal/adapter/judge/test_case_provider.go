package judge

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/jackc/pgx/v5"
	infraPostgres "github.com/training-judge-center/backend/internal/adapter/postgres"
	appJudge "github.com/training-judge-center/backend/internal/application/judge"
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// Defensive read limit per file, mirroring the maxFileSizeTestCaseInputMB that upload enforces.
const maxTestCaseFileBytes = 64 << 20

type TestCaseProvider struct {
	reader gcsReader
	db     infraPostgres.Querier
}

func NewTestCaseProvider(client *storage.Client, bucket string, db infraPostgres.Querier) *TestCaseProvider {
	return &TestCaseProvider{reader: newGCSReader(client, bucket), db: db}
}

func (p *TestCaseProvider) GetTestCases(ctx context.Context, problemID string) ([]appJudge.TestCase, error) {
	q := infraPostgres.GetQuerier(ctx, p.db)

	var testCasesKey *string
	err := q.QueryRow(ctx, `SELECT test_cases_key FROM problems WHERE id = $1`, problemID).Scan(&testCasesKey)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "problem not found")
	}
	if err != nil {
		slog.ErrorContext(ctx, "test_case_provider: query failed", "error", err, "problem_id", problemID)
		return nil, apperror.NewInternal()
	}

	if testCasesKey == nil || *testCasesKey == "" {
		return []appJudge.TestCase{}, nil
	}

	zipKey := appProblem.TestCasesZipKey(*testCasesKey)
	rc, err := p.reader.readObject(ctx, zipKey)
	if err != nil {
		if errors.Is(err, storage.ErrObjectNotExist) {
			slog.ErrorContext(ctx, "test_case_provider: ZIP not found in GCS", "key", zipKey)
			return nil, apperror.NewNotFound(apperror.ErrCodeNotFound, "test cases not found")
		}
		slog.ErrorContext(ctx, "test_case_provider: failed to open ZIP from GCS", "key", zipKey, "error", err)
		return nil, apperror.NewInternal()
	}
	defer rc.Close()

	zipData, err := io.ReadAll(rc)
	if err != nil {
		slog.ErrorContext(ctx, "test_case_provider: failed to read ZIP", "key", zipKey, "error", err)
		return nil, apperror.NewInternal()
	}

	testCases, err := parseTestCasesZip(zipData, maxTestCaseFileBytes)
	if err != nil {
		slog.ErrorContext(ctx, "test_case_provider: failed to parse ZIP", "key", zipKey, "error", err)
		return nil, apperror.NewInternal()
	}
	return testCases, nil
}

// parseTestCasesZip follows the ICPC Problem Package Format (data/sample/ and data/secret/).
func parseTestCasesZip(data []byte, maxFileBytes int) ([]appJudge.TestCase, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid zip: %w", err)
	}

	inputs := make(map[string][]byte)
	answers := make(map[string][]byte)

	for _, f := range r.File {
		cleanPath := filepath.ToSlash(filepath.Clean(f.Name))

		// strings.Index handles optional root-prefix dirs (e.g. "problem-abc/data/secret/001.in").
		// Stripping only "data/" (not "data/sample/"/"data/secret/") keeps the
		// sample/secret group as part of relPath, e.g. "secret/001.in".
		var relPath string
		if idx := strings.Index(cleanPath, "data/sample/"); idx >= 0 {
			relPath = cleanPath[idx+len("data/"):]
		} else if idx := strings.Index(cleanPath, "data/secret/"); idx >= 0 {
			relPath = cleanPath[idx+len("data/"):]
		} else {
			continue
		}

		if f.FileInfo().IsDir() || relPath == "" {
			continue
		}

		ext := strings.ToLower(filepath.Ext(relPath))
		baseName := strings.TrimSuffix(relPath, filepath.Ext(relPath))

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("open %s in zip: %w", cleanPath, err)
		}
		// One byte past the cap: LimitReader on its own would truncate in silence.
		content, readErr := io.ReadAll(io.LimitReader(rc, int64(maxFileBytes)+1))
		rc.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s in zip: %w", cleanPath, readErr)
		}
		if len(content) > maxFileBytes {
			return nil, fmt.Errorf("%s in zip exceeds the %d byte limit", cleanPath, maxFileBytes)
		}

		switch ext {
		case ".in":
			inputs[baseName] = content
		case ".ans":
			answers[baseName] = content
		}
	}

	var names []string
	for name := range inputs {
		if _, ok := answers[name]; ok {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	testCases := make([]appJudge.TestCase, 0, len(names))
	for _, name := range names {
		testCases = append(testCases, appJudge.TestCase{
			Name:           name,
			Input:          inputs[name],
			ExpectedOutput: answers[name],
		})
	}
	return testCases, nil
}
