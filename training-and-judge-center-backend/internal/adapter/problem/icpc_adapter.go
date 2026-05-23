package problem

import (
	"context"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
)

type ZipParser struct {
	inner *ICPCParser
}

var _ appProblem.ZipParser = (*ZipParser)(nil)

func NewZipParser(inner *ICPCParser) *ZipParser {
	return &ZipParser{inner: inner}
}

func (a *ZipParser) ParseTestCasesZip(ctx context.Context, zipData []byte) ([]appProblem.ParsedFile, error) {
	extracted, err := a.inner.ParseTestCasesZip(ctx, zipData)
	if err != nil {
		return nil, err
	}
	result := make([]appProblem.ParsedFile, len(extracted))
	for i, f := range extracted {
		result[i] = appProblem.ParsedFile{Path: f.Path, Content: f.Content}
	}
	return result, nil
}

type ICPCPackageParser struct {
	inner *ICPCParser
}

var _ appProblem.ICPCPackageParser = (*ICPCPackageParser)(nil)

func NewICPCPackageParser(inner *ICPCParser) *ICPCPackageParser {
	return &ICPCPackageParser{inner: inner}
}

func (a *ICPCPackageParser) ParsePackageZip(ctx context.Context, zipData []byte) (*appProblem.ParsedPackage, error) {
	pkg, err := a.inner.ParsePackageZip(ctx, zipData)
	if err != nil {
		return nil, err
	}

	toAppFiles := func(src []ExtractedFile) []appProblem.ParsedFile {
		out := make([]appProblem.ParsedFile, len(src))
		for i, f := range src {
			out[i] = appProblem.ParsedFile{Path: f.Path, Content: f.Content}
		}
		return out
	}

	toAppFilePtr := func(src *ExtractedFile) *appProblem.ParsedFile {
		if src == nil {
			return nil
		}
		return &appProblem.ParsedFile{Path: src.Path, Content: src.Content}
	}

	return &appProblem.ParsedPackage{
		Title:       pkg.Title,
		TimeLimitMs: pkg.TimeLimitMs,
		MemoryLimit: pkg.MemoryLimit,
		Statement:   pkg.Statement,
		SampleFiles: toAppFiles(pkg.SampleFiles),
		ZipData:     pkg.ZipData,
		Solutions:   toAppFiles(pkg.Solutions),
		Checker:     toAppFilePtr(pkg.Checker),
		Validator:   toAppFilePtr(pkg.Validator),
	}, nil
}
