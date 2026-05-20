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
	createContest *appContest.CreateContestUseCase
	updateContest *appContest.UpdateContestUseCase
	getContest    *appContest.GetContestUseCase
	listContests  *appContest.ListContestsUseCase
}

func NewHandler(
	createContest *appContest.CreateContestUseCase,
	updateContest *appContest.UpdateContestUseCase,
	getContest *appContest.GetContestUseCase,
	listContests *appContest.ListContestsUseCase,
) *Handler {
	return &Handler{
		createContest: createContest,
		updateContest: updateContest,
		getContest:    getContest,
		listContests:  listContests,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(r.Context(), w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role}
	return &u, true
}
