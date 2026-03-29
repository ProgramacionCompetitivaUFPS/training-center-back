package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/training-judge-center/backend/pkg/apperror"
)

type ICPCParser struct {
	maxUncompressedSizeMB int
	maxFiles              int
	maxSampleFiles        int
}

func NewICPCParser(maxUncompressedSizeMB int, maxFiles int, maxSampleFiles int) *ICPCParser {
	return &ICPCParser{
		maxUncompressedSizeMB: maxUncompressedSizeMB,
		maxFiles:              maxFiles,
		maxSampleFiles:        maxSampleFiles,
	}
}

type ExtractedFile struct {
	Path    string
	Content []byte
}

func (p *ICPCParser) ParseTestCasesZip(zipData []byte) ([]ExtractedFile, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return nil, apperror.NewValidation([]apperror.FieldError{
			{Field: "file", Message: "Invalid ZIP file format"},
		})
	}

	var sampleFiles []*zip.File
	var totalSize int64
	var fileCount int
	var sampleCount int
	var validCount int
	maxLimit := int64(p.maxUncompressedSizeMB) * 1024 * 1024

	hasSampleDir := false
	hasSecretDir := false

	for _, file := range reader.File {
		fileCount++
		if fileCount > p.maxFiles {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Message: "ZIP contains too many files"},
			})
		}

		cleanPath := filepath.ToSlash(filepath.Clean(file.Name))
		if strings.HasPrefix(cleanPath, "../") || filepath.IsAbs(cleanPath) {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Message: "Invalid malicious path detected in ZIP"},
			})
		}

		if file.FileInfo().IsDir() {
			continue
		}

		searchPath := "/" + cleanPath
		isSample := strings.Contains(searchPath, "/data/sample/")
		isSecret := strings.Contains(searchPath, "/data/secret/")
		if isSample {
			hasSampleDir = true
		} else if isSecret {
			hasSecretDir = true
		} else {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Message: fmt.Sprintf("Invalid file or directory outside allowed paths: %s", cleanPath)},
			})
		}

		ext := strings.ToLower(filepath.Ext(cleanPath))
		if ext != ".in" && ext != ".ans" {
			return nil, apperror.NewValidation([]apperror.FieldError{
				{Field: "file", Message: fmt.Sprintf("Invalid file extension found: %s", ext)},
			})
		}

		if file.UncompressedSize64 > 0 {
			totalSize += int64(file.UncompressedSize64)
			if totalSize > maxLimit {
				return nil, apperror.NewValidation([]apperror.FieldError{
					{Field: "file", Message: "Extracted ZIP size exceeds safe limit"},
				})
			}
		}

		validCount++
		if isSample {
			sampleCount++
			if sampleCount > p.maxSampleFiles {
				return nil, apperror.NewValidation([]apperror.FieldError{
					{Field: "file", Message: fmt.Sprintf("Too many sample files: max %d allowed", p.maxSampleFiles)},
				})
			}
			sampleFiles = append(sampleFiles, file)
		}
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

	var extracted []ExtractedFile
	for _, file := range sampleFiles {
		cleanPath := filepath.ToSlash(filepath.Clean(file.Name))

		content, err := p.extractFileContent(file)
		if err != nil {
			slog.Error("failed to extract file from ZIP", "error", err, "file", cleanPath)
			return nil, apperror.NewInternal()
		}

		extracted = append(extracted, ExtractedFile{
			Path:    cleanPath,
			Content: content,
		})
	}

	return extracted, nil
}

func (p *ICPCParser) extractFileContent(file *zip.File) ([]byte, error) {
	rc, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	lr := io.LimitReader(rc, int64(file.UncompressedSize64)+1024)

	content, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}

	return content, nil
}
