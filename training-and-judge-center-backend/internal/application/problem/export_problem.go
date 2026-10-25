package problem

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"path"

	"gopkg.in/yaml.v3"

	appshared "github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// Layout of the package, the same one ICPCPackageParser reads on import.
const (
	exportYAMLPath      = "problem.yaml"
	exportStatementPath = "problem_statement/problem.en.tex"
	exportSolutionsDir  = "solutions/"
	exportCheckerName   = "checker"
	exportValidatorName = "validator"
)

// exportMetadata is problem.yaml. Time is in seconds (ICPC), memory in MiB, as on import.
type exportMetadata struct {
	Name        string  `yaml:"name"`
	TimeLimit   float64 `yaml:"time_limit,omitempty"`
	MemoryLimit int     `yaml:"memory_limit,omitempty"`
}

type ExportProblemInput struct {
	Slug        string
	CurrentUser appshared.CurrentUser
}

type ExportProblemOutput struct {
	FileName string
	Data     []byte
}

type ExportProblemUseCase struct {
	repo    problem.Repository
	storage ProblemFileRepository
}

func NewExportProblemUseCase(repo problem.Repository, storage ProblemFileRepository) *ExportProblemUseCase {
	return &ExportProblemUseCase{repo: repo, storage: storage}
}

func (uc *ExportProblemUseCase) Execute(ctx context.Context, in ExportProblemInput) (*ExportProblemOutput, error) {
	p, err := loadProblemForFiles(ctx, uc.repo, in.Slug, in.CurrentUser)
	if err != nil {
		return nil, err
	}

	data, err := BuildICPCPackage(ctx, p, uc.storage)
	if err != nil {
		return nil, err
	}
	return &ExportProblemOutput{FileName: p.Slug().String() + ".zip", Data: data}, nil
}

// BuildICPCPackage writes the problem as an ICPC package: the inverse of importing one.
// What the import reads is what gets written — title, limits, statement, the test cases, the
// solutions, the checker and the validator — so a package exported here imports back to the
// same problem. Tags, visibility, modifiers and per-language overrides aren't part of the
// format (the import doesn't read them either).
func BuildICPCPackage(ctx context.Context, p *problem.Problem, storage ProblemFileRepository) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	add := func(name string, content []byte) error {
		w, err := zw.Create(name)
		if err != nil {
			return err
		}
		_, err = w.Write(content)
		return err
	}
	fail := func(err error) ([]byte, error) {
		if _, ok := err.(*apperror.AppError); ok {
			return nil, err
		}
		slog.ErrorContext(ctx, "export: failed to build package", "slug", p.Slug().String(), "error", err)
		return nil, apperror.NewInternal()
	}

	meta := exportMetadata{Name: p.Title().String()}
	if tl := p.TimeLimit(); tl != nil {
		meta.TimeLimit = float64(tl.Milliseconds()) / 1000
	}
	if ml := p.MemoryLimit(); ml != nil {
		meta.MemoryLimit = ml.Megabytes()
	}
	yamlData, err := yaml.Marshal(meta)
	if err != nil {
		return fail(err)
	}
	if err := add(exportYAMLPath, yamlData); err != nil {
		return fail(err)
	}

	if statement := p.Statement().Value(); statement != nil && *statement != "" {
		if err := add(exportStatementPath, []byte(*statement)); err != nil {
			return fail(err)
		}
	}

	if p.TestCasesKey() != nil {
		zipData, err := loadTestCasesZip(ctx, storage, p)
		if err != nil {
			return fail(err)
		}
		archive, err := openTestCaseArchive(ctx, zipData)
		if err != nil {
			return fail(err)
		}
		for _, e := range archive.entries {
			for _, side := range []struct {
				file *zip.File
				ext  string
			}{{e.Input, ".in"}, {e.Answer, ".ans"}} {
				if side.file == nil {
					continue
				}
				content, err := readWholeZipFile(side.file)
				if err != nil {
					return fail(err)
				}
				if err := add(fmt.Sprintf("data/%s/%s%s", e.Kind, e.Name, side.ext), content); err != nil {
					return fail(err)
				}
			}
		}
	}

	for _, sol := range p.Solutions() {
		content, err := storage.DownloadFile(ctx, sol.FileKey())
		if err != nil {
			return fail(missingFileError(err, "Solution "+sol.Filename()))
		}
		if err := add(exportSolutionsDir+sol.Filename(), content); err != nil {
			return fail(err)
		}
	}

	for _, v := range []struct {
		file *problem.JudgingFile
		name string
	}{{p.Checker(), exportCheckerName}, {p.Validator(), exportValidatorName}} {
		if v.file == nil {
			continue
		}
		content, err := storage.DownloadFile(ctx, v.file.FileKey())
		if err != nil {
			return fail(missingFileError(err, "The "+v.name))
		}
		// The import looks for checker/validator + the language extension at the root.
		if err := add(v.name+path.Ext(v.file.FileKey()), content); err != nil {
			return fail(err)
		}
	}

	if err := zw.Close(); err != nil {
		return fail(err)
	}
	return buf.Bytes(), nil
}

// readWholeZipFile reads a case in full. The ZIP was validated against the size limits when
// it was uploaded, so the header's size is a safe bound.
func readWholeZipFile(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(io.LimitReader(rc, int64(f.UncompressedSize64)+1024))
}
