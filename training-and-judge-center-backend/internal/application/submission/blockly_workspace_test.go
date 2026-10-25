package submission

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/training-judge-center/backend/internal/testutil"
	domainSubmission "github.com/training-judge-center/backend/internal/domain/submission"
	"github.com/training-judge-center/backend/pkg/apperror"
)

const testWorkspaceXML = `<xml xmlns="https://developers.google.com/blockly/xml"><block type="text_print" id="a" x="10" y="10"></block></xml>`

func blocklyInput() SubmitSolutionInput {
	in := validInput()
	in.Language = "python310"
	in.Compiler = "py"
	in.FileName = "solution.py"
	in.FileData = []byte("print(int(input()) + int(input()))\n")
	in.WorkspaceXML = []byte(testWorkspaceXML)
	return in
}

func requireWorkspaceRejected(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var ae *apperror.AppError
	require.True(t, errors.As(err, &ae))
	assert.Equal(t, apperror.KindBadRequest, ae.Kind)
	assert.Equal(t, ErrCodeInvalidBlocklyWorkspace, ae.Code)
}

func TestValidateBlocklyWorkspace(t *testing.T) {
	cases := []struct {
		name  string
		lang  string
		xml   string
		valid bool
	}{
		{"blockly workspace", "python310", testWorkspaceXML, true},
		{"empty workspace", "python310", `<xml xmlns="https://developers.google.com/blockly/xml"></xml>`, true},
		{"xml declaration and whitespace", "python310", "<?xml version=\"1.0\"?>\n<xml></xml>\n", true},
		{"not python310", "cpp20", testWorkspaceXML, false},
		{"empty body", "python310", "", false},
		{"malformed", "python310", "<xml><block></xml>", false},
		{"wrong root", "python310", "<svg></svg>", false},
		{"two roots", "python310", "<xml></xml><xml></xml>", false},
		{"text outside root", "python310", "<xml></xml>junk", false},
		{"doctype with entities", "python310", `<!DOCTYPE xml [<!ENTITY a "aaaa">]><xml>&a;</xml>`, false},
		{"too large", "python310", "<xml>" + strings.Repeat("a", MaxBlocklyWorkspaceBytes) + "</xml>", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateBlocklyWorkspace(tc.lang, []byte(tc.xml))
			if tc.valid {
				assert.NoError(t, err)
			} else {
				requireWorkspaceRejected(t, err)
			}
		})
	}
}

func TestWorkspaceXMLPathFor(t *testing.T) {
	assert.Equal(t, "p/u/general/id.workspace.xml", workspaceXMLPathFor("p/u/general/id.py"))
}

func TestSubmitSolution_Blockly_StoresWorkspaceNextToSource(t *testing.T) {
	repo := cleanRepo()
	storage := &mockSourceStorage{}
	uc := newUseCase(publicProblem(), repo, storage)

	out, err := uc.Execute(ctx(), blocklyInput())
	require.NoError(t, err)
	assert.True(t, out.IsBlockly)
	assert.Equal(t, "python310", out.Language)

	require.Len(t, storage.uploaded, 2)
	assert.True(t, strings.HasSuffix(storage.uploaded[0], out.ID+".py"))
	assert.True(t, strings.HasSuffix(storage.uploaded[1], out.ID+".workspace.xml"))

	require.NotNil(t, repo.saved)
	require.NotNil(t, repo.saved.WorkspaceXMLPath())
	assert.Equal(t, storage.uploaded[1], *repo.saved.WorkspaceXMLPath())
}

func TestSubmitSolution_RegularSubmission_IsNotBlockly(t *testing.T) {
	repo := cleanRepo()
	storage := &mockSourceStorage{}
	out, err := newUseCase(publicProblem(), repo, storage).Execute(ctx(), validInput())
	require.NoError(t, err)
	assert.False(t, out.IsBlockly)
	assert.Len(t, storage.uploaded, 1)
	assert.Nil(t, repo.saved.WorkspaceXMLPath())
}

