package contest

import (
	appContest "github.com/training-judge-center/backend/internal/application/contest"
)

type Handler struct {
	createContest             *appContest.CreateContestUseCase
	updateContest             *appContest.UpdateContestUseCase
	deleteContest             *appContest.DeleteContestUseCase
	getContest                *appContest.GetContestUseCase
	listContests              *appContest.ListContestsUseCase
	listMyContests            *appContest.ListMyContestsUseCase
	registerToContest         *appContest.RegisterToContestUseCase
	unregisterFromContest     *appContest.UnregisterFromContestUseCase
	getRegistrationStatus     *appContest.GetRegistrationStatusUseCase
	listContestRegistrations  *appContest.ListContestRegistrationsUseCase
	getStandings              *appContest.GetStandingsUseCase
	listContestSubmissions    *appContest.ListContestSubmissionsUseCase
}

func NewHandler(
	createContest *appContest.CreateContestUseCase,
	updateContest *appContest.UpdateContestUseCase,
	deleteContest *appContest.DeleteContestUseCase,
	getContest *appContest.GetContestUseCase,
	listContests *appContest.ListContestsUseCase,
	listMyContests *appContest.ListMyContestsUseCase,
	registerToContest *appContest.RegisterToContestUseCase,
	unregisterFromContest *appContest.UnregisterFromContestUseCase,
	getRegistrationStatus *appContest.GetRegistrationStatusUseCase,
	listContestRegistrations *appContest.ListContestRegistrationsUseCase,
	getStandings *appContest.GetStandingsUseCase,
	listContestSubmissions *appContest.ListContestSubmissionsUseCase,
) *Handler {
	return &Handler{
		createContest:            createContest,
		updateContest:            updateContest,
		deleteContest:            deleteContest,
		getContest:               getContest,
		listContests:             listContests,
		listMyContests:           listMyContests,
		registerToContest:        registerToContest,
		unregisterFromContest:    unregisterFromContest,
		getRegistrationStatus:    getRegistrationStatus,
		listContestRegistrations: listContestRegistrations,
		getStandings:             getStandings,
		listContestSubmissions:   listContestSubmissions,
	}
}
