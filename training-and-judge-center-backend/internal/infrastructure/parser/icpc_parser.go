package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
	"gopkg.in/yaml.v3"
)

type ICPCParser struct {
	maxUncompressedSizeMB int
	maxMetadataFileBytes  int
	maxFiles              int
	maxSampleFiles        int
	languageExtensions    map[string]bool
}

func NewICPCParser(maxUncompressedSizeMB int, maxMetadataFileSizeMB int, maxFiles int, maxSampleFiles int, languageExtensions map[string]string) *ICPCParser {
	exts := make(map[string]bool, len(languageExtensions))
	for ext := range languageExtensions {
		exts[ext] = true
	}
	return &ICPCParser{
		maxUncompressedSizeMB: maxUncompressedSizeMB,
		maxMetadataFileBytes:  maxMetadataFileSizeMB * 1024 * 1024,
		maxFiles:              maxFiles,
		maxSampleFiles:        maxSampleFiles,
		languageExtensions:    exts,
	}
}

type CleanZipFile struct {
	File      *zip.File
	CleanPath string
	Ext       string
	IsDir     bool
}

type ParsedZip struct {
	Reader    *zip.Reader
	FileIndex map[string]*CleanZipFile
	Files     []*CleanZipFile
}

type ExtractedFile struct {
	Path    string
	Content []byte
}

type ParsedPackage struct {
	Title       string
	TimeLimitMs *int
	MemoryLimit *int
	Statement   *string
	SampleFiles []ExtractedFile
	ZipData     []byte
	Solutions   []ExtractedFile
	Checker     *ExtractedFile
	Validator   *ExtractedFile
}

type problemYAML struct {
	Title       string  `yaml:"name"`
	TimeLimit   float64 `yaml:"time_limit"`
	MemoryLimit int     `yaml:"memory_limit"`
}

func (p *ICPCParser) ParseTestCasesZip(zipData []byte) ([]ExtractedFile, error) {
	pz, err := p.prepareZip(zipData)
	if err != nil {
		return nil, err
	}

	prefix, _, err := p.detectRootPrefix(pz, AnchorData)
	if err != nil {
		return nil, err
	}

	if err := p.validateMetadataFileSizes(pz, prefix); err != nil {
		return nil, err
	}

	validCount, sampleFiles, hasSampleDir, hasSecretDir, err := p.parseTestCases(pz, prefix, true)
	if err != nil {
		return nil, err
	}

	if !hasSampleDir && !hasSecretDir {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "file", Message: "ZIP must properly contain data/sample/ and data/secret/ directories"},
		})
	}
	if validCount == 0 {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "file", Message: "ZIP contains no valid testcases"},
		})
	}

	return sampleFiles, nil
}

func (p *ICPCParser) ParsePackageZip(zipData []byte) (*ParsedPackage, error) {
	pz, err := p.prepareZip(zipData)
	if err != nil {
		return nil, err
	}

	prefix, yamlFile, err := p.detectRootPrefix(pz, AnchorYAML)
	if err != nil {
		return nil, err
	}

	if err := p.validateMetadataFileSizes(pz, prefix); err != nil {
		return nil, err
	}

	lookup := func(path string) (*CleanZipFile, bool) {
		f, ok := pz.FileIndex[prefix+path]
		return f, ok
	}

	var errorLogs []string

	yamlContent, err := p.extractFileContent(yamlFile)
	if err != nil {
		return nil, apperror.NewInternal()
	}
	var meta problemYAML
	if err := yaml.Unmarshal(yamlContent, &meta); err != nil {
		return nil, apperror.NewBadRequest(apperror.ErrCodeInvalidPackage, "Failed to parse problem.yaml: "+err.Error())
	}
	if meta.Title == "" {
		errorLogs = append(errorLogs, "problem.yaml is missing required field: name")
	}
	if len(errorLogs) > 0 {
		return nil, invalidPackage(errorLogs)
	}

	pkg := &ParsedPackage{
		Title: meta.Title,
	}

	if meta.TimeLimit > 0 {
		ms := int(math.Round(meta.TimeLimit * 1000))
		pkg.TimeLimitMs = &ms
	}
	if meta.MemoryLimit > 0 {
		ml := meta.MemoryLimit
		pkg.MemoryLimit = &ml
	}

	if texFile, ok := lookup("problem_statement/problem.en.tex"); ok {
		content, err := p.extractFileContent(texFile.File)
		if err != nil {
			return nil, apperror.NewInternal()
		}
		s := string(content)
		pkg.Statement = &s
	}

	validCount, sampleFiles, _, _, err := p.parseTestCases(pz, prefix, false)
	if err != nil {
		return nil, err
	}
	if validCount > 0 {
		pkg.SampleFiles = sampleFiles
		pkg.ZipData = zipData
	}

	var solutions []ExtractedFile
	solPrefix := prefix + "solutions/"
	for _, czf := range pz.Files {
		cleanPath := czf.CleanPath
		if !strings.HasPrefix(cleanPath, solPrefix) || czf.IsDir {
			continue
		}
		if !p.languageExtensions[czf.Ext] {
			continue
		}
		content, err := p.extractFileContent(czf.File)
		if err != nil {
			return nil, apperror.NewInternal()
		}
		solutions = append(solutions, ExtractedFile{Path: cleanPath, Content: content})
	}
	pkg.Solutions = solutions

	checker, err := p.extractRootFile(pz, prefix, "checker")
	if err != nil {
		return nil, err
	}
	pkg.Checker = checker

	validator, err := p.extractRootFile(pz, prefix, "validator")
	if err != nil {
		return nil, err
	}
	pkg.Validator = validator

	return pkg, nil
}

