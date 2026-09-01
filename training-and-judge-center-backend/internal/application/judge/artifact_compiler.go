package judge

import (
	"context"

	"github.com/training-judge-center/backend/internal/domain/submission"
)

const (
	artifactRoleChecker   = "Checker"
	artifactRoleValidator = "Validator"
)

// ArtifactRole doubles as the artifact's fixed name inside the sandbox.
type ArtifactRole struct {
	value string
}

func NewArtifactRoleChecker() ArtifactRole   { return ArtifactRole{value: artifactRoleChecker} }
func NewArtifactRoleValidator() ArtifactRole { return ArtifactRole{value: artifactRoleValidator} }

func (r ArtifactRole) String() string { return r.value }

type CompileArtifactRequest struct {
	Role       ArtifactRole
	Language   submission.Language
	SourceCode []byte
}

type CompileArtifactResult struct {
	Success  bool
	Log      string
	Artifact []byte // nil if !Success
}

// ArtifactCompiler compiles a problem's checker or validator in the sandbox and
// returns the artifact that gets stored and run later.
type ArtifactCompiler interface {
	Compile(ctx context.Context, req CompileArtifactRequest) (CompileArtifactResult, error)
}
