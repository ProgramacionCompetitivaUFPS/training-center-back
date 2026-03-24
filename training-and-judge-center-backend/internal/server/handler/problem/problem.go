package problem

import (
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
)

type Handler struct {
	createUC         *appProblem.CreateProblemUseCase
	updateUC         *appProblem.UpdateProblemUseCase
	uploadUC         *appProblem.UploadProblemFilesUseCase
	deleteFileUC     *appProblem.DeleteProblemFileUseCase
	addModifierUC    *appProblem.AddModifierUseCase
	removeModifierUC *appProblem.RemoveModifierUseCase
	listModifiersUC  *appProblem.ListModifiersUseCase
	userProvider     appProblem.UserProvider
	settings         domainProblem.PlatformSettingsService
}

func NewHandler(
	createUC *appProblem.CreateProblemUseCase,
	updateUC *appProblem.UpdateProblemUseCase,
	uploadUC *appProblem.UploadProblemFilesUseCase,
	deleteFileUC *appProblem.DeleteProblemFileUseCase,
	addModifierUC *appProblem.AddModifierUseCase,
	removeModifierUC *appProblem.RemoveModifierUseCase,
	listModifiersUC *appProblem.ListModifiersUseCase,
	userProvider appProblem.UserProvider,
	settings domainProblem.PlatformSettingsService,
) *Handler {
	return &Handler{
		createUC:         createUC,
		updateUC:         updateUC,
		uploadUC:         uploadUC,
		deleteFileUC:     deleteFileUC,
		addModifierUC:    addModifierUC,
		removeModifierUC: removeModifierUC,
		listModifiersUC:  listModifiersUC,
		userProvider:     userProvider,
		settings:         settings,
	}
}
