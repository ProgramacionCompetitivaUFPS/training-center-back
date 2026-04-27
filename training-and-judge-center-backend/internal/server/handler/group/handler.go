package group

import (
	"net/http"

	appGroup "github.com/training-judge-center/backend/internal/application/group"
	"github.com/training-judge-center/backend/internal/domain/shared"
	"github.com/training-judge-center/backend/internal/server/handler"
	"github.com/training-judge-center/backend/internal/server/middleware"
	"github.com/training-judge-center/backend/pkg/apperror"
)

type Handler struct {
	createGroup     *appGroup.CreateGroupUseCase
	listGroups      *appGroup.ListGroupsUseCase
	getGroup        *appGroup.GetGroupUseCase
	listMyGroups    *appGroup.ListMyGroupsUseCase
	joinGroup       *appGroup.JoinGroupUseCase
	requestJoin     *appGroup.RequestJoinUseCase
	approveRequest  *appGroup.ApproveRequestUseCase
	rejectRequest   *appGroup.RejectRequestUseCase
	listRequests    *appGroup.ListJoinRequestsUseCase
	getMyRequest    *appGroup.GetMyRequestUseCase
	cancelMyRequest *appGroup.CancelMyRequestUseCase
}

func NewHandler(
	createGroup *appGroup.CreateGroupUseCase,
	listGroups *appGroup.ListGroupsUseCase,
	getGroup *appGroup.GetGroupUseCase,
	listMyGroups *appGroup.ListMyGroupsUseCase,
	joinGroup *appGroup.JoinGroupUseCase,
	requestJoin *appGroup.RequestJoinUseCase,
	approveRequest *appGroup.ApproveRequestUseCase,
	rejectRequest *appGroup.RejectRequestUseCase,
	listRequests *appGroup.ListJoinRequestsUseCase,
	getMyRequest *appGroup.GetMyRequestUseCase,
	cancelMyRequest *appGroup.CancelMyRequestUseCase,
) *Handler {
	return &Handler{
		createGroup:     createGroup,
		listGroups:      listGroups,
		getGroup:        getGroup,
		listMyGroups:    listMyGroups,
		joinGroup:       joinGroup,
		requestJoin:     requestJoin,
		approveRequest:  approveRequest,
		rejectRequest:   rejectRequest,
		listRequests:    listRequests,
		getMyRequest:    getMyRequest,
		cancelMyRequest: cancelMyRequest,
	}
}

func (h *Handler) requireCurrentUser(w http.ResponseWriter, r *http.Request) (*shared.CurrentUser, bool) {
	claims := middleware.GetClaims(r.Context())
	if claims == nil {
		handler.WriteError(w, apperror.NewUnauthorized(apperror.ErrCodeUnauthorized, "Invalid or missing authentication token"))
		return nil, false
	}
	u := shared.CurrentUser{ID: claims.UserID, Role: claims.Role.String()}
	return &u, true
}