func TestSubmitSolution_Blockly_InvalidWorkspace_StoresNothing(t *testing.T) {
	storage := &mockSourceStorage{}
	in := blocklyInput()
	in.WorkspaceXML = []byte("<svg onload=\"alert(1)\"></svg>")

	_, err := newUseCase(publicProblem(), cleanRepo(), storage).Execute(ctx(), in)
	requireWorkspaceRejected(t, err)
	assert.Empty(t, storage.uploaded)
}

func TestSubmitSolution_Blockly_SaveFails_DeletesBothFiles(t *testing.T) {
	storage := &mockSourceStorage{}
	repo := &mockSubmissionRepo{saveErr: apperror.NewInternal()}

	_, err := newUseCase(publicProblem(), repo, storage).Execute(ctx(), blocklyInput())
	require.Error(t, err)
	assert.ElementsMatch(t, storage.uploaded, storage.deleted)
	assert.Len(t, storage.deleted, 2)
}

func TestSubmitContestSolution_Blockly_StoresWorkspace(t *testing.T) {
	repo := cleanRepo()
	storage := &mockSourceStorage{}
	uc := NewSubmitContestSolutionUseCase(
		&mockContestSubmissionProvider{fn: func(_, _ string) (*ContestSubmissionInfo, error) {
			return activeContestInfo(), nil
		}},
		&mockStandingIDResolver{},
		publicProblem(),
		repo,
		storage,
		&mockQueue{},
		maxFileSize,
		rateLimitSeconds,
	)
	in := contestInput()
	in.Language, in.Compiler, in.FileName = "python310", "py", "solution.py"
	in.FileData = []byte("print(1)\n")
	in.WorkspaceXML = []byte(testWorkspaceXML)

	out, err := uc.Execute(ctx(), in)
	require.NoError(t, err)
	assert.True(t, out.IsBlockly)
	require.NotNil(t, repo.saved.WorkspaceXMLPath())
	assert.Contains(t, *repo.saved.WorkspaceXMLPath(), testContestID)
}

func TestGetSubmission_Blockly_ReturnsWorkspace(t *testing.T) {
	sub := aSubmission(testAuthorID, "PRIVATE", nil)
	sub.AttachBlocklyWorkspace("ws/path.workspace.xml")
	repo := &mockSubmissionRepo{findByIDFn: func(_ string) (*domainSubmission.Submission, error) { return sub, nil }}
	reader := &mockSourceCodeReader{fn: func(path string) ([]byte, error) {
		if path == "ws/path.workspace.xml" {
			return []byte(testWorkspaceXML), nil
		}
		return []byte("print(1)"), nil
	}}

	out, err := newGetUseCase(repo, reader, nil, nil, nil, nil, nil).Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	require.NoError(t, err)
	assert.True(t, out.IsBlockly)
	require.NotNil(t, out.WorkspaceXML)
	assert.Equal(t, testWorkspaceXML, *out.WorkspaceXML)
}

func TestGetSubmission_Blockly_MissingWorkspace_StillReturnsDetail(t *testing.T) {
	sub := aSubmission(testAuthorID, "PRIVATE", nil)
	sub.AttachBlocklyWorkspace("ws/gone.workspace.xml")
	repo := &mockSubmissionRepo{findByIDFn: func(_ string) (*domainSubmission.Submission, error) { return sub, nil }}
	reader := &mockSourceCodeReader{fn: func(path string) ([]byte, error) {
		if path == "ws/gone.workspace.xml" {
			return nil, apperror.NewNotFound(ErrCodeSourceCodeNotFound, "gone")
		}
		return []byte("print(1)"), nil
	}}

	out, err := newGetUseCase(repo, reader, nil, nil, nil, nil, nil).Execute(context.Background(), GetSubmissionInput{
		CurrentUser:  testutil.AsContestant(testAuthorID),
		SubmissionID: testSubmissionID,
	})
	require.NoError(t, err)
	assert.True(t, out.IsBlockly)
	assert.Nil(t, out.WorkspaceXML)
	assert.NotNil(t, out.SourceCode)
}
