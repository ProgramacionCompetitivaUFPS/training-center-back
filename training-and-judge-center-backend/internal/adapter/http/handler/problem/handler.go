package problem

import (
	appProblem "github.com/training-judge-center/backend/internal/application/problem"
	domainProblem "github.com/training-judge-center/backend/internal/domain/problem"
)

type Handler struct {
	createProblem        *appProblem.CreateProblemUseCase
	importProblem        *appProblem.ImportProblemUseCase
	updateProblem        *appProblem.UpdateProblemUseCase
	uploadProblemFiles   *appProblem.UploadProblemFilesUseCase
	deleteProblemFile    *appProblem.DeleteProblemFileUseCase
	addModifier          *appProblem.AddModifierUseCase
	removeModifier       *appProblem.RemoveModifierUseCase
	listModifiers        *appProblem.ListModifiersUseCase
	getProblem           *appProblem.GetProblemUseCase
	listProblems         *appProblem.ListProblemsUseCase
	unpublishProblem     *appProblem.UnpublishProblemUseCase
	changeAccessibility  *appProblem.ChangeAccessibilityUseCase
	deleteProblem        *appProblem.DeleteProblemUseCase
	getProblemStatistics *appProblem.GetProblemStatisticsUseCase
	rejudgeSubmissions   *appProblem.RejudgeSubmissionsUseCase
	userProvider         appProblem.UserProvider
	settings             domainProblem.PlatformSettings
}

func NewHandler(
	createProblem *appProblem.CreateProblemUseCase,
	importProblem *appProblem.ImportProblemUseCase,
	updateProblem *appProblem.UpdateProblemUseCase,
	uploadProblemFiles *appProblem.UploadProblemFilesUseCase,
	deleteProblemFile *appProblem.DeleteProblemFileUseCase,
	addModifier *appProblem.AddModifierUseCase,
	removeModifier *appProblem.RemoveModifierUseCase,
	listModifiers *appProblem.ListModifiersUseCase,
	getProblem *appProblem.GetProblemUseCase,
	listProblems *appProblem.ListProblemsUseCase,
	unpublishProblem *appProblem.UnpublishProblemUseCase,
	changeAccessibility *appProblem.ChangeAccessibilityUseCase,
	deleteProblem *appProblem.DeleteProblemUseCase,
	getProblemStatistics *appProblem.GetProblemStatisticsUseCase,
	rejudgeSubmissions *appProblem.RejudgeSubmissionsUseCase,
	userProvider appProblem.UserProvider,
	settings domainProblem.PlatformSettings,
) *Handler {
	return &Handler{
		createProblem:        createProblem,
		importProblem:        importProblem,
		updateProblem:        updateProblem,
		uploadProblemFiles:   uploadProblemFiles,
		deleteProblemFile:    deleteProblemFile,
		addModifier:          addModifier,
		removeModifier:       removeModifier,
		listModifiers:        listModifiers,
		getProblem:           getProblem,
		listProblems:         listProblems,
		unpublishProblem:     unpublishProblem,
		changeAccessibility:  changeAccessibility,
		deleteProblem:        deleteProblem,
		getProblemStatistics: getProblemStatistics,
		rejudgeSubmissions:   rejudgeSubmissions,
		userProvider:         userProvider,
		settings:             settings,
	}
}
