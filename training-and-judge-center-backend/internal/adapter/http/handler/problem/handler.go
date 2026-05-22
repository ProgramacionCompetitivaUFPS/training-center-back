package problem

import (
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
)

type Handler struct {
	createUC              *appProblem.CreateProblemUseCase
	importUC              *appProblem.ImportProblemUseCase
	updateUC              *appProblem.UpdateProblemUseCase
	uploadUC              *appProblem.UploadProblemFilesUseCase
	deleteFileUC          *appProblem.DeleteProblemFileUseCase
	addModifierUC         *appProblem.AddModifierUseCase
	removeModifierUC      *appProblem.RemoveModifierUseCase
	listModifiersUC       *appProblem.ListModifiersUseCase
	getProblemUC          *appProblem.GetProblemUseCase
	listProblemsUC        *appProblem.ListProblemsUseCase
	unpublishUC           *appProblem.UnpublishProblemUseCase
	changeAccessibilityUC *appProblem.ChangeAccessibilityUseCase
	deleteProblemUC       *appProblem.DeleteProblemUseCase
	userProvider          appProblem.UserProvider
	settings              domainProblem.PlatformSettings
}

func NewHandler(
	createUC *appProblem.CreateProblemUseCase,
	importUC *appProblem.ImportProblemUseCase,
	updateUC *appProblem.UpdateProblemUseCase,
	uploadUC *appProblem.UploadProblemFilesUseCase,
	deleteFileUC *appProblem.DeleteProblemFileUseCase,
	addModifierUC *appProblem.AddModifierUseCase,
	removeModifierUC *appProblem.RemoveModifierUseCase,
	listModifiersUC *appProblem.ListModifiersUseCase,
	getProblemUC *appProblem.GetProblemUseCase,
	listProblemsUC *appProblem.ListProblemsUseCase,
	unpublishUC *appProblem.UnpublishProblemUseCase,
	changeAccessibilityUC *appProblem.ChangeAccessibilityUseCase,
	deleteProblemUC *appProblem.DeleteProblemUseCase,
	userProvider appProblem.UserProvider,
	settings domainProblem.PlatformSettings,
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
