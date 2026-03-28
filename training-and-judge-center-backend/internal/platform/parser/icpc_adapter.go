package parser

import (
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	infraParser "github.com/training-judge-center/backend/internal/infrastructure/parser"
)

type ICPCParserAdapter struct {
	inner *infraParser.ICPCParser
}

var _ appProblem.ZipParser = (*ICPCParserAdapter)(nil)

func NewICPCParserAdapter(inner *infraParser.ICPCParser) *ICPCParserAdapter {
	return &ICPCParserAdapter{inner: inner}
}

func (a *ICPCParserAdapter) ParseTestCasesZip(zipData []byte) ([]appProblem.ParsedFile, error) {
	extracted, err := a.inner.ParseTestCasesZip(zipData)
	if err != nil {
		return nil, err
	}
	result := make([]appProblem.ParsedFile, len(extracted))
	for i, f := range extracted {
		result[i] = appProblem.ParsedFile{Path: f.Path, Content: f.Content}
	}
	return result, nil
}
