package problem

import (
	"context"

	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
	"github.com/training-judge-center/backend/internal/domain/user"
)

type createProblemUC interface {
	Execute(ctx context.Context, input appProblem.CreateProblemInput) (*appProblem.CreateProblemResult, error)
}

type importProblemUC interface {
	Execute(ctx context.Context, input appProblem.ImportProblemInput) (*appProblem.ImportProblemResult, error)
}

type updateProblemUC interface {
	Execute(ctx context.Context, input appProblem.UpdateProblemInput) (*appProblem.UpdateProblemResult, error)
}

type uploadFilesUC interface {
	Execute(ctx context.Context, input appProblem.UploadProblemFilesInput) (*appProblem.UploadProblemFilesResult, error)
}

type deleteFileUC interface {
	Execute(ctx context.Context, input appProblem.DeleteProblemFileInput) error
}

type addModifierUC interface {
	Execute(ctx context.Context, input appProblem.AddModifierInput) error
}

type removeModifierUC interface {
	Execute(ctx context.Context, input appProblem.RemoveModifierInput) error
}

type listModifiersUC interface {
	Execute(ctx context.Context, slugStr string, currentUser user.CurrentUser) ([]string, error)
}

type getProblemUC interface {
	Execute(ctx context.Context, in appProblem.GetProblemInput) (*appProblem.GetProblemOutput, error)
}

type listProblemsUC interface {
	Execute(ctx context.Context, in appProblem.ListProblemsInput) (*appProblem.ListProblemsOutput, error)
}

type unpublishProblemUC interface {
	Execute(ctx context.Context, in appProblem.UnpublishProblemInput) (*appProblem.UnpublishProblemOutput, error)
}

type changeAccessibilityProblemUC interface {
	Execute(ctx context.Context, in appProblem.ChangeAccessibilityInput) (*appProblem.ChangeAccessibilityOutput, error)
}

type deleteProblemUC interface {
	Execute(ctx context.Context, in appProblem.DeleteProblemInput) error
}

type Handler struct {
	createUC              createProblemUC
	importUC              importProblemUC
	updateUC              updateProblemUC
	uploadUC              uploadFilesUC
	deleteFileUC          deleteFileUC
	addModifierUC         addModifierUC
	removeModifierUC      removeModifierUC
	listModifiersUC       listModifiersUC
	getProblemUC          getProblemUC
	listProblemsUC        listProblemsUC
	unpublishUC           unpublishProblemUC
	changeAccessibilityUC changeAccessibilityProblemUC
	deleteProblemUC       deleteProblemUC
	userProvider          appProblem.UserProvider
	settings              domainProblem.PlatformSettingsService
}

func NewHandler(
	createUC createProblemUC,
	importUC importProblemUC,
	updateUC updateProblemUC,
	uploadUC uploadFilesUC,
	deleteFileUC deleteFileUC,
	addModifierUC addModifierUC,
	removeModifierUC removeModifierUC,
	listModifiersUC listModifiersUC,
	getProblemUC getProblemUC,
	listProblemsUC listProblemsUC,
	unpublishUC unpublishProblemUC,
	changeAccessibilityUC changeAccessibilityProblemUC,
	deleteProblemUC deleteProblemUC,
	userProvider appProblem.UserProvider,
	settings domainProblem.PlatformSettingsService,
) *Handler {
	return &Handler{
		createUC:              createUC,
		importUC:              importUC,
		updateUC:              updateUC,
		uploadUC:              uploadUC,
		deleteFileUC:          deleteFileUC,
		addModifierUC:         addModifierUC,
		removeModifierUC:      removeModifierUC,
		listModifiersUC:       listModifiersUC,
		getProblemUC:          getProblemUC,
		listProblemsUC:        listProblemsUC,
		unpublishUC:           unpublishUC,
		changeAccessibilityUC: changeAccessibilityUC,
		deleteProblemUC:       deleteProblemUC,
		userProvider:          userProvider,
		settings:              settings,
	}
}