func (p *ICPCParser) parseTestCases(
	pz *ParsedZip,
	prefix string,
	strict bool,
) (validCount int, sampleFiles []ExtractedFile, hasSampleDir bool, hasSecretDir bool, err error) {
	var sampleCount int
	var sampleZipFiles []*zip.File

	for _, czf := range pz.Files {
		stripped := strings.TrimPrefix(czf.CleanPath, prefix)
		searchPath := "/" + stripped
		dir := filepath.ToSlash(filepath.Dir(searchPath))

		isSample := dir == "/data/sample"
		isSecret := dir == "/data/secret"

		if !isSample && !isSecret {
			if strict && !czf.IsDir {
				return 0, nil, false, false, apperror.NewValidation([]apperror.FieldError{
					{Field: "file", Message: fmt.Sprintf("Invalid file or directory outside allowed testcase paths: %s", stripped)},
				})
			}
			continue
		}

		if isSample {
			hasSampleDir = true
		}
		if isSecret {
			hasSecretDir = true
		}

		if czf.IsDir {
			continue
		}

		if czf.Ext != ".in" && czf.Ext != ".ans" {
			return 0, nil, false, false, apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Message: fmt.Sprintf("Invalid file extension found: %s", czf.Ext)},
			})
		}

		validCount++
		if isSample {
			sampleCount++
			if sampleCount > p.maxSampleFiles {
				return 0, nil, false, false, apperror.NewValidation([]apperror.FieldError{
					{Field: "file", Message: fmt.Sprintf("Too many sample files: max %d allowed", p.maxSampleFiles)},
				})
			}
			sampleZipFiles = append(sampleZipFiles, czf.File)
		}
	}

	for _, f := range sampleZipFiles {
		cleanPath := filepath.ToSlash(filepath.Clean(f.Name))
		content, extractErr := p.extractFileContent(f)
		if extractErr != nil {
			slog.Error("failed to extract file from ZIP", "error", extractErr, "file", cleanPath)
			return 0, nil, false, false, apperror.NewInternal()
		}
		sampleFiles = append(sampleFiles, ExtractedFile{Path: cleanPath, Content: content})
	}

	return validCount, sampleFiles, hasSampleDir, hasSecretDir, nil
}

func (p *ICPCParser) extractRootFile(pz *ParsedZip, prefix, name string) (*ExtractedFile, error) {
	var found []*CleanZipFile
	for ext := range p.languageExtensions {
		targetPath := prefix + name + ext
		if czf, ok := pz.FileIndex[targetPath]; ok {
			found = append(found, czf)
		}
	}
	switch len(found) {
	case 0:
		return nil, nil
	case 1:
		content, err := p.extractFileContent(found[0].File)
		if err != nil {
			return nil, apperror.NewInternal()
		}
		return &ExtractedFile{Path: found[0].CleanPath, Content: content}, nil
	default:
		return nil, apperror.NewBadRequest(apperror.ErrCodeInvalidPackage,
			fmt.Sprintf("Multiple %s files found: only one is allowed", name))
	}
}

func (p *ICPCParser) extractFileContent(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	// +1024 buffer to avoid truncating at the boundary due to minor metadata overhead in the ZIP entry
	lr := io.LimitReader(rc, int64(file.UncompressedSize64)+1024)
	return io.ReadAll(lr)
}

type AnchorType string

const (
	AnchorYAML AnchorType = "yaml"
	AnchorData AnchorType = "data"
)

