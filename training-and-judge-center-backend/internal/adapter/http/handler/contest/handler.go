package contest

import (
	"net/http"

	"github.com/training-judge-center/backend/internal/adapter/http/handler"
	"github.com/training-judge-center/backend/internal/adapter/http/middleware"
	appContest "github.com/training-judge-center/backend/internal/application/contest"
	"github.com/training-judge-center/backend/internal/application/shared"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	createContest             *appContest.CreateContestUseCase
	updateContest             *appContest.UpdateContestUseCase
	deleteContest             *appContest.DeleteContestUseCase
	getContest                *appContest.GetContestUseCase
	listContests              *appContest.ListContestsUseCase
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
		registerToContest:        registerToContest,
		unregisterFromContest:    unregisterFromContest,
		getRegistrationStatus:    getRegistrationStatus,
		listContestRegistrations: listContestRegistrations,
		getStandings:             getStandings,
		listContestSubmissions:   listContestSubmissions,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	cu, ok := middleware.GetCurrentUser(r.Context())
	if !ok {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	return &cu, true
}
