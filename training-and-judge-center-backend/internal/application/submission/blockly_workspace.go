package submission

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
	"strings"

	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

// A submission made with the block editor (Blockly) is a python310 submission
// whose source is the generated code; the workspace XML travels alongside it
// so the blocks can be shown in the detail and reloaded into the editor.
const (
	blocklyLanguage = "python310"

	// MaxBlocklyWorkspaceBytes caps the stored workspace. A workspace for a
	// competitive programming problem is a few KB; this leaves ample room.
	MaxBlocklyWorkspaceBytes = 512 << 10
)

// validateBlocklyWorkspace checks an uploaded workspace before anything is
// stored. The XML is user-controlled and later handed back to browsers, so it
// must be a single well-formed <xml> document with no DOCTYPE/ENTITY
// declarations (entity expansion is the classic way to blow up a parser).
func validateBlocklyWorkspace(language string, data []byte) error {
	if language != blocklyLanguage {
		return apperror.NewBadRequest(ErrCodeInvalidBlocklyWorkspace,
			"a block editor workspace can only be sent with language python310")
	}
	if len(data) > MaxBlocklyWorkspaceBytes {
		return apperror.NewBadRequest(ErrCodeInvalidBlocklyWorkspace,
			fmt.Sprintf("workspace exceeds maximum allowed size of %d bytes", MaxBlocklyWorkspaceBytes))
	}
	if !isBlocklyWorkspaceXML(data) {
		return apperror.NewBadRequest(ErrCodeInvalidBlocklyWorkspace,
			"workspace must be a well-formed Blockly XML document")
	}
	return nil
}

func isBlocklyWorkspaceXML(data []byte) bool {
	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.Strict = true
	root, depth := "", 0
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return root == "xml" && depth == 0
		}
		if err != nil {
			return false
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if depth == 0 {
				if root != "" {
					return false // more than one root element
				}
				root = t.Name.Local
			}
			depth++
		case xml.EndElement:
			depth--
		case xml.Directive:
			return false
		case xml.CharData:
			if depth == 0 && len(bytes.TrimSpace(t)) > 0 {
				return false // text outside the root element
			}
		}
	}
}

// workspaceXMLPathFor stores the workspace next to the source it belongs to:
// "<problem>/<user>/<scope>/<id>.py" → "<problem>/<user>/<scope>/<id>.workspace.xml".
func workspaceXMLPathFor(sourcePath string) string {
	return strings.TrimSuffix(sourcePath, filepath.Ext(sourcePath)) + ".workspace.xml"
}

// uploadSubmissionFiles stores the source and, for block editor submissions,
// the workspace. It returns the workspace path (nil when there is none) and
// leaves nothing behind when it fails.
func uploadSubmissionFiles(ctx context.Context, storage SourceStorage, sourcePath string, source, workspace []byte) (*string, error) {
	if err := storage.Upload(ctx, sourcePath, source); err != nil {
		return nil, err
	}
	if workspace == nil {
		return nil, nil
	}
	wsPath := workspaceXMLPathFor(sourcePath)
	if err := storage.Upload(ctx, wsPath, workspace); err != nil {
		deleteSubmissionFiles(ctx, storage, sourcePath, nil)
		return nil, err
	}
	return &wsPath, nil
}

// deleteSubmissionFiles removes the files of a submission that could not be
// saved; failures are only logged, the save error is what the caller reports.
func deleteSubmissionFiles(ctx context.Context, storage SourceStorage, sourcePath string, workspacePath *string) {
	paths := []string{sourcePath}
	if workspacePath != nil {
		paths = append(paths, *workspacePath)
	}
	for _, p := range paths {
		if err := storage.Delete(ctx, p); err != nil {
			slog.ErrorContext(ctx, "submission: failed to delete orphaned file after save error",
				"path", p, "error", err)
		}
	}
}

// blocklyFilterValue is the pseudo-language list filters accept for block
// editor submissions, which are stored as python310.
const blocklyFilterValue = "blockly"

// languageFilter turns the list's language query into repository filters so
// each submission shows up under exactly one language: "blockly" selects the
// block editor submissions and "python310" leaves them out.
func languageFilter(language *string) (*string, *bool, error) {
	if language == nil {
		return nil, nil, nil
	}
	if *language == blocklyFilterValue {
		yes := true
		return nil, &yes, nil
	}
	if _, err := domainSubmission.NewLanguage(*language); err != nil {
		return nil, nil, err
	}
	if *language == blocklyLanguage {
		no := false
		return language, &no, nil
	}
	return language, nil, nil
}