func (p *ICPCParser) detectRootPrefix(pz *ParsedZip, anchor AnchorType) (string, *zip.File, error) {
	seenPrefixes := make(map[string]bool)
	var candidates []string
	var yamlFile *zip.File

	for _, czf := range pz.Files {
		cleanPath := czf.CleanPath
		prefix := ""
		isMatch := false

		if anchor == AnchorYAML {
			if strings.EqualFold(filepath.Base(cleanPath), "problem.yaml") {
				isMatch = true
				yamlFile = czf.File
				if idx := strings.LastIndex(cleanPath, "/"); idx != -1 {
					prefix = cleanPath[:idx+1]
				}
			}
		}

		if anchor == AnchorData {
			searchPath := "/" + cleanPath
			isTestCaseFile := czf.Ext == ".in" || czf.Ext == ".ans"

			if isTestCaseFile {
				dir := filepath.ToSlash(filepath.Dir(searchPath))
				if strings.HasSuffix(dir, "/data/sample") || strings.HasSuffix(dir, "/data/secret") {
					isMatch = true
					idx := strings.Index(searchPath, "/data/")
					prefix = searchPath[1 : idx+1]
				}
			}
		}

		if isMatch {
			if !seenPrefixes[prefix] {
				seenPrefixes[prefix] = true
				candidates = append(candidates, prefix)
			}
		}
	}

	if len(candidates) == 0 {
		msg := "problem.yaml not found (required)"
		if anchor == AnchorData {
			msg = "Valid test case files (.in/.ans) in data/sample/ or data/secret/ not found"
		}
		return "", nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: msg}})
	}

	if len(candidates) > 1 {
		return "", nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: "Ambiguous package: multiple root directories found"}})
	}

	return candidates[0], yamlFile, nil
}

func (p *ICPCParser) validateMetadataFileSizes(pz *ParsedZip, prefix string) error {
	for _, czf := range pz.Files {
		if czf.IsDir {
			continue
		}
		stripped := strings.TrimPrefix(czf.CleanPath, prefix)
		isTestData := strings.HasPrefix("/"+stripped, "/data/sample/") ||
			strings.HasPrefix("/"+stripped, "/data/secret/")
		if !isTestData && czf.File.UncompressedSize64 > uint64(p.maxMetadataFileBytes) {
			return apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Message: fmt.Sprintf("Auxiliary file '%s' exceeds the allowed limit of %d bytes.", czf.CleanPath, p.maxMetadataFileBytes)},
			})
		}
	}
	return nil
}

func invalidPackage(logs []string) error {
	return apperror.NewBadRequest(apperror.ErrCodeInvalidPackage, fmt.Sprintf("Invalid ICPC problem package: %s", strings.Join(logs, "; ")))
}

func (p *ICPCParser) prepareZip(zipData []byte) (*ParsedZip, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: "Invalid ZIP file format"}})
	}

	var totalSize uint64
	var fileCount int
	maxLimit := uint64(p.maxUncompressedSizeMB) * 1024 * 1024

	parsed := &ParsedZip{
		Reader:    reader,
		FileIndex: make(map[string]*CleanZipFile, len(reader.File)),
		Files:     make([]*CleanZipFile, 0, len(reader.File)),
	}

	for _, f := range reader.File {
		cleanPath := filepath.ToSlash(filepath.Clean(f.Name))
		if strings.HasPrefix(cleanPath, "../") || filepath.IsAbs(cleanPath) {
			return nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: "Invalid malicious path detected in ZIP"}})
		}
		if strings.Contains(cleanPath, "__MACOSX") || strings.HasSuffix(cleanPath, ".DS_Store") {
			continue
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: "ZIP contains symlinks which are not allowed"}})
		}

		isDir := f.FileInfo().IsDir()

		if !isDir {
			fileCount++
			if fileCount > p.maxFiles {
				return nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: "ZIP contains too many files"}})
			}
			if f.UncompressedSize64 > maxLimit || totalSize > maxLimit-f.UncompressedSize64 {
				return nil, apperror.NewValidation([]apperror.FieldError{{Field: "file", Message: "Extracted ZIP size exceeds safe limit"}})
			}
			totalSize += f.UncompressedSize64
		}

		czf := &CleanZipFile{
			File:      f,
			CleanPath: cleanPath,
			Ext:       strings.ToLower(filepath.Ext(cleanPath)),
			IsDir:     isDir,
		}
		parsed.FileIndex[cleanPath] = czf
		parsed.Files = append(parsed.Files, czf)
	}
	return parsed, nil
}
